package core

import (
"errors"
"fmt"
"strconv"

"gopkg.in/yaml.v3"
)

var (
// ErrUnsupportedYAMLTag is returned when a YAML node has no PJVM equivalent.
ErrUnsupportedYAMLTag = errors.New("unsupported YAML tag")
// ErrNonStringYAMLKey is returned when a YAML mapping key is not a string scalar.
ErrNonStringYAMLKey = errors.New("non-string YAML mapping key")
)

// FromYAML converts a parsed YAML node into a PJVM Value.
func FromYAML(node *yaml.Node) (Value, error) {
if node == nil {
return Value{}, fmt.Errorf("yaml node: %w", ErrUnsupportedYAMLTag)
}

switch node.Kind {
case yaml.DocumentNode:
if len(node.Content) != 1 {
return Value{}, fmt.Errorf("yaml document with %d roots: %w", len(node.Content), ErrUnsupportedYAMLTag)
}
return FromYAML(node.Content[0])
case yaml.ScalarNode:
return scalarFromYAML(node)
case yaml.SequenceNode:
items := make([]Value, 0, len(node.Content))
for i, child := range node.Content {
value, err := FromYAML(child)
if err != nil {
return Value{}, fmt.Errorf("yaml sequence index %d: %w", i, err)
}
items = append(items, value)
}
return NewArray(items), nil
case yaml.MappingNode:
items := make(map[string]Value, len(node.Content)/2)
for i := 0; i < len(node.Content); i += 2 {
keyNode := node.Content[i]
valueNode := node.Content[i+1]
if keyNode.Kind != yaml.ScalarNode || keyNode.Tag != "!!str" {
return Value{}, fmt.Errorf("yaml mapping key %q has tag %s: %w", keyNode.Value, keyNode.Tag, ErrNonStringYAMLKey)
}
value, err := FromYAML(valueNode)
if err != nil {
return Value{}, fmt.Errorf("yaml mapping key %q: %w", keyNode.Value, err)
}
items[keyNode.Value] = value
}
return NewObject(items)
default:
return Value{}, fmt.Errorf("yaml kind %d tag %s: %w", node.Kind, node.Tag, ErrUnsupportedYAMLTag)
}
}

func scalarFromYAML(node *yaml.Node) (Value, error) {
switch node.Tag {
case "!!null":
return NewNull(), nil
case "!!bool":
value, err := strconv.ParseBool(node.Value)
if err != nil {
return Value{}, fmt.Errorf("yaml bool %q: %w", node.Value, err)
}
return NewBool(value), nil
case "!!int", "!!float":
var value float64
if err := node.Decode(&value); err != nil {
return Value{}, fmt.Errorf("yaml number %q: %w", node.Value, err)
}
return NewNumber(value)
case "!!str":
return NewString(node.Value), nil
default:
return Value{}, fmt.Errorf("yaml scalar tag %s: %w", node.Tag, ErrUnsupportedYAMLTag)
}
}
