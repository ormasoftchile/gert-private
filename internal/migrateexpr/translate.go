// Package migrateexpr implements the translation engine for gert migrate-expr.
//
// Translation rules (Stream D audit / Stream E spec):
//
//	E-001  {{ .IDENT }}            →  ${IDENT}              (GIS, all string positions)
//	E-002  {{ .A.B }}              →  ${A.B}                (GDP path)
//	E-003  {{ .A[N] }}             →  ${A[N]}               (array index)
//	E-004  &&  (expr-position)     →  and
//	E-005  ||  (expr-position)     →  or
//	E-006  !X / !(...)            →  not X / not (...)     (expr-position)
//	E-007  X contains Y           →  str/list.contains or WARN if ambiguous
//	E-008  over: "$.IDENT"        →  over: IDENT           (iterate.over)
//	E-009  legacy $${...} escape  →  \${...}               (OI-GIS-01)
//	E-010  {{ .x | default "v" }} →  WARN: requires inputs block
//	E-011  {{ now }}              →  ${now()}
//	E-011b {{ FUNCNAME }}         →  WARN: deferred
package migrateexpr

import (
	"fmt"
	"regexp"
	"strings"
)

// Warning records a pattern that could not be automatically translated.
type Warning struct {
	RuleID  string `json:"rule_id"`
	Line    int    `json:"line"`
	Before  string `json:"before"`
	Message string `json:"message"`
}

// Translation records a single automated substitution.
type Translation struct {
	RuleID string
	Before string
	After  string
}

// expressionPositionKeys are YAML field names whose values are GXL boolean
// expressions. E-004..E-007 apply only to these fields.
var expressionPositionKeys = map[string]bool{
	"when":      true,
	"condition": true,
	"until":     true,
}

// iterateOverKey is the field name whose value is a GCP iteration source.
// E-008 applies when this key appears as a child of an "iterate" mapping.
const iterateOverKey = "over"

// ── compiled regexps ────────────────────────────────────────────────────────

var (
	// E-009 / OI-GIS-01: legacy $${...} escape (pre-existing in source,
	// before any E-001 conversion; must run first).
	reOldEscape = regexp.MustCompile(`\$\$\{`)

	// E-010: template pipe — {{ .x | default "v" }} — warn only.
	reTemplatePipe = regexp.MustCompile(`\{\{[^}]*\|[^}]*\}\}`)

	// E-011: {{ now }} → ${now()}.
	reTemplateNow = regexp.MustCompile(`\{\{\s*now\s*\}\}`)

	// E-011b: any {{ FUNCNAME ... }} (no leading dot, not "now").
	// Matches function calls that may have arguments (no leading dot, no pipe).
	reTemplateFuncOnly = regexp.MustCompile(`\{\{\s*([A-Za-z_][A-Za-z0-9_]*)(?:\s[^{}]*)?\s*\}\}`)

	// E-001/E-002/E-003: {{ .IDENT[.IDENT|[N]]* }}.
	// The dot is mandatory; this excludes function calls.
	reTemplateVar = regexp.MustCompile(
		`\{\{\s*\.([A-Za-z_][A-Za-z0-9_]*(?:(?:\.[A-Za-z_][A-Za-z0-9_]*)|\[\d+\])*)\s*\}\}`,
	)

	// E-006: !IDENT — negation prefix in expression position.
	// Excludes != (next char after ! must be letter/underscore, not =).
	reNegation = regexp.MustCompile(`!([A-Za-z_][A-Za-z0-9_]*)`)

	// E-007: X contains "Y" or X contains Y — infix contains.
	// Only matches when contains is surrounded by whitespace (not str.contains / list.contains).
	reContainsInfix = regexp.MustCompile(
		`((?:"(?:[^"\\]|\\.)*")|[A-Za-z_][A-Za-z0-9_.]*)\s+contains\s+("(?:[^"\\]|\\.)*"|[A-Za-z_][A-Za-z0-9_.]*)`,
	)

	// E-008: $.IDENT jq-style root prefix.
	reJQRoot = regexp.MustCompile(`^\$\.([A-Za-z_][A-Za-z0-9_.]*)$`)
)

