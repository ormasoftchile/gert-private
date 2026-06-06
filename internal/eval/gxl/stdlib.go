package gxl

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/ormasoftchile/gert/internal/eval/core"
)

type argEvaluator func(Node) (core.Value, error)

func evalStdlibCall(call *CallNode, evalArg argEvaluator, clock core.Clock) (core.Value, error) {
	if call.Namespace == "" {
		switch call.Method {
		case "len":
			args, err := evalArgs(call, evalArg, 1)
			if err != nil {
				return core.Value{}, err
			}
			return builtinLen(args[0], call.Position)
		case "now":
			if len(call.Args) != 0 {
				return core.Value{}, evalError(CodeTypeArity, call.Position, "now expects 0 arguments, got %d", len(call.Args))
			}
			return core.NewString(clock.Now().UTC().Format("2006-01-02T15:04:05Z")), nil
		}
	}

	switch call.Namespace {
	case "str":
		return evalStrCall(call, evalArg)
	case "list":
		return evalListCall(call, evalArg)
	case "regex":
		return evalRegexCall(call, evalArg)
	}
	return core.Value{}, evalError(CodeUnknownNamespace, call.Position, "unknown namespace %q", call.Namespace)
}

func evalStrCall(call *CallNode, evalArg argEvaluator) (core.Value, error) {
	switch call.Method {
	case "startsWith", "endsWith", "contains", "trimPrefix", "trimSuffix":
		args, err := evalArgs(call, evalArg, 2)
		if err != nil {
			return core.Value{}, err
		}
		left, err := requireString(args[0], call.Position)
		if err != nil {
			return core.Value{}, err
		}
		right, err := requireString(args[1], call.Position)
		if err != nil {
			return core.Value{}, err
		}
		switch call.Method {
		case "startsWith":
			return core.NewBool(strings.HasPrefix(left, right)), nil
		case "endsWith":
			return core.NewBool(strings.HasSuffix(left, right)), nil
		case "contains":
			return core.NewBool(strings.Contains(left, right)), nil
		case "trimPrefix":
			return core.NewString(strings.TrimPrefix(left, right)), nil
		case "trimSuffix":
			return core.NewString(strings.TrimSuffix(left, right)), nil
		}
	case "toLower", "toUpper", "trim", "length":
		args, err := evalArgs(call, evalArg, 1)
		if err != nil {
			return core.Value{}, err
		}
		s, err := requireString(args[0], call.Position)
		if err != nil {
			return core.Value{}, err
		}
		switch call.Method {
		case "toLower":
			return core.NewString(strings.ToLower(s)), nil
		case "toUpper":
			return core.NewString(strings.ToUpper(s)), nil
		case "trim":
			return core.NewString(strings.TrimSpace(s)), nil
		case "length":
			return numberValue(float64(utf8.RuneCountInString(s)), call.Position)
		}
	}
	return core.Value{}, evalError(CodeUnknownMethod, call.Position, "unknown method str.%s", call.Method)
}

func evalListCall(call *CallNode, evalArg argEvaluator) (core.Value, error) {
	switch call.Method {
	case "contains", "indexOf":
		args, err := evalArgs(call, evalArg, 2)
		if err != nil {
			return core.Value{}, err
		}
		items, err := requireList(args[0], call.Position)
		if err != nil {
			return core.Value{}, err
		}
		idx, err := listIndexOf(items, args[1], call.Position)
		if err != nil {
			return core.Value{}, err
		}
		if call.Method == "contains" {
			return core.NewBool(idx >= 0), nil
		}
		return numberValue(float64(idx), call.Position)
	case "length":
		args, err := evalArgs(call, evalArg, 1)
		if err != nil {
			return core.Value{}, err
		}
		items, err := requireList(args[0], call.Position)
		if err != nil {
			return core.Value{}, err
		}
		return numberValue(float64(len(items)), call.Position)
	}
	return core.Value{}, evalError(CodeUnknownMethod, call.Position, "unknown method list.%s", call.Method)
}

func evalRegexCall(call *CallNode, evalArg argEvaluator) (core.Value, error) {
	if call.Method != "match" {
		return core.Value{}, evalError(CodeUnknownMethod, call.Position, "unknown method regex.%s", call.Method)
	}
	args, err := evalArgs(call, evalArg, 2)
	if err != nil {
		return core.Value{}, err
	}
	s, err := requireString(args[0], call.Position)
	if err != nil {
		return core.Value{}, err
	}
	pattern, err := requireString(args[1], call.Position)
	if err != nil {
		return core.Value{}, err
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return core.Value{}, evalError(CodeEvalRegex, call.Position, "invalid regex pattern")
	}
	return core.NewBool(re.MatchString(s)), nil
}

func evalArgs(call *CallNode, evalArg argEvaluator, want int) ([]core.Value, error) {
	if len(call.Args) != want {
		return nil, evalError(CodeTypeArity, call.Position, "%s.%s expects %d arguments, got %d", call.Namespace, call.Method, want, len(call.Args))
	}
	args := make([]core.Value, 0, len(call.Args))
	for _, arg := range call.Args {
		value, err := evalArg(arg)
		if err != nil {
			return nil, err
		}
		args = append(args, value)
	}
	return args, nil
}

func builtinLen(value core.Value, pos Position) (core.Value, error) {
	switch value.Kind() {
	case core.KindString:
		s, _ := value.AsString()
		return numberValue(float64(utf8.RuneCountInString(s)), pos)
	case core.KindArray:
		items, _ := value.AsArray()
		return numberValue(float64(len(items)), pos)
	default:
		return core.Value{}, evalError(CodeTypeArgument, pos, "len does not accept %s", value.Kind())
	}
}

func requireString(value core.Value, pos Position) (string, error) {
	s, ok := value.AsString()
	if !ok {
		return "", evalError(CodeTypeArgument, pos, "expected string, got %s", value.Kind())
	}
	return s, nil
}

func requireList(value core.Value, pos Position) ([]core.Value, error) {
	items, ok := value.AsArray()
	if !ok {
		return nil, evalError(CodeTypeArgument, pos, "expected list, got %s", value.Kind())
	}
	return items, nil
}

func listIndexOf(items []core.Value, needle core.Value, pos Position) (int, error) {
	if needle.Kind() == core.KindNull {
		return -1, nil
	}
	if !isScalar(needle) {
		return 0, evalError(CodeTypeArgument, pos, "list item must be scalar, got %s", needle.Kind())
	}
	for i, item := range items {
		if item.Kind() == core.KindNull {
			continue
		}
		if !isScalar(item) {
			continue
		}
		if item.Kind() == needle.Kind() && item.Equal(needle) {
			return i, nil
		}
	}
	return -1, nil
}

func isScalar(value core.Value) bool {
	switch value.Kind() {
	case core.KindBool, core.KindNumber, core.KindString:
		return true
	default:
		return false
	}
}
