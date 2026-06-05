package migrateexpr

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Result is returned by TranslateFile.
type Result struct {
	Output     []byte
	Translated int
	Deferred   int
	Warnings   []Warning
	ByRule     map[string]int
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

	knownSymbols := map[string]bool{}
	if len(doc.Content) > 0 {
		knownSymbols = collectKnownSymbols(doc.Content[0])
	}

	// Walk the document node (wraps the root mapping/sequence).
	if len(doc.Content) > 0 {
		walkNode(doc.Content[0], "", false, res, knownSymbols)
	}

	// If the walker found no automated rewrites, preserve the input byte-for-byte.
	// This keeps already-migrated fixtures idempotent and avoids yaml.v3 style churn.
	if res.Translated == 0 {
		res.Output = src
		return res, nil
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
func walkNode(node *yaml.Node, parentKey string, inIterate bool, res *Result, knownSymbols map[string]bool) {
	switch node.Kind {
	case yaml.DocumentNode:
		for _, child := range node.Content {
			walkNode(child, "", false, res, knownSymbols)
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
			walkNode(valNode, keyNode.Value, nextIterate, res, knownSymbols)
		}

	case yaml.SequenceNode:
		for _, child := range node.Content {
			walkNode(child, parentKey, inIterate, res, knownSymbols)
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
		translateScalar(node, parentKey, inIterate, res, knownSymbols)
	}
}

func collectKnownSymbols(root *yaml.Node) map[string]bool {
	symbols := map[string]bool{}
	collectDirectKeysForName(root, "inputs", symbols)
	collectDirectKeysForName(root, "captures", symbols)
	collectFieldNamesForName(root, "fields", symbols)
	return symbols
}

func collectDirectKeysForName(node *yaml.Node, name string, symbols map[string]bool) {
	if node == nil {
		return
	}
	switch node.Kind {
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valNode := node.Content[i+1]
			if keyNode.Kind == yaml.ScalarNode && keyNode.Value == name {
				addDirectMappingKeys(valNode, symbols)
			}
			collectDirectKeysForName(valNode, name, symbols)
		}
	case yaml.SequenceNode, yaml.DocumentNode:
		for _, child := range node.Content {
			collectDirectKeysForName(child, name, symbols)
		}
	}
}

func addDirectMappingKeys(node *yaml.Node, symbols map[string]bool) {
	if node == nil || node.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		if keyNode.Kind == yaml.ScalarNode {
			symbols[keyNode.Value] = true
		}
	}
}

func collectFieldNamesForName(node *yaml.Node, name string, symbols map[string]bool) {
	if node == nil {
		return
	}
	switch node.Kind {
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valNode := node.Content[i+1]
			if keyNode.Kind == yaml.ScalarNode && keyNode.Value == name {
				addNamedFieldSymbols(valNode, symbols)
			}
			collectFieldNamesForName(valNode, name, symbols)
		}
	case yaml.SequenceNode, yaml.DocumentNode:
		for _, child := range node.Content {
			collectFieldNamesForName(child, name, symbols)
		}
	}
}

func addNamedFieldSymbols(node *yaml.Node, symbols map[string]bool) {
	if node == nil || node.Kind != yaml.SequenceNode {
		return
	}
	for _, child := range node.Content {
		if child.Kind != yaml.MappingNode {
			continue
		}
		for i := 0; i+1 < len(child.Content); i += 2 {
			keyNode := child.Content[i]
			valNode := child.Content[i+1]
			if keyNode.Kind == yaml.ScalarNode && keyNode.Value == "name" && valNode.Kind == yaml.ScalarNode {
				symbols[valNode.Value] = true
			}
		}
	}
}

func protectDollarInterpolations(s string, knownSymbols map[string]bool) (string, map[string]string) {
	if len(knownSymbols) == 0 || !strings.Contains(s, "$${") {
		return s, nil
	}
	restore := map[string]string{}
	protected := s
	i := 0
	for symbol := range knownSymbols {
		needle := "$${" + symbol + "}"
		if !strings.Contains(protected, needle) {
			continue
		}
		token := fmt.Sprintf("__GERT_MIGRATE_EXPR_DOLLAR_%d__", i)
		i++
		protected = strings.ReplaceAll(protected, needle, token)
		restore[token] = needle
	}
	return protected, restore
}

func restoreDollarInterpolations(s string, restore map[string]string) string {
	for token, original := range restore {
		s = strings.ReplaceAll(s, token, original)
	}
	return s
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
func translateScalar(node *yaml.Node, parentKey string, inIterate bool, res *Result, knownSymbols map[string]bool) {
	original := node.Value
	lineHint := node.Line
	protected, restore := protectDollarInterpolations(original, knownSymbols)

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
		translated, trans, warns = TranslateExprString(protected, lineHint)
	} else {
		translated, trans, warns = TranslateGISString(protected, lineHint)
	}
	translated = restoreDollarInterpolations(translated, restore)
	for i := range trans {
		trans[i].Before = restoreDollarInterpolations(trans[i].Before, restore)
		trans[i].After = restoreDollarInterpolations(trans[i].After, restore)
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
			if hasLegacyNegation(exprVal) {
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