// TranslateGISString applies all GIS-position rules to a string value
// (E-009, E-010, E-011, E-001/E-002/E-003).
// It does NOT apply expression-position rules (E-004..E-007).
// Returns the translated string and any warnings produced.
func TranslateGISString(s string, lineHint int) (string, []Translation, []Warning) {
	var translations []Translation
	var warnings []Warning

	// E-009 / OI-GIS-01: convert legacy $${...} escape BEFORE E-001.
	// $${X} in pre-existing GIS meant "literal ${X}"; new canonical form is \${X}.
	// This must run before E-001 so that ${{ .var }} → $${var} (E-001 output)
	// is NOT misidentified as a legacy escape (since the source has ${{ not $${).
	if reOldEscape.MatchString(s) {
		before := s
		s = reOldEscape.ReplaceAllString(s, `\${`)
		translations = append(translations, Translation{"E-009", before, s})
	}

	// E-010: template pipe — warn only, do not auto-translate.
	if reTemplatePipe.MatchString(s) {
		// Collect all occurrences for targeted warning.
		for _, m := range reTemplatePipe.FindAllString(s, -1) {
			warnings = append(warnings, Warning{
				RuleID:  "E-010",
				Line:    lineHint,
				Before:  m,
				Message: fmt.Sprintf("template pipe %q cannot be auto-translated; add an inputs.<name>.default declaration and replace with ${<name>}", m),
			})
		}
		// Do not modify s; fall through to check for remaining {{ .var }} patterns.
	}

	// E-011: {{ now }} → ${now()}.
	if reTemplateNow.MatchString(s) {
		before := s
		s = reTemplateNow.ReplaceAllString(s, `${now()}`)
		translations = append(translations, Translation{"E-011", before, s})
	}

	// E-001/E-002/E-003: {{ .IDENT[.IDENT|[N]]* }} → ${IDENT...}.
	if reTemplateVar.MatchString(s) {
		before := s
		s = reTemplateVar.ReplaceAllStringFunc(s, func(m string) string {
			sub := reTemplateVar.FindStringSubmatch(m)
			if len(sub) < 2 {
				return m
			}
			return "${" + sub[1] + "}"
		})
		translations = append(translations, Translation{"E-001", before, s})
	}

	// E-011b: remaining {{ FUNCNAME }} (no dot) — warn only.
	if reTemplateFuncOnly.MatchString(s) {
		for _, m := range reTemplateFuncOnly.FindAllStringSubmatch(s, -1) {
			warnings = append(warnings, Warning{
				RuleID:  "E-011b",
				Line:    lineHint,
				Before:  m[0],
				Message: fmt.Sprintf("template function call %q has no automatic translation; migrate manually", m[0]),
			})
		}
	}

	return s, translations, warnings
}

// TranslateExprString applies all expression-position rules to a GXL value
// (E-004, E-005, E-006, E-007), plus all GIS rules via TranslateGISString.
// Expression positions are: when, condition, until.
func TranslateExprString(s string, lineHint int) (string, []Translation, []Warning) {
	// First, apply GIS rules (E-009, E-011, E-001..E-003).
	s, trans, warns := TranslateGISString(s, lineHint)

	// E-007: infix contains → str.contains(X, Y) or list.contains(X, Y).
	// Ambiguous subjects are left untouched and reported for manual migration.
	if reContainsInfix.MatchString(s) {
		var containsTrans []Translation
		var containsWarns []Warning
		s, containsTrans, containsWarns = translateContainsInfix(s, lineHint)
		trans = append(trans, containsTrans...)
		warns = append(warns, containsWarns...)
	}

	// E-006: !X / !(...) → not X / not (...), preserving != and string literals.
	if hasLegacyNegation(s) {
		before := s
		s = translateNegation(s)
		trans = append(trans, Translation{"E-006", before, s})
	}

	// E-005: || → or.
	if strings.Contains(s, "||") {
		before := s
		s = strings.ReplaceAll(s, "||", "or")
		// Normalise extra spaces around newly inserted keyword.
		s = normaliseSpaces(s)
		trans = append(trans, Translation{"E-005", before, s})
	}

	// E-004: && → and.
	if strings.Contains(s, "&&") {
		before := s
		s = strings.ReplaceAll(s, "&&", "and")
		s = normaliseSpaces(s)
		trans = append(trans, Translation{"E-004", before, s})
	}

	return s, trans, warns
}

