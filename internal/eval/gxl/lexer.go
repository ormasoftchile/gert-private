package gxl

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	CodeUnexpectedToken      = "GXL-PARSE-001"
	CodeUnterminatedString  = "GXL-PARSE-002"
	CodeInvalidNumber       = "GXL-PARSE-003"
	CodeInvalidEscape       = "GXL-PARSE-004"
	CodeUnknownNamespace    = "GXL-PARSE-005"
	CodeUnknownMethod       = "GXL-PARSE-006"
	CodeForbiddenSyntax     = "GXL-PARSE-007"
	CodeUnmatchedParen      = "GXL-PARSE-008"
	CodeChainedComparison   = "GXL-PARSE-009"
	CodeKeywordAsIdentifier = "GXL-PARSE-010"
)

// ParseError is a position-tagged GXL parse diagnostic.
type ParseError struct {
	Code    string
	Message string
	At      Position
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("%s at %d:%d: %s", e.Code, e.At.Line, e.At.Column, e.Message)
}

// ErrorCode returns the stable conformance error code.
func (e *ParseError) ErrorCode() string { return e.Code }

func parseError(code string, pos Position, format string, args ...any) *ParseError {
	return &ParseError{Code: code, At: pos, Message: fmt.Sprintf(format, args...)}
}

// TokenKind identifies a lexical token in GXL.
type TokenKind int

const (
	TokenEOF TokenKind = iota
	TokenIdent
	TokenNumber
	TokenString
	TokenTrue
	TokenFalse
	TokenNull
	TokenAnd
	TokenOr
	TokenNot
	TokenLen
	TokenNow
	TokenStr
	TokenList
	TokenRegex
	TokenMath
	TokenEQ
	TokenNEQ
	TokenLT
	TokenGT
	TokenLTE
	TokenGTE
	TokenPlus
	TokenMinus
	TokenStar
	TokenSlash
	TokenPercent
	TokenLParen
	TokenRParen
	TokenComma
	TokenDot
	TokenLBracket
	TokenRBracket
)

func (k TokenKind) String() string {
	switch k {
	case TokenEOF:
		return "EOF"
	case TokenIdent:
		return "IDENT"
	case TokenNumber:
		return "NUMBER"
	case TokenString:
		return "STRING"
	case TokenTrue:
		return "true"
	case TokenFalse:
		return "false"
	case TokenNull:
		return "null"
	case TokenAnd:
		return "and"
	case TokenOr:
		return "or"
	case TokenNot:
		return "not"
	case TokenLen:
		return "len"
	case TokenNow:
		return "now"
	case TokenStr:
		return "str"
	case TokenList:
		return "list"
	case TokenRegex:
		return "regex"
	case TokenMath:
		return "math"
	case TokenEQ:
		return "=="
	case TokenNEQ:
		return "!="
	case TokenLT:
		return "<"
	case TokenGT:
		return ">"
	case TokenLTE:
		return "<="
	case TokenGTE:
		return ">="
	case TokenPlus:
		return "+"
	case TokenMinus:
		return "-"
	case TokenStar:
		return "*"
	case TokenSlash:
		return "/"
	case TokenPercent:
		return "%"
	case TokenLParen:
		return "("
	case TokenRParen:
		return ")"
	case TokenComma:
		return ","
	case TokenDot:
		return "."
	case TokenLBracket:
		return "["
	case TokenRBracket:
		return "]"
	default:
		return "<unknown>"
	}
}

// Token is one lexed GXL token with its source position.
type Token struct {
	Kind   TokenKind
	Lexeme string
	Value  string
	At     Position
}

type lexer struct {
	input  []rune
	idx    int
	line   int
	column int
}

// Lex tokenizes a GXL expression according to design/gert/grammar/gxl.ebnf.
func Lex(input string) ([]Token, error) {
	l := &lexer{input: []rune(input), line: 1, column: 1}
	tokens := []Token{}
	for {
		if err := l.skipTrivia(); err != nil {
			return nil, err
		}
		start := l.position()
		if l.done() {
			tokens = append(tokens, Token{Kind: TokenEOF, At: start})
			return tokens, nil
		}
		ch := l.peek()
		switch {
		case ch == '\'' || ch == '"':
			tok, err := l.scanString()
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, tok)
		case isDigit(ch):
			tok, err := l.scanNumber()
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, tok)
		case isIdentStart(ch):
			tok, err := l.scanWord()
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, tok)
		default:
			tok, err := l.scanOperator()
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, tok)
		}
	}
}

