package tool

import (
	"strings"
	"testing"
)

func TestVSCodeInputAdapterValidation(t *testing.T) {
	valid := `apiVersion: tool/v1
meta: { name: icm, version: "1.0.0" }
transport: { mode: vscode-mcp }
actions:
  - name: get-incident
    args:
      incident_id: { type: string, required: true }
    vscode_input_adapter:
      incidentId: { from: incident_id, coerce: integer }
`
	if _, err := parseToolFileFromString(t, valid); err != nil {
		t.Fatalf("valid vscode input adapter rejected: %v", err)
	}

	for name, content := range map[string]string{
		"non-vscode transport": strings.Replace(valid, "transport: { mode: vscode-mcp }", "transport: { mode: native }", 1),
		"unknown source":       strings.Replace(valid, "from: incident_id", "from: missing", 1),
		"unsupported coercion": strings.Replace(valid, "coerce: integer", "coerce: decimal", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseToolFileFromString(t, content); err == nil || !strings.Contains(err.Error(), "VSCODE-002") {
				t.Fatalf("expected VSCODE-002, got %v", err)
			}
		})
	}
}
