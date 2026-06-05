package gxl

// Parse lexes and parses a GXL expression into an AST.
func Parse(input string) (Node, error) {
	tokens, err := Lex(input)
	if err != nil {
		return nil, err
	}
	p := &parser{tokens: tokens}
	node, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if p.check(TokenRParen) {
		return nil, parseError(CodeUnmatchedParen, p.peek().At, "unmatched right parenthesis")
	}
	if !p.check(TokenEOF) {
		if isComparisonOp(p.peek().Kind) {
			return nil, parseError(CodeChainedComparison, p.peek().At, "chained comparison")
		}
		if p.peek().Kind == TokenIdent && (p.peek().Lexeme == "contains" || p.peek().Lexeme == "in") {
			return nil, parseError(CodeForbiddenSyntax, p.peek().At, "forbidden infix operator %q", p.peek().Lexeme)
		}
		return nil, parseError(CodeUnexpectedToken, p.peek().At, "unexpected token %s", p.peek().Kind)
	}
	return node, nil
}

type parser struct {
	tokens []Token
	pos    int
}

func (p *parser) parseExpression() (Node, error) { return p.parseOr() }

func (p *parser) parseOr() (Node, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.match(TokenOr) {
		op := p.previous()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &BinaryNode{Position: op.At, Op: op.Kind, Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseAnd() (Node, error) {
	left, err := p.parseNot()
	if err != nil {
		return nil, err
	}
	for p.match(TokenAnd) {
		op := p.previous()
		right, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		left = &BinaryNode{Position: op.At, Op: op.Kind, Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseNot() (Node, error) {
	if p.match(TokenNot) {
		op := p.previous()
		expr, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		return &UnaryNode{Position: op.At, Op: op.Kind, Expr: expr}, nil
	}
	return p.parseComparison()
}

func (p *parser) parseComparison() (Node, error) {
	left, err := p.parseAddition()
	if err != nil {
		return nil, err
	}
	if !isComparisonOp(p.peek().Kind) {
		return left, nil
	}
	op := p.advance()
	right, err := p.parseAddition()
	if err != nil {
		return nil, err
	}
	if isComparisonOp(p.peek().Kind) {
		return nil, parseError(CodeChainedComparison, p.peek().At, "chained comparison")
	}
	return &BinaryNode{Position: op.At, Op: op.Kind, Left: left, Right: right}, nil
}

func (p *parser) parseAddition() (Node, error) {
	left, err := p.parseMultiply()
	if err != nil {
		return nil, err
	}
	for p.match(TokenPlus, TokenMinus) {
		op := p.previous()
		right, err := p.parseMultiply()
		if err != nil {
			return nil, err
		}
		left = &BinaryNode{Position: op.At, Op: op.Kind, Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseMultiply() (Node, error) {
	left, err := p.parseUnaryMinus()
	if err != nil {
		return nil, err
	}
	for p.match(TokenStar, TokenSlash, TokenPercent) {
		op := p.previous()
		right, err := p.parseUnaryMinus()
		if err != nil {
			return nil, err
		}
		left = &BinaryNode{Position: op.At, Op: op.Kind, Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseUnaryMinus() (Node, error) {
	if p.match(TokenMinus) {
		op := p.previous()
		expr, err := p.parseUnaryMinus()
		if err != nil {
			return nil, err
		}
		return &UnaryNode{Position: op.At, Op: op.Kind, Expr: expr}, nil
	}
	if p.check(TokenPlus) {
		return nil, parseError(CodeForbiddenSyntax, p.peek().At, "unary plus is forbidden")
	}
	return p.parsePrimary()
}

func (p *parser) parsePrimary() (Node, error) {
	tok := p.peek()
	switch tok.Kind {
	case TokenNumber:
		p.advance()
		return &LiteralNode{Position: tok.At, Kind: LiteralNumber, Lexeme: tok.Lexeme, Value: tok.Value}, nil
	case TokenString:
		p.advance()
		return &LiteralNode{Position: tok.At, Kind: LiteralString, Lexeme: tok.Lexeme, Value: tok.Value}, nil
	case TokenTrue, TokenFalse:
		p.advance()
		return &LiteralNode{Position: tok.At, Kind: LiteralBool, Lexeme: tok.Lexeme, Value: tok.Value}, nil
	case TokenNull:
		p.advance()
		return &LiteralNode{Position: tok.At, Kind: LiteralNull, Lexeme: tok.Lexeme, Value: tok.Value}, nil
	case TokenLParen:
		p.advance()
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if !p.match(TokenRParen) {
			return nil, parseError(CodeUnmatchedParen, tok.At, "unmatched left parenthesis")
		}
		return expr, nil
	case TokenLen:
		return p.parseBuiltinCall("len")
	case TokenNow:
		return p.parseBuiltinCall("now")
	case TokenStr, TokenList, TokenRegex, TokenMath:
		return p.parseNamespaceLike()
	case TokenIdent:
		if p.look(1).Kind == TokenLParen {
			return nil, parseError(CodeForbiddenSyntax, p.look(1).At, "bare function call is forbidden")
		}
		if p.look(1).Kind == TokenDot && p.look(2).Kind == TokenIdent && p.look(3).Kind == TokenLParen {
			return nil, parseError(CodeUnknownNamespace, tok.At, "unknown namespace %q", tok.Lexeme)
		}
		return p.parseGDP()
	case TokenAnd, TokenOr, TokenNot:
		return nil, parseError(CodeKeywordAsIdentifier, tok.At, "keyword %q used as identifier", tok.Lexeme)
	case TokenRParen:
		return nil, parseError(CodeUnmatchedParen, tok.At, "unmatched right parenthesis")
	case TokenEOF:
		return nil, parseError(CodeUnexpectedToken, tok.At, "unexpected end of expression")
	default:
		return nil, parseError(CodeUnexpectedToken, tok.At, "unexpected token %s", tok.Kind)
	}
}

func (p *parser) parseBuiltinCall(name string) (Node, error) {
	start := p.advance()
	if !p.match(TokenLParen) {
		return nil, parseError(CodeUnexpectedToken, p.peek().At, "expected ( after %s", name)
	}
	var args []Node
	if name == "now" {
		if !p.match(TokenRParen) {
			return nil, parseError(CodeUnexpectedToken, p.peek().At, "now() takes no parse-time arguments")
		}
		return &CallNode{Position: start.At, Method: name}, nil
	}
	if p.check(TokenRParen) {
		return nil, parseError(CodeUnexpectedToken, p.peek().At, "expected argument to %s", name)
	}
	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	args = append(args, expr)
	if p.match(TokenComma) {
		return nil, parseError(CodeUnexpectedToken, p.previous().At, "%s expects one argument", name)
	}
	if !p.match(TokenRParen) {
		return nil, parseError(CodeUnmatchedParen, start.At, "unmatched left parenthesis")
	}
	return &CallNode{Position: start.At, Method: name, Args: args}, nil
}

func (p *parser) parseNamespaceLike() (Node, error) {
	start := p.advance()
	if start.Kind == TokenMath {
		return nil, parseError(CodeUnknownNamespace, start.At, "unknown namespace math")
	}
	if !p.match(TokenDot) {
		return nil, parseError(CodeUnexpectedToken, start.At, "expected namespace call")
	}
	methodTok := p.peek()
	if !p.match(TokenIdent) {
		return nil, parseError(CodeUnexpectedToken, methodTok.At, "expected method name")
	}
	if !p.match(TokenLParen) {
		return nil, parseError(CodeUnexpectedToken, start.At, "namespace call requires parentheses")
	}
	namespace := start.Lexeme
	if !knownMethod(namespace, methodTok.Lexeme) {
		return nil, parseError(CodeUnknownMethod, methodTok.At, "unknown method %s.%s", namespace, methodTok.Lexeme)
	}
	args, err := p.parseArgList()
	if err != nil {
		return nil, err
	}
	if !p.match(TokenRParen) {
		return nil, parseError(CodeUnmatchedParen, start.At, "unmatched left parenthesis")
	}
	return &CallNode{Position: start.At, Namespace: namespace, Method: methodTok.Lexeme, Args: args}, nil
}

func (p *parser) parseArgList() ([]Node, error) {
	if p.check(TokenRParen) {
		return nil, nil
	}
	args := []Node{}
	for {
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		args = append(args, expr)
		if !p.match(TokenComma) {
			return args, nil
		}
		if p.check(TokenRParen) {
			return nil, parseError(CodeUnexpectedToken, p.peek().At, "expected expression after comma")
		}
	}
}

func (p *parser) parseGDP() (Node, error) {
	root := p.advance()
	path := &PathNode{Position: root.At, Root: root.Lexeme}
	for {
		if p.match(TokenDot) {
			dot := p.previous()
			seg := p.peek()
			if !p.match(TokenIdent) {
				if isKeyword(p.peek().Kind) {
					return nil, parseError(CodeKeywordAsIdentifier, p.peek().At, "keyword %q used as identifier", p.peek().Lexeme)
				}
				return nil, parseError(CodeUnexpectedToken, dot.At, "expected field name after dot")
			}
			path.Segments = append(path.Segments, PathSegment{Kind: PathSegmentField, Name: seg.Lexeme, Position: seg.At})
			continue
		}
		if p.match(TokenLBracket) {
			open := p.previous()
			if p.check(TokenMinus) {
				return nil, parseError(CodeForbiddenSyntax, p.peek().At, "negative array index")
			}
			idx := p.peek()
			if !p.match(TokenNumber) || !isIntegerLexeme(idx.Lexeme) {
				return nil, parseError(CodeUnexpectedToken, idx.At, "expected array index integer")
			}
			if !p.match(TokenRBracket) {
				return nil, parseError(CodeUnexpectedToken, open.At, "expected ]")
			}
			path.Segments = append(path.Segments, PathSegment{Kind: PathSegmentIndex, Index: idx.Lexeme, Position: idx.At})
			continue
		}
		break
	}
	return path, nil
}

func (p *parser) match(kinds ...TokenKind) bool {
	for _, kind := range kinds {
		if p.check(kind) {
			p.advance()
			return true
		}
	}
	return false
}

func (p *parser) check(kind TokenKind) bool { return p.peek().Kind == kind }
func (p *parser) advance() Token {
	if !p.check(TokenEOF) {
		p.pos++
	}
	return p.tokens[p.pos-1]
}
func (p *parser) peek() Token { return p.look(0) }
func (p *parser) previous() Token { return p.tokens[p.pos-1] }
func (p *parser) look(offset int) Token {
	idx := p.pos + offset
	if idx >= len(p.tokens) {
		return p.tokens[len(p.tokens)-1]
	}
	return p.tokens[idx]
}

func isComparisonOp(kind TokenKind) bool {
	switch kind {
	case TokenEQ, TokenNEQ, TokenLT, TokenGT, TokenLTE, TokenGTE:
		return true
	default:
		return false
	}
}

func isKeyword(kind TokenKind) bool {
	switch kind {
	case TokenTrue, TokenFalse, TokenNull, TokenAnd, TokenOr, TokenNot, TokenLen, TokenNow, TokenStr, TokenList, TokenRegex, TokenMath:
		return true
	default:
		return false
	}
}

func knownMethod(namespace, method string) bool {
	switch namespace {
	case "str":
		switch method {
		case "startsWith", "endsWith", "contains", "toLower", "toUpper", "trim", "length", "trimPrefix", "trimSuffix":
			return true
		}
	case "list":
		switch method {
		case "contains", "indexOf", "length":
			return true
		}
	case "regex":
		return method == "match"
	}
	return false
}

func isIntegerLexeme(s string) bool {
	for _, ch := range s {
		if ch == '.' || ch == 'e' || ch == 'E' {
			return false
		}
	}
	return s != ""
}