// TranslateOverValue applies E-008: strips the jq-style $.IDENT root prefix
// from an iterate.over field value.
// Returns (translatedValue, ruleID, changed).
func TranslateOverValue(s string) (string, string, bool) {
	if m := reJQRoot.FindStringSubmatch(s); len(m) == 2 {
		return m[1], "E-008", true
	}
	return s, "", false
}

func translateContainsInfix(s string, lineHint int) (string, []Translation, []Warning) {
	var translations []Translation
	var warnings []Warning
	offset := 0

	for offset < len(s) {
		idx := reContainsInfix.FindStringSubmatchIndex(s[offset:])
		if idx == nil {
			break
		}
		for i := range idx {
			if idx[i] >= 0 {
				idx[i] += offset
			}
		}

		if idx[0] > 0 && s[idx[0]-1] == '.' {
			offset = idx[1]
			continue
		}

		subject := s[idx[2]:idx[3]]
		object := s[idx[4]:idx[5]]
		kind := classifyContainsSubject(subject)
		if kind == "" {
			warnings = append(warnings, Warning{
				RuleID:  "E-007",
				Line:    lineHint,
				Before:  s[idx[0]:idx[1]],
				Message: fmt.Sprintf("infix contains subject %q is ambiguous (string vs list); migrate manually to str.contains(...) or list.contains(...)", subject),
			})
			offset = idx[1]
			continue
		}

		before := s
		replacement := kind + ".contains(" + subject + ", " + object + ")"
		s = s[:idx[0]] + replacement + s[idx[5]:]
		translations = append(translations, Translation{"E-007", before, s})
		offset = idx[0] + len(replacement)
	}

	return s, translations, warnings
}

func classifyContainsSubject(subject string) string {
	trimmed := strings.TrimSpace(subject)
	if strings.HasPrefix(trimmed, `"`) && strings.HasSuffix(trimmed, `"`) {
		return "str"
	}

	last := trimmed
	if dot := strings.LastIndex(last, "."); dot >= 0 {
		last = last[dot+1:]
	}
	last = strings.ToLower(last)

	stringSuffixes := []string{"_json", "_text", "_stdout", "_stderr", "_body", "_message", "_log", "_logs", "_output"}
	for _, suffix := range stringSuffixes {
		if strings.HasSuffix(last, suffix) {
			return "str"
		}
	}
	if last == "stdout" || last == "stderr" || last == "body" || last == "message" || last == "output" {
		return "str"
	}

	listSuffixes := []string{"_list", "_ids", "_items", "_names", "_services", "_pods", "_nodes"}
	for _, suffix := range listSuffixes {
		if strings.HasSuffix(last, suffix) {
			return "list"
		}
	}
	if last == "items" || last == "services" || last == "pods" || last == "nodes" {
		return "list"
	}

	return ""
}

func hasLegacyNegation(s string) bool {
	return scanNegation(s, false) != s
}

func translateNegation(s string) string {
	return scanNegation(s, true)
}

func scanNegation(s string, replace bool) string {
	var b strings.Builder
	inSingle := false
	inDouble := false
	escaped := false
	changed := false

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inDouble {
			b.WriteByte(ch)
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
			} else if ch == '"' {
				inDouble = false
			}
			continue
		}
		if inSingle {
			b.WriteByte(ch)
			if ch == '\'' {
				inSingle = false
			}
			continue
		}

		switch ch {
		case '"':
			inDouble = true
			b.WriteByte(ch)
		case '\'':
			inSingle = true
			b.WriteByte(ch)
		case '!':
			if i+1 < len(s) && s[i+1] == '=' {
				b.WriteByte(ch)
				continue
			}
			changed = true
			if replace {
				b.WriteString("not ")
			} else {
				return ""
			}
		default:
			b.WriteByte(ch)
		}
	}

	if !changed {
		return s
	}
	return b.String()
}

// normaliseSpaces collapses runs of more than one space into a single space,
// used after boolean operator substitution to clean up `a  or  b`.
func normaliseSpaces(s string) string {
	reMultiSpace := regexp.MustCompile(` {2,}`)
	return reMultiSpace.ReplaceAllString(s, " ")
}