func (l *lexer) skipTrivia() error {
	for !l.done() {
		switch l.peek() {
		case ' ', '\t', '\r', '\n':
			l.advance()
		case '#':
			for !l.done() && l.peek() != '\n' && l.peek() != '\r' {
				l.advance()
			}
		default:
			return nil
		}
	}
	return nil
}

func (l *lexer) scanString() (Token, error) {
	start := l.position()
	quote := l.advance()
	var value strings.Builder
	for !l.done() {
		ch := l.advance()
		if ch == quote {
			lexeme := string(l.input[positionIndex(l.input, start, l.idx):l.idx])
			return Token{Kind: TokenString, Lexeme: lexeme, Value: value.String(), At: start}, nil
		}
		if ch == '\n' || ch == '\r' {
			return Token{}, parseError(CodeUnterminatedString, start, "unterminated string literal")
		}
		if ch != '\\' {
			value.WriteRune(ch)
			continue
		}
		if l.done() {
			return Token{}, parseError(CodeUnterminatedString, start, "unterminated string literal")
		}
		escPos := l.position()
		esc := l.advance()
		switch esc {
		case '\\', '"', '\'':
			value.WriteRune(esc)
		case 'n':
			value.WriteRune('\n')
		case 'r':
			value.WriteRune('\r')
		case 't':
			value.WriteRune('\t')
		case '0':
			value.WriteRune('\x00')
		case 'u':
			r, err := l.scanHexRune(4, escPos)
			if err != nil {
				return Token{}, err
			}
			value.WriteRune(r)
		case 'U':
			r, err := l.scanHexRune(8, escPos)
			if err != nil {
				return Token{}, err
			}
			value.WriteRune(r)
		default:
			return Token{}, parseError(CodeInvalidEscape, escPos, "invalid escape sequence \\%c", esc)
		}
	}
	return Token{}, parseError(CodeUnterminatedString, start, "unterminated string literal")
}

func (l *lexer) scanHexRune(width int, pos Position) (rune, error) {
	var value rune
	for i := 0; i < width; i++ {
		if l.done() || !isHex(l.peek()) {
			return 0, parseError(CodeInvalidEscape, pos, "invalid unicode escape")
		}
		ch := l.advance()
		value *= 16
		switch {
		case ch >= '0' && ch <= '9':
			value += ch - '0'
		case ch >= 'a' && ch <= 'f':
			value += ch - 'a' + 10
		case ch >= 'A' && ch <= 'F':
			value += ch - 'A' + 10
		}
	}
	if !utf8.ValidRune(value) {
		return 0, parseError(CodeInvalidEscape, pos, "invalid unicode code point")
	}
	return value, nil
}

func (l *lexer) scanNumber() (Token, error) {
	startIdx := l.idx
	start := l.position()
	if l.peek() == '0' {
		l.advance()
		if !l.done() && isDigit(l.peek()) {
			return Token{}, parseError(CodeInvalidNumber, start, "leading zero in number literal")
		}
	} else {
		for !l.done() && isDigit(l.peek()) {
			l.advance()
		}
	}
	if !l.done() && l.peek() == '.' && l.hasNextDigit() {
		l.advance()
		for !l.done() && isDigit(l.peek()) {
			l.advance()
		}
	}
	if !l.done() && (l.peek() == 'e' || l.peek() == 'E') {
		expIdx, expPos := l.idx, l.position()
		l.advance()
		if !l.done() && (l.peek() == '+' || l.peek() == '-') {
			l.advance()
		}
		if l.done() || !isDigit(l.peek()) {
			l.idx = expIdx
			l.line = expPos.Line
			l.column = expPos.Column
		} else {
			for !l.done() && isDigit(l.peek()) {
				l.advance()
			}
		}
	}
	if !l.done() && (isIdentStart(l.peek()) || l.peek() == 'x' || l.peek() == 'o' || l.peek() == 'b') {
		return Token{}, parseError(CodeInvalidNumber, start, "invalid number literal")
	}
	lexeme := string(l.input[startIdx:l.idx])
	return Token{Kind: TokenNumber, Lexeme: lexeme, Value: lexeme, At: start}, nil
}

