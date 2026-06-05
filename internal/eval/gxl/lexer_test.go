package gxl

import "testing"

func TestLexTokenTypes(t *testing.T) {
	tokens, err := Lex(`str.contains(name, "x") and list.indexOf(items, 'p') >= 1.5e2 or not flag # comment`)
	if err != nil {
		t.Fatalf("Lex returned error: %v", err)
	}
	want := []TokenKind{
		TokenStr, TokenDot, TokenIdent, TokenLParen, TokenIdent, TokenComma, TokenString, TokenRParen,
		TokenAnd, TokenList, TokenDot, TokenIdent, TokenLParen, TokenIdent, TokenComma, TokenString, TokenRParen,
		TokenGTE, TokenNumber, TokenOr, TokenNot, TokenIdent, TokenEOF,
	}
	if len(tokens) != len(want) {
		t.Fatalf("got %d tokens, want %d: %#v", len(tokens), len(want), tokens)
	}
	for i := range want {
		if tokens[i].Kind != want[i] {
			t.Fatalf("token %d kind = %s, want %s", i, tokens[i].Kind, want[i])
		}
	}
	if tokens[6].Value != "x" || tokens[15].Value != "p" || tokens[19].Lexeme != "1.5e2" {
		t.Fatalf("unexpected token payloads: %#v", tokens)
	}
}

func TestLexTracksLineAndColumn(t *testing.T) {
	tokens, err := Lex("a\n  b")
	if err != nil {
		t.Fatalf("Lex returned error: %v", err)
	}
	if tokens[1].At.Line != 2 || tokens[1].At.Column != 3 {
		t.Fatalf("second token position = %d:%d, want 2:3", tokens[1].At.Line, tokens[1].At.Column)
	}
}

func TestLexRejectsForbiddenSyntax(t *testing.T) {
	cases := map[string]string{
		"a && b":          CodeForbiddenSyntax,
		"a || b":          CodeForbiddenSyntax,
		"!ready":          CodeForbiddenSyntax,
		"007":             CodeInvalidNumber,
		`"bad \q"`:       CodeInvalidEscape,
		`"unterminated`:    CodeUnterminatedString,
	}
	for input, wantCode := range cases {
		_, err := Lex(input)
		if err == nil {
			t.Fatalf("Lex(%q) succeeded, want %s", input, wantCode)
		}
		parseErr, ok := err.(*ParseError)
		if !ok || parseErr.Code != wantCode {
			t.Fatalf("Lex(%q) error = %v, want code %s", input, err, wantCode)
		}
	}
}
