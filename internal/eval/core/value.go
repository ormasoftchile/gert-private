package core

// Kind identifies one of the six GERT Portable JSON Value Model kinds.
type Kind uint8

const (
	KindInvalid Kind = iota
	KindNull
	KindBool
	KindNumber
	KindString
	KindArray
	KindObject
)

// Value is the Go representation of a PJVM value.
// Exactly one payload field is meaningful for each non-null Kind.
type Value struct {
	Kind   Kind
	Bool   bool
	Number float64
	String string
	Array  []Value
	Object map[string]Value
}

// Bindings are the run variable context supplied to GXL and GIS evaluation.
type Bindings map[string]Value

// Object is a PJVM object with string keys and PJVM values.
type Object map[string]Value

// Array is an ordered sequence of PJVM values.
type Array []Value
