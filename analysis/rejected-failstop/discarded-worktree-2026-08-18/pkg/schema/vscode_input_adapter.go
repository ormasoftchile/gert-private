package schema

// VSCodeInputBinding maps a logical action argument to a VS Code MCP provider
// parameter. Coerce is optional; "integer" performs an exact integer
// conversion in the VS Code bridge.
type VSCodeInputBinding struct {
	From   string `yaml:"from" json:"from"`
	Coerce string `yaml:"coerce,omitempty" json:"coerce,omitempty"`
}
