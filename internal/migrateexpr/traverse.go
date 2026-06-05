package migrateexpr

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Result is returned by TranslateFile.
type Result struct {
	Output      []byte
	Translated  int
	Deferred    int
	Warnings    []Warning
	ByRule      map[string]int
}

// TranslateFile parses a YAML document, walks every scalar node, applies
// translation rules position-aware, and returns the modified YAML bytes.
func TranslateFile(src []byte) (*Result, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(src, &doc); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	res := &Result{
		ByRule: make(map[string]int),
	}

	// Walk the document node (wraps the root mapping/sequence).
	if len(doc.Content) > 0 {
		walkNode(doc.Content[0], "", false, res)
	}

	// Marshal back preserving node styles.
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return nil, fmt.Errorf("encoding YAML: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("closing encoder: %w", err)
	}

	res.Output = buf.Bytes()
	return res, nil
}

// walkNode recursively visits yaml.Node trees.
//
// parentKey is the mapping key name whose value the current node is, used for
// determining position type (expression vs GIS interpolation).
// inIterate tracks whether we're inside an "iterate:" mapping, needed for E-008.
func walkNode(node *yaml.Node, parentKey string, inIterate bool, res *Result) {
	switch node.Kind {
	case yaml.DocumentNode:
		for _, child := range node.Content {
			walkNode(child, "", false, res)
		}

	case yaml.MappingNode:
		nextIterate := inIterate
		// If we are entering an "iterate" mapping (parentKey == "iterate"),
		// signal to children that the "over" key should use E-008.
		if parentKey == "iterate" {
			nextIterate = true
		}
		for i := 0; i+1 < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valNode := node.Content[i+1]
			walkNode(valNode, keyNode.Value, nextIterate, res)
		}

	case yaml.SequenceNode:
		for _, child := range node.Content {
			walkNode(child, parentKey, inIterate, res)
		}

	case yaml.ScalarNode:
		// Only translate string scalars; skip null, bool, int, float.
		if node.Tag != "!!str" && node.Tag != "" {
			// yaml.v3 uses "" tag for scalars when the tag is implicit.
			// If tag is explicitly non-string, skip.
			if !isLikelyString(node) {
				return
			}
		}
		translateScalar(node, parentKey, inIterate, res)
	}
}

// isLikelyString returns true for untagged scalars whose style implies string,
// or whose tag is !!str.
func isLikelyString(node *yaml.Node) bool {
	switch node.Tag {
	case "!!str", "":
		return true
	default:
		return false
	}
}

// translateScalar applies the appropriate rules to a YAML scalar node.
func translateScalar(node *yaml.Node, parentKey string, inIterate bool, res *Result) {
	original := node.Value
	lineHint := node.Line

	// E-008: iterate.over: "$.IDENT" → bare identifier.
	if inIterate && parentKey == iterateOverKey {
		if translated, ruleID, changed := TranslateOverValue(original); changed {
			node.Value = translated
			// Strip surrounding quotes from the YAML scalar style so it
			// marshals as a bare identifier, not a quoted string.
			node.Style = 0
			node.Tag = "!!str"
			res.Translated++
			res.ByRule[ruleID]++
			recordChange(res, ruleID, original, translated, lineHint)
			return // over: only has one rule; no further GIS/GXL needed
		}
	}

	var (
		translated string
		trans      []Translation
		warns      []Warning
	)

	if expressionPositionKeys[parentKey] {
		translated, trans, warns = TranslateExprString(original, lineHint)
	} else {
		translated, trans, warns = TranslateGISString(original, lineHint)
	}

	if translated != original {
		node.Value = translated
		res.Translated += len(trans)
		for _, t := range trans {
			res.ByRule[t.RuleID]++
		}
	}

	for _, w := range warns {
		res.Warnings = append(res.Warnings, w)
		res.Deferred++
	}
}

func recordChange(res *Result, ruleID, before, after string, line int) {
	// Recorded implicitly via ByRule and Translated; detailed change log
	// could be added to Result in a future iteration.
	_ = before
	_ = after
	_ = line
	_ = ruleID
}

// VerifyClean re-greps for forbidden patterns in translated bytes.
// Returns a list of violation descriptions; empty slice = all clear.
func VerifyClean(filename string, src []byte) []string {
	var violations []string
	lines := strings.Split(string(src), "\n")

	for i, line := range lines {
		ln := i + 1
		// Skip comment lines.
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		// P1: {{ }} text/template syntax.
		if strings.Contains(line, "{{") {
			violations = append(violations, fmt.Sprintf("%s:%d: forbidden {{ }} pattern: %s", filename, ln, trimmed))
		}

		// P2: && or || in expression-position fields.
		if isExprLine(line) && (strings.Contains(line, "&&") || strings.Contains(line, "||")) {
			violations = append(violations, fmt.Sprintf("%s:%d: forbidden &&/|| in expression position: %s", filename, ln, trimmed))
		}

		// P3: ! prefix in when: field (but not !=).
		if isWhenLine(line) {
			exprVal := extractFieldValue(line)
			if reNegation.MatchString(exprVal) {
				violations = append(violations, fmt.Sprintf("%s:%d: forbidden ! prefix in when: %s", filename, ln, trimmed))
			}
		}

		// P4: infix contains in expression-position fields (not str./list.contains).
		if isExprLine(line) && reContainsInfix.MatchString(line) {
			violations = append(violations, fmt.Sprintf("%s:%d: forbidden infix contains in expression position: %s", filename, ln, trimmed))
		}

		// P5: $.path jq-style.
		if strings.Contains(line, "$.") {
			violations = append(violations, fmt.Sprintf("%s:%d: forbidden $. jq-style path: %s", filename, ln, trimmed))
		}
	}
	return violations
}

func isExprLine(line string) bool {
	for key := range expressionPositionKeys {
		if strings.Contains(line, key+":") {
			return true
		}
	}
	return false
}

func isWhenLine(line string) bool {
	return strings.Contains(line, "when:")
}

func extractFieldValue(line string) string {
	if idx := strings.Index(line, ":"); idx >= 0 {
		return strings.TrimSpace(line[idx+1:])
	}
	return line
}
