package gxl

import (
	"fmt"
	"math"
	"strconv"

	"github.com/ormasoftchile/gert/internal/eval/core"
)

const (
	CodeTypeComparison   = "GXL-TYPE-001"
	CodeTypeBool         = "GXL-TYPE-002"
	CodeTypeArgument     = "GXL-TYPE-003"
	CodeTypeArity        = "GXL-TYPE-004"
	CodePathMissing      = "GXL-PATH-001"
	CodePathOutOfBounds  = "GXL-PATH-002"
	CodePathNonArray     = "GXL-PATH-003"
	CodeEvalDivisionZero = "GXL-EVAL-002"
	CodeEvalRegex        = "GXL-EVAL-003"
	CodeEvalNullOrdered  = "GXL-EVAL-004"
	CodeTBD              = "TBD"
)

// Eval evaluates a parsed GXL AST against PJVM bindings using an injectable clock.
func Eval(node Node, bindings map[string]core.Value, clock core.Clock) (core.Value, error) {
	if clock == nil {
		clock = core.SystemClock()
	}
	e := evaluator{bindings: bindings, clock: clock}
	return e.eval(node)
}

// EvalError is a structured GXL evaluation diagnostic with a stable conformance code.
type EvalError struct {
	Code    string
	Message string
	At      Position
}

func (e *EvalError) Error() string {
	return fmt.Sprintf("%s at %d:%d: %s", e.Code, e.At.Line, e.At.Column, e.Message)
}

// ErrorCode returns the stable conformance error code.
func (e *EvalError) ErrorCode() string { return e.Code }

func evalError(code string, pos Position, format string, args ...any) *EvalError {
	return &EvalError{Code: code, At: pos, Message: fmt.Sprintf(format, args...)}
}

type evaluator struct {
	bindings map[string]core.Value
	clock    core.Clock
}

func (e evaluator) eval(node Node) (core.Value, error) {
	switch n := node.(type) {
	case *LiteralNode:
		return e.evalLiteral(n)
	case *PathNode:
		return e.evalPath(n)
	case *UnaryNode:
		return e.evalUnary(n)
	case *BinaryNode:
		return e.evalBinary(n)
	case *CallNode:
		return e.evalCall(n)
	default:
		return core.Value{}, evalError("GXL-EVAL-001", node.Pos(), "unknown AST node %T", node)
	}
}

func (e evaluator) evalLiteral(n *LiteralNode) (core.Value, error) {
	switch n.Kind {
	case LiteralString:
		return core.NewString(n.Value), nil
	case LiteralNumber:
		num, err := strconv.ParseFloat(n.Value, 64)
		if err != nil {
			return core.Value{}, evalError(CodeInvalidNumber, n.Position, "invalid number literal %q", n.Value)
		}
		return numberValue(num, n.Position)
	case LiteralBool:
		return core.NewBool(n.Value == "true"), nil
	case LiteralNull:
		return core.NewNull(), nil
	default:
		return core.Value{}, evalError("GXL-EVAL-001", n.Position, "unknown literal kind %s", n.Kind)
	}
}

func (e evaluator) evalPath(n *PathNode) (core.Value, error) {
	value, ok := e.bindings[n.Root]
	if !ok {
		return core.Value{}, evalError(CodePathMissing, n.Position, "variable %q not found", n.Root)
	}
	for _, segment := range n.Segments {
		switch segment.Kind {
		case PathSegmentField:
			obj, ok := value.AsObject()
			if !ok {
				return core.Value{}, evalError(CodePathMissing, segment.Position, "field %q on non-object", segment.Name)
			}
			next, ok := obj[segment.Name]
			if !ok {
				return core.Value{}, evalError(CodePathMissing, segment.Position, "field %q not found", segment.Name)
			}
			value = next
		case PathSegmentIndex:
			arr, ok := value.AsArray()
			if !ok {
				return core.Value{}, evalError(CodePathNonArray, segment.Position, "index on non-array")
			}
			idx, err := strconv.Atoi(segment.Index)
			if err != nil || idx < 0 {
				return core.Value{}, evalError(CodePathOutOfBounds, segment.Position, "invalid index %q", segment.Index)
			}
			if idx >= len(arr) {
				return core.Value{}, evalError(CodePathOutOfBounds, segment.Position, "index %d out of bounds", idx)
			}
			value = arr[idx]
		}
	}
	return value, nil
}

func (e evaluator) evalUnary(n *UnaryNode) (core.Value, error) {
	value, err := e.eval(n.Expr)
	if err != nil {
		return core.Value{}, err
	}
	switch n.Op {
	case TokenNot:
		b, ok := value.AsBool()
		if !ok {
			return core.Value{}, evalError(CodeTypeBool, n.Position, "not requires bool operand, got %s", value.Kind())
		}
		return core.NewBool(!b), nil
	case TokenMinus:
		num, ok := value.AsNumber()
		if !ok {
			return core.Value{}, evalError(CodeTypeArgument, n.Position, "unary minus requires number operand, got %s", value.Kind())
		}
		return numberValue(-num, n.Position)
	default:
		return core.Value{}, evalError("GXL-EVAL-001", n.Position, "unknown unary operator %s", n.Op)
	}
}

