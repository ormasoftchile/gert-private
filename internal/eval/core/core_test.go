package core

import (
"errors"
"math"
"testing"
"time"

"gopkg.in/yaml.v3"
)

func TestConstructorsAccessorsAndEquality(t *testing.T) {
number, err := NewNumber(42)
if err != nil {
t.Fatalf("NewNumber: %v", err)
}
object, err := NewObject(map[string]Value{
"":       NewString("empty key allowed"),
"flag":   NewBool(true),
"number": number,
})
if err != nil {
t.Fatalf("NewObject: %v", err)
}
clone, err := NewObject(map[string]Value{
"number": number,
"flag":   NewBool(true),
"":       NewString("empty key allowed"),
})
if err != nil {
t.Fatalf("NewObject clone: %v", err)
}
if object.Kind() != KindObject {
t.Fatalf("Kind() = %s, want object", object.Kind())
}
if !object.Equal(clone) {
t.Fatalf("Equal() = false, want true")
}
if _, ok := object.AsString(); ok {
t.Fatalf("AsString on object ok = true, want false")
}
gotObject, ok := object.AsObject()
if !ok || len(gotObject) != 3 {
t.Fatalf("AsObject ok=%v len=%d, want true len 3", ok, len(gotObject))
}
}

func TestNewNumberRejectsNonFinite(t *testing.T) {
for _, input := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
_, err := NewNumber(input)
if !errors.Is(err, ErrInvalidNumber) {
t.Fatalf("NewNumber(%v) err=%v, want ErrInvalidNumber", input, err)
}
}
}

func TestNewObjectRejectsNilMap(t *testing.T) {
_, err := NewObject(nil)
if !errors.Is(err, ErrInvalidObject) {
t.Fatalf("NewObject(nil) err=%v, want ErrInvalidObject", err)
}
}

func TestFromYAMLPrimitives(t *testing.T) {
cases := []struct {
name string
yaml string
want Value
}{
{name: "null", yaml: "null", want: NewNull()},
{name: "bool", yaml: "true", want: NewBool(true)},
{name: "int", yaml: "7", want: mustNumber(t, 7)},
{name: "float", yaml: "3.5", want: mustNumber(t, 3.5)},
{name: "string", yaml: "hello", want: NewString("hello")},
}
for _, tc := range cases {
t.Run(tc.name, func(t *testing.T) {
got := parseYAMLValue(t, tc.yaml)
if !got.Equal(tc.want) {
t.Fatalf("FromYAML(%q) = %s, want %s", tc.yaml, got, tc.want)
}
})
}
}

func TestFromYAMLNestedStructures(t *testing.T) {
got := parseYAMLValue(t, "name: svc\nports: [80, 443]\nmeta:\n  enabled: true\n  nothing: null\n")
wantObject, err := NewObject(map[string]Value{
"name":  NewString("svc"),
"ports": NewArray([]Value{mustNumber(t, 80), mustNumber(t, 443)}),
"meta": mustObject(t, map[string]Value{
"enabled": NewBool(true),
"nothing": NewNull(),
}),
})
if err != nil {
t.Fatalf("NewObject: %v", err)
}
if !got.Equal(wantObject) {
t.Fatalf("nested FromYAML = %s, want %s", got, wantObject)
}
}

func TestFromYAMLRejectsNaN(t *testing.T) {
var node yaml.Node
if err := yaml.Unmarshal([]byte(".nan"), &node); err != nil {
t.Fatalf("yaml.Unmarshal: %v", err)
}
_, err := FromYAML(&node)
if !errors.Is(err, ErrInvalidNumber) {
t.Fatalf("FromYAML(.nan) err=%v, want ErrInvalidNumber", err)
}
}

func TestFromYAMLRejectsNonStringMappingKey(t *testing.T) {
var node yaml.Node
if err := yaml.Unmarshal([]byte("1: one"), &node); err != nil {
t.Fatalf("yaml.Unmarshal: %v", err)
}
_, err := FromYAML(&node)
if !errors.Is(err, ErrNonStringYAMLKey) {
t.Fatalf("FromYAML(non-string key) err=%v, want ErrNonStringYAMLKey", err)
}
}

func TestClockImplementations(t *testing.T) {
fixed := time.Date(2026, 6, 5, 13, 29, 45, 398000000, time.FixedZone("PDT", -7*60*60))
if !FixedClock(fixed).Now().Equal(fixed) {
t.Fatalf("FixedClock did not return fixed time")
}
if SystemClock().Now().IsZero() {
t.Fatalf("SystemClock returned zero time")
}
}

func parseYAMLValue(t *testing.T, source string) Value {
t.Helper()
var node yaml.Node
if err := yaml.Unmarshal([]byte(source), &node); err != nil {
t.Fatalf("yaml.Unmarshal: %v", err)
}
value, err := FromYAML(&node)
if err != nil {
t.Fatalf("FromYAML: %v", err)
}
return value
}

func mustNumber(t *testing.T, input float64) Value {
t.Helper()
value, err := NewNumber(input)
if err != nil {
t.Fatalf("NewNumber(%v): %v", input, err)
}
return value
}

func mustObject(t *testing.T, input map[string]Value) Value {
t.Helper()
value, err := NewObject(input)
if err != nil {
t.Fatalf("NewObject: %v", err)
}
return value
}
