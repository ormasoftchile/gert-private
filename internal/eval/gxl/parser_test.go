package gxl

import "testing"

func TestParseProductions(t *testing.T) {
	cases := []string{
		`42`,
		`"hello"`,
		`true`,
		`null`,
		`response.items[0].metadata.name`,
		`str.contains(message, "error")`,
		`len(items)`,
		`now()`,
		`not a and b or c`,
		`a + b * -c`,
		`(a + b) >= 3.14`,
	}
	for _, input := range cases {
		if _, err := Parse(input); err != nil {
			t.Fatalf("Parse(%q) returned error: %v", input, err)
		}
	}
}

func TestParseASTShape(t *testing.T) {
	node, err := Parse(`a + b * c`)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	root, ok := node.(*BinaryNode)
	if !ok || root.Op != TokenPlus {
		t.Fatalf("root = %#v, want plus BinaryNode", node)
	}
	right, ok := root.Right.(*BinaryNode)
	if !ok || right.Op != TokenStar {
		t.Fatalf("right = %#v, want multiply BinaryNode", root.Right)
	}
}

func TestParseNegativeCodes(t *testing.T) {
	cases := map[string]string{
		`.foo`:          CodeUnexpectedToken,
		`a +`:           CodeUnexpectedToken,
		`math.abs(x)`:   CodeUnknownNamespace,
		`http.get(url)`: CodeUnknownNamespace,
		`str.explode(s)`: CodeUnknownMethod,
		`items contains "prod"`: CodeForbiddenSyntax,
		`foo[-1]`:       CodeForbiddenSyntax,
		`foo()`:         CodeForbiddenSyntax,
		`+5`:            CodeForbiddenSyntax,
		`(a + b`:        CodeUnmatchedParen,
		`a + b)`:        CodeUnmatchedParen,
		`a < b < c`:     CodeChainedComparison,
		`and`:           CodeKeywordAsIdentifier,
		`str.foo`:       CodeUnexpectedToken,
		`len()`:         CodeUnexpectedToken,
		`1.2.3`:         CodeUnexpectedToken,
	}
	for input, wantCode := range cases {
		_, err := Parse(input)
		if err == nil {
			t.Fatalf("Parse(%q) succeeded, want %s", input, wantCode)
		}
		parseErr, ok := err.(*ParseError)
		if !ok || parseErr.Code != wantCode {
			t.Fatalf("Parse(%q) error = %v, want code %s", input, err, wantCode)
		}
	}
}