func (l *lexer) scanWord() (Token, error) {
	startIdx := l.idx
	start := l.position()
	l.advance()
	for !l.done() && isIdentCont(l.peek()) {
		l.advance()
	}
	lexeme := string(l.input[startIdx:l.idx])
	kind := TokenIdent
	switch lexeme {
	case "true":
		kind = TokenTrue
	case "false":
		kind = TokenFalse
	case "null":
		kind = TokenNull
	case "and":
		kind = TokenAnd
	case "or":
		kind = TokenOr
	case "not":
		kind = TokenNot
	case "len":
		kind = TokenLen
	case "now":
		kind = TokenNow
	case "str":
		kind = TokenStr
	case "list":
		kind = TokenList
	case "regex":
		kind = TokenRegex
	case "math":
		kind = TokenMath
	}
	return Token{Kind: kind, Lexeme: lexeme, Value: lexeme, At: start}, nil
}

func (l *lexer) scanOperator() (Token, error) {
	start := l.position()
	ch := l.advance()
	switch ch {
	case '&':
		if l.match('&') {
			return Token{}, parseError(CodeForbiddenSyntax, start, "forbidden token &&")
		}
	case '|':
		if l.match('|') {
			return Token{}, parseError(CodeForbiddenSyntax, start, "forbidden token ||")
		}
		return Token{}, parseError(CodeForbiddenSyntax, start, "forbidden token |")
	case '!':
		if l.match('=') {
			return Token{Kind: TokenNEQ, Lexeme: "!=", At: start}, nil
		}
		return Token{}, parseError(CodeForbiddenSyntax, start, "forbidden token !")
	case '?', ':':
		return Token{}, parseError(CodeForbiddenSyntax, start, "forbidden token %c", ch)
	case '=':
		if l.match('=') {
			return Token{Kind: TokenEQ, Lexeme: "==", At: start}, nil
		}
		return Token{}, parseError(CodeForbiddenSyntax, start, "forbidden assignment")
	case '<':
		if l.match('=') {
			return Token{Kind: TokenLTE, Lexeme: "<=", At: start}, nil
		}
		return Token{Kind: TokenLT, Lexeme: "<", At: start}, nil
	case '>':
		if l.match('=') {
			return Token{Kind: TokenGTE, Lexeme: ">=", At: start}, nil
		}
		return Token{Kind: TokenGT, Lexeme: ">", At: start}, nil
	case '+':
		return Token{Kind: TokenPlus, Lexeme: "+", At: start}, nil
	case '-':
		return Token{Kind: TokenMinus, Lexeme: "-", At: start}, nil
	case '*':
		return Token{Kind: TokenStar, Lexeme: "*", At: start}, nil
	case '/':
		return Token{Kind: TokenSlash, Lexeme: "/", At: start}, nil
	case '%':
		return Token{Kind: TokenPercent, Lexeme: "%", At: start}, nil
	case '(':
		return Token{Kind: TokenLParen, Lexeme: "(", At: start}, nil
	case ')':
		return Token{Kind: TokenRParen, Lexeme: ")", At: start}, nil
	case ',':
		return Token{Kind: TokenComma, Lexeme: ",", At: start}, nil
	case '.':
		return Token{Kind: TokenDot, Lexeme: ".", At: start}, nil
	case '[':
		return Token{Kind: TokenLBracket, Lexeme: "[", At: start}, nil
	case ']':
		return Token{Kind: TokenRBracket, Lexeme: "]", At: start}, nil
	}
	return Token{}, parseError(CodeUnexpectedToken, start, "unexpected character %q", ch)
}

func (l *lexer) done() bool { return l.idx >= len(l.input) }
func (l *lexer) peek() rune { return l.input[l.idx] }
func (l *lexer) position() Position {
	return Position{Line: l.line, Column: l.column}
}
func (l *lexer) advance() rune {
	ch := l.input[l.idx]
	l.idx++
	if ch == '\n' {
		l.line++
		l.column = 1
	} else {
		l.column++
	}
	return ch
}
func (l *lexer) match(ch rune) bool {
	if l.done() || l.peek() != ch {
		return false
	}
	l.advance()
	return true
}
func (l *lexer) hasNextDigit() bool {
	return l.idx+1 < len(l.input) && isDigit(l.input[l.idx+1])
}

func positionIndex(input []rune, pos Position, fallback int) int {
	line, col := 1, 1
	for i, ch := range input {
		if line == pos.Line && col == pos.Column {
			return i
		}
		if ch == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}
	return fallback
}

func isDigit(ch rune) bool { return ch >= '0' && ch <= '9' }
func isHex(ch rune) bool {
	return isDigit(ch) || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}
func isIdentStart(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}
func isIdentCont(ch rune) bool { return isIdentStart(ch) || isDigit(ch) }
