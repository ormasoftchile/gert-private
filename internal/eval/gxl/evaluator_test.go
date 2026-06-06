package gxl

import (
	"testing"
	"time"

	"github.com/ormasoftchile/gert/internal/eval/core"
)

func TestEvalOperators(t *testing.T) {
	cases := []struct {
		name string
		expr string
		want core.Value
	}{
		{name: "addition", expr: "1 + 2", want: mustNumber(t, 3)},
		{name: "subtraction", expr: "5 - 3", want: mustNumber(t, 2)},
		{name: "multiplication", expr: "3 * 4", want: mustNumber(t, 12)},
		{name: "division", expr: "7 / 2", want: mustNumber(t, 3.5)},
		{name: "modulo", expr: "-7 % 3", want: mustNumber(t, -1)},
		{name: "unary minus", expr: "-(1 + 2)", want: mustNumber(t, -3)},
		{name: "number compare", expr: "1 < 2", want: core.NewBool(true)},
		{name: "string compare", expr: `"abc" < "abd"`, want: core.NewBool(true)},
		{name: "bool equality", expr: "true != false", want: core.NewBool(true)},
		{name: "null equality", expr: "null == null", want: core.NewBool(true)},
		{name: "not", expr: "not false", want: core.NewBool(true)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := evalSource(t, tc.expr, nil, core.FixedClock(time.Unix(0, 0).UTC()))
			if err != nil {
				t.Fatalf("Eval() error = %v", err)
			}
			if !got.Equal(tc.want) {
				t.Fatalf("Eval() = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestEvalShortCircuit(t *testing.T) {
	cases := []struct {
		expr string
		want core.Value
	}{
		{expr: "false and missing", want: core.NewBool(false)},
		{expr: "true or missing", want: core.NewBool(true)},
		{expr: "true and 42", want: mustNumber(t, 42)},
		{expr: "false or \"fallback\"", want: core.NewString("fallback")},
	}
	for _, tc := range cases {
		t.Run(tc.expr, func(t *testing.T) {
			got, err := evalSource(t, tc.expr, nil, nil)
			if err != nil {
				t.Fatalf("Eval() error = %v", err)
			}
			if !got.Equal(tc.want) {
				t.Fatalf("Eval() = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestEvalTypeErrors(t *testing.T) {
	cases := []struct {
		expr string
		code string
	}{
		{expr: `1 == "1"`, code: CodeTypeComparison},
		{expr: "not null", code: CodeTypeBool},
		{expr: `1 + "x"`, code: CodeTypeArgument},
		{expr: "1 / 0", code: CodeEvalDivisionZero},
		{expr: "null < 0", code: CodeEvalNullOrdered},
	}
	for _, tc := range cases {
		t.Run(tc.expr, func(t *testing.T) {
			_, err := evalSource(t, tc.expr, nil, nil)
			if err == nil {
				t.Fatal("Eval() error = nil")
			}
			if codeOf(err) != tc.code {
				t.Fatalf("Eval() code = %q, want %q (%v)", codeOf(err), tc.code, err)
			}
		})
	}
}

func TestEvalIdentifierAndPathResolution(t *testing.T) {
	num := mustNumber(t, 2)
	obj, err := core.NewObject(map[string]core.Value{"items": core.NewArray([]core.Value{core.NewString("zero"), num})})
	if err != nil {
		t.Fatal(err)
	}
	got, err := evalSource(t, "config.items[1]", map[string]core.Value{"config": obj}, nil)
	if err != nil {
		t.Fatalf("Eval() error = %v", err)
	}
	if !got.Equal(num) {
		t.Fatalf("Eval() = %s, want %s", got, num)
	}
	_, err = evalSource(t, "missing", nil, nil)
	if codeOf(err) != CodePathMissing {
		t.Fatalf("missing identifier code = %q, want %q", codeOf(err), CodePathMissing)
	}
}

func TestEvalStdlib(t *testing.T) {
	list := core.NewArray([]core.Value{core.NewString("a"), mustNumber(t, 2)})
	bindings := map[string]core.Value{"list": list, "nullVar": core.NewNull()}
	cases := []struct {
		expr string
		want core.Value
	}{
		{expr: `len("hé")`, want: mustNumber(t, 2)},
		{expr: "len(list)", want: mustNumber(t, 2)},
		{expr: `str.startsWith("hello", "he")`, want: core.NewBool(true)},
		{expr: `str.endsWith("hello", "lo")`, want: core.NewBool(true)},
		{expr: `str.contains("hello", "ell")`, want: core.NewBool(true)},
		{expr: `str.toLower("HELLO")`, want: core.NewString("hello")},
		{expr: `str.toUpper("hello")`, want: core.NewString("HELLO")},
		{expr: `str.trim("  hello  ")`, want: core.NewString("hello")},
		{expr: `str.trimPrefix("prefix-value", "prefix-")`, want: core.NewString("value")},
		{expr: `str.trimSuffix("value.txt", ".txt")`, want: core.NewString("value")},
		{expr: `str.length("hello")`, want: mustNumber(t, 5)},
		{expr: `list.contains(list, "a")`, want: core.NewBool(true)},
		{expr: "list.indexOf(list, 2)", want: mustNumber(t, 1)},
		{expr: "list.length(list)", want: mustNumber(t, 2)},
		{expr: `regex.match("abc123", "^[a-z]+\\d+$")`, want: core.NewBool(true)},
	}
	for _, tc := range cases {
		t.Run(tc.expr, func(t *testing.T) {
			got, err := evalSource(t, tc.expr, bindings, nil)
			if err != nil {
				t.Fatalf("Eval() error = %v", err)
			}
			if !got.Equal(tc.want) {
				t.Fatalf("Eval() = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestEvalStdlibErrorsAndNow(t *testing.T) {
	_, err := evalSource(t, `str.startsWith("hello")`, nil, nil)
	if codeOf(err) != CodeTypeArity {
		t.Fatalf("arity code = %q, want %q", codeOf(err), CodeTypeArity)
	}
	_, err = evalSource(t, "len(nullVar)", map[string]core.Value{"nullVar": core.NewNull()}, nil)
	if codeOf(err) != CodeTypeArgument {
		t.Fatalf("len type code = %q, want %q", codeOf(err), CodeTypeArgument)
	}
	_, err = evalSource(t, `regex.match("x", "[")`, nil, nil)
	if codeOf(err) != CodeEvalRegex {
		t.Fatalf("regex code = %q, want %q", codeOf(err), CodeEvalRegex)
	}

	fixed := time.Date(2026, 6, 5, 17, 57, 33, 200000000, time.FixedZone("PDT", -7*60*60))
	got, err := evalSource(t, "now()", nil, core.FixedClock(fixed))
	if err != nil {
		t.Fatalf("now() error = %v", err)
	}
	want := core.NewString("2026-06-06T00:57:33Z")
	if !got.Equal(want) {
		t.Fatalf("now() = %s, want %s", got, want)
	}
}

func evalSource(t *testing.T, expr string, bindings map[string]core.Value, clock core.Clock) (core.Value, error) {
	t.Helper()
	ast, err := Parse(expr)
	if err != nil {
		return core.Value{}, err
	}
	return Eval(ast, bindings, clock)
}

func mustNumber(t *testing.T, value float64) core.Value {
	t.Helper()
	num, err := core.NewNumber(value)
	if err != nil {
		t.Fatal(err)
	}
	return num
}

func codeOf(err error) string {
	type coded interface {
		ErrorCode() string
	}
	if err == nil {
		return ""
	}
	if codedErr, ok := err.(coded); ok {
		return codedErr.ErrorCode()
	}
	return ""
}
