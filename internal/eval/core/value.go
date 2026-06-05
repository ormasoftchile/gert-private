package core

import (
"errors"
"fmt"
"math"
"sort"
"strconv"
"strings"
)

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

var (
// ErrInvalidNumber is returned when a PJVM number would not be finite JSON.
ErrInvalidNumber = errors.New("invalid PJVM number")
// ErrInvalidObject is returned when a PJVM object cannot be constructed safely.
ErrInvalidObject = errors.New("invalid PJVM object")
)

// Value is the Go representation of a PJVM value.
// Values should be built with the New* constructors so callers cannot smuggle
// non-finite numbers or host-native types around the governance boundary.
type Value struct {
kind        Kind
boolValue   bool
numberValue float64
stringValue string
arrayValue  []Value
objectValue map[string]Value
}

// Bindings are the run variable context supplied to GXL and GIS evaluation.
type Bindings map[string]Value

// Object is a PJVM object with string keys and PJVM values.
type Object map[string]Value

// Array is an ordered sequence of PJVM values.
type Array []Value

// NewNull constructs a PJVM null.
func NewNull() Value {
return Value{kind: KindNull}
}

// NewBool constructs a PJVM boolean.
func NewBool(v bool) Value {
return Value{kind: KindBool, boolValue: v}
}

// NewNumber constructs a PJVM number. PJVM follows JSON number portability, so
// NaN and infinities are rejected.
func NewNumber(v float64) (Value, error) {
if math.IsNaN(v) || math.IsInf(v, 0) {
return Value{}, ErrInvalidNumber
}
return Value{kind: KindNumber, numberValue: v}, nil
}

// NewString constructs a PJVM string.
func NewString(v string) Value {
return Value{kind: KindString, stringValue: v}
}

// NewArray constructs a PJVM array. A nil slice is treated as an empty array.
func NewArray(v []Value) Value {
items := make([]Value, len(v))
copy(items, v)
return Value{kind: KindArray, arrayValue: items}
}

// NewObject constructs a PJVM object. Empty string keys are allowed because PJVM
// tracks JSON object semantics and the GERT specs do not currently forbid them.
func NewObject(v map[string]Value) (Value, error) {
if v == nil {
return Value{}, ErrInvalidObject
}
items := make(map[string]Value, len(v))
for key, value := range v {
items[key] = value
}
return Value{kind: KindObject, objectValue: items}, nil
}

// Kind returns the PJVM kind for this value.
func (v Value) Kind() Kind {
return v.kind
}

// AsBool returns the boolean payload and true when the value is a PJVM boolean.
func (v Value) AsBool() (bool, bool) {
if v.kind != KindBool {
return false, false
}
return v.boolValue, true
}

// AsNumber returns the number payload and true when the value is a PJVM number.
func (v Value) AsNumber() (float64, bool) {
if v.kind != KindNumber {
return 0, false
}
return v.numberValue, true
}

// AsString returns the string payload and true when the value is a PJVM string.
func (v Value) AsString() (string, bool) {
if v.kind != KindString {
return "", false
}
return v.stringValue, true
}

// AsArray returns a copy of the array payload and true when the value is a PJVM array.
func (v Value) AsArray() ([]Value, bool) {
if v.kind != KindArray {
return nil, false
}
items := make([]Value, len(v.arrayValue))
copy(items, v.arrayValue)
return items, true
}

// AsObject returns a copy of the object payload and true when the value is a PJVM object.
func (v Value) AsObject() (map[string]Value, bool) {
if v.kind != KindObject {
return nil, false
}
items := make(map[string]Value, len(v.objectValue))
for key, value := range v.objectValue {
items[key] = value
}
return items, true
}

// Equal reports whether two PJVM values are deeply equal.
func (v Value) Equal(other Value) bool {
if v.kind != other.kind {
return false
}
switch v.kind {
case KindNull:
return true
case KindBool:
return v.boolValue == other.boolValue
case KindNumber:
return v.numberValue == other.numberValue
case KindString:
return v.stringValue == other.stringValue
case KindArray:
if len(v.arrayValue) != len(other.arrayValue) {
return false
}
for i := range v.arrayValue {
if !v.arrayValue[i].Equal(other.arrayValue[i]) {
return false
}
}
return true
case KindObject:
if len(v.objectValue) != len(other.objectValue) {
return false
}
for key, left := range v.objectValue {
right, ok := other.objectValue[key]
if !ok || !left.Equal(right) {
return false
}
}
return true
default:
return other.kind == KindInvalid
}
}

// String returns a human-readable representation for diagnostics. It is not the
// canonical renderer or any wire format.
func (v Value) String() string {
switch v.kind {
case KindNull:
return "null"
case KindBool:
return strconv.FormatBool(v.boolValue)
case KindNumber:
return strconv.FormatFloat(v.numberValue, 'g', -1, 64)
case KindString:
return strconv.Quote(v.stringValue)
case KindArray:
parts := make([]string, len(v.arrayValue))
for i, item := range v.arrayValue {
parts[i] = item.String()
}
return "[" + strings.Join(parts, ", ") + "]"
case KindObject:
keys := make([]string, 0, len(v.objectValue))
for key := range v.objectValue {
keys = append(keys, key)
}
sort.Strings(keys)
parts := make([]string, 0, len(keys))
for _, key := range keys {
parts = append(parts, fmt.Sprintf("%s: %s", strconv.Quote(key), v.objectValue[key].String()))
}
return "{" + strings.Join(parts, ", ") + "}"
default:
return "<invalid>"
}
}

func (k Kind) String() string {
switch k {
case KindNull:
return "null"
case KindBool:
return "bool"
case KindNumber:
return "number"
case KindString:
return "string"
case KindArray:
return "array"
case KindObject:
return "object"
default:
return "invalid"
}
}