func (e evaluator) evalBinary(n *BinaryNode) (core.Value, error) {
	switch n.Op {
	case TokenAnd:
		left, err := e.eval(n.Left)
		if err != nil {
			return core.Value{}, err
		}
		b, ok := left.AsBool()
		if !ok {
			return core.Value{}, evalError(CodeTypeBool, n.Position, "and requires bool left operand, got %s", left.Kind())
		}
		if !b {
			return left, nil
		}
		return e.eval(n.Right)
	case TokenOr:
		left, err := e.eval(n.Left)
		if err != nil {
			return core.Value{}, err
		}
		b, ok := left.AsBool()
		if !ok {
			return core.Value{}, evalError(CodeTypeBool, n.Position, "or requires bool left operand, got %s", left.Kind())
		}
		if b {
			return left, nil
		}
		return e.eval(n.Right)
	}

	left, err := e.eval(n.Left)
	if err != nil {
		return core.Value{}, err
	}
	right, err := e.eval(n.Right)
	if err != nil {
		return core.Value{}, err
	}

	switch n.Op {
	case TokenEQ, TokenNEQ, TokenLT, TokenGT, TokenLTE, TokenGTE:
		return compareValues(n.Op, left, right, n.Position)
	case TokenPlus, TokenMinus, TokenStar, TokenSlash, TokenPercent:
		return arithmeticValues(n.Op, left, right, n.Position)
	default:
		return core.Value{}, evalError("GXL-EVAL-001", n.Position, "unknown binary operator %s", n.Op)
	}
}

func (e evaluator) evalCall(n *CallNode) (core.Value, error) {
	return evalStdlibCall(n, func(arg Node) (core.Value, error) { return e.eval(arg) }, e.clock)
}

func compareValues(op TokenKind, left, right core.Value, pos Position) (core.Value, error) {
	if op == TokenEQ || op == TokenNEQ {
		if left.Kind() == core.KindNull || right.Kind() == core.KindNull {
			eq := left.Kind() == core.KindNull && right.Kind() == core.KindNull
			if op == TokenNEQ {
				eq = !eq
			}
			return core.NewBool(eq), nil
		}
		if left.Kind() != right.Kind() {
			return core.Value{}, evalError(CodeTypeComparison, pos, "cannot compare %s to %s", left.Kind(), right.Kind())
		}
		switch left.Kind() {
		case core.KindBool, core.KindNumber, core.KindString:
			eq := left.Equal(right)
			if op == TokenNEQ {
				eq = !eq
			}
			return core.NewBool(eq), nil
		default:
			return core.Value{}, evalError(CodeTBD, pos, "equality for %s is unspecified", left.Kind())
		}
	}

	if left.Kind() == core.KindNull || right.Kind() == core.KindNull {
		return core.Value{}, evalError(CodeEvalNullOrdered, pos, "null in ordered comparison")
	}
	if left.Kind() != right.Kind() {
		return core.Value{}, evalError(CodeTypeComparison, pos, "cannot compare %s to %s", left.Kind(), right.Kind())
	}
	switch left.Kind() {
	case core.KindNumber:
		l, _ := left.AsNumber()
		r, _ := right.AsNumber()
		return core.NewBool(compareFloat(op, l, r)), nil
	case core.KindString:
		l, _ := left.AsString()
		r, _ := right.AsString()
		return core.NewBool(compareString(op, l, r)), nil
	case core.KindBool:
		return core.Value{}, evalError(CodeTBD, pos, "ordered comparison for bool is unspecified")
	default:
		return core.Value{}, evalError(CodeTypeComparison, pos, "ordered comparison for %s is not supported", left.Kind())
	}
}

func compareFloat(op TokenKind, left, right float64) bool {
	switch op {
	case TokenLT:
		return left < right
	case TokenGT:
		return left > right
	case TokenLTE:
		return left <= right
	case TokenGTE:
		return left >= right
	}
	return false
}

func compareString(op TokenKind, left, right string) bool {
	switch op {
	case TokenLT:
		return left < right
	case TokenGT:
		return left > right
	case TokenLTE:
		return left <= right
	case TokenGTE:
		return left >= right
	}
	return false
}

func arithmeticValues(op TokenKind, left, right core.Value, pos Position) (core.Value, error) {
	l, ok := left.AsNumber()
	if !ok {
		return core.Value{}, evalError(CodeTypeArgument, pos, "arithmetic requires number left operand, got %s", left.Kind())
	}
	r, ok := right.AsNumber()
	if !ok {
		return core.Value{}, evalError(CodeTypeArgument, pos, "arithmetic requires number right operand, got %s", right.Kind())
	}
	switch op {
	case TokenPlus:
		return numberValue(l+r, pos)
	case TokenMinus:
		return numberValue(l-r, pos)
	case TokenStar:
		return numberValue(l*r, pos)
	case TokenSlash:
		if r == 0 {
			return core.Value{}, evalError(CodeEvalDivisionZero, pos, "division by zero")
		}
		return numberValue(l/r, pos)
	case TokenPercent:
		if r == 0 {
			return core.Value{}, evalError(CodeEvalDivisionZero, pos, "modulo by zero")
		}
		return numberValue(math.Mod(l, r), pos)
	default:
		return core.Value{}, evalError("GXL-EVAL-001", pos, "unknown arithmetic operator %s", op)
	}
}

func numberValue(v float64, pos Position) (core.Value, error) {
	value, err := core.NewNumber(v)
	if err != nil {
		return core.Value{}, evalError("GXL-EVAL-001", pos, "invalid number result")
	}
	return value, nil
}
