package migrateexpr

import (
	"strings"
	"testing"
)

// ── E-001/E-002/E-003: GIS interpolation ────────────────────────────────────

func TestTranslateGISString_SimpleVar(t *testing.T) {
	got, trans, warns := TranslateGISString("hello {{ .name }}", 1)
	assertEqual(t, "hello ${name}", got)
	assertRule(t, trans, "E-001")
	assertNoWarns(t, warns)
}

func TestTranslateGISString_MultiVar(t *testing.T) {
	got, _, _ := TranslateGISString("{{ .host }}:{{ .port }}", 1)
	assertEqual(t, "${host}:${port}", got)
}

func TestTranslateGISString_GDPPath(t *testing.T) {
	// E-002: dot-separated path.
	got, trans, _ := TranslateGISString("region={{ .config.region }}", 1)
	assertEqual(t, "region=${config.region}", got)
	assertRule(t, trans, "E-001") // same rule ID covers paths
}

func TestTranslateGISString_ArrayIndex(t *testing.T) {
	// E-003: array index.
	got, _, _ := TranslateGISString("item={{ .items[0] }}", 1)
	assertEqual(t, "item=${items[0]}", got)
}

func TestTranslateGISString_DeepPath(t *testing.T) {
	got, _, _ := TranslateGISString("{{ .a.b.c }}", 1)
	assertEqual(t, "${a.b.c}", got)
}

func TestTranslateGISString_LiteralDollarPlusTemplate(t *testing.T) {
	// RISK-005: ${{ .amount }} — the $ is a raw char in GIS, then ${amount} is interpolation.
	// After E-001: $${amount} — raw dollar + interpolation block.
	got, _, _ := TranslateGISString("Amount: ${{ .amount }}", 1)
	assertEqual(t, "Amount: $${amount}", got)
}

// ── E-009 / OI-GIS-01: legacy $${...} escape ────────────────────────────────

func TestTranslateGISString_OldEscape(t *testing.T) {
	// Pre-existing $${literal} in source (old-style escape for literal ${)
	// must become \${literal} BEFORE E-001 converts {{ .var }}.
	got, trans, _ := TranslateGISString("text $${literal} more", 1)
	assertEqual(t, `text \${literal} more`, got)
	assertRule(t, trans, "E-009")
}

func TestTranslateGISString_OldEscapeNotConfusedWithRISK005(t *testing.T) {
	// ${{ .var }} — the ${{ pattern is dollar + double-brace (text/template),
	// NOT dollar + dollar + brace. OI-GIS-01 must NOT fire; only E-001 fires.
	got, _, warns := TranslateGISString("Amount: ${{ .amount }}", 1)
	assertEqual(t, "Amount: $${amount}", got)
	assertNoWarns(t, warns)
}

// ── E-010: template pipe — warn only ────────────────────────────────────────

func TestTranslateGISString_TemplatePipeWarn(t *testing.T) {
	got, _, warns := TranslateGISString(`{{ .env | default "dev" }}`, 5)
	// Value should NOT be auto-translated.
	if got != `{{ .env | default "dev" }}` {
		t.Errorf("E-010: expected no auto-translation, got %q", got)
	}
	if len(warns) == 0 {
		t.Fatal("E-010: expected a warning for template pipe, got none")
	}
	if warns[0].RuleID != "E-010" {
		t.Errorf("E-010: expected warn RuleID E-010, got %q", warns[0].RuleID)
	}
	if warns[0].Line != 5 {
		t.Errorf("E-010: expected line 5, got %d", warns[0].Line)
	}
}

// ── E-011: {{ now }} → ${now()} ─────────────────────────────────────────────

func TestTranslateGISString_Now(t *testing.T) {
	got, trans, _ := TranslateGISString(`"{{ now }}"`, 10)
	assertEqual(t, `"${now()}"`, got)
	assertRule(t, trans, "E-011")
}

func TestTranslateGISString_NowInSentence(t *testing.T) {
	got, _, _ := TranslateGISString("Completed at {{ now }}", 1)
	assertEqual(t, "Completed at ${now()}", got)
}

// ── E-011b: {{ FUNCNAME }} (not now) — warn only ─────────────────────────────

func TestTranslateGISString_UnknownFuncWarn(t *testing.T) {
	// E-011b: {{ rand }} has no dot and is not "now" — should warn.
	_, _, warns := TranslateGISString("{{ rand }}", 3)
	found := false
	for _, w := range warns {
		if w.RuleID == "E-011b" {
			found = true
		}
	}
	if !found {
		t.Error("E-011b: expected warning for unknown function call, got none")
	}
}

func TestTranslateGISString_UnknownFuncWithArgs_Warn(t *testing.T) {
	// E-011b: {{ printf "%s" value }} — function call with arguments should warn.
	_, _, warns := TranslateGISString(`{{ printf "%s" }}`, 3)
	found := false
	for _, w := range warns {
		if w.RuleID == "E-011b" {
			found = true
		}
	}
	if !found {
		t.Error("E-011b: expected warning for function call with args, got none")
	}
}

// ── E-004/E-005: boolean operators in expression position ────────────────────

func TestTranslateExprString_And(t *testing.T) {
	got, trans, _ := TranslateExprString("error_rate < 0.01 && p95_latency < 0.2", 1)
	assertEqual(t, "error_rate < 0.01 and p95_latency < 0.2", got)
	assertRule(t, trans, "E-004")
}

func TestTranslateExprString_Or(t *testing.T) {
	got, trans, _ := TranslateExprString("error_rate > 0.05 || p95_latency > 0.5", 1)
	assertEqual(t, "error_rate > 0.05 or p95_latency > 0.5", got)
	assertRule(t, trans, "E-005")
}

func TestTranslateExprString_AndOr_Combined(t *testing.T) {
	got, _, _ := TranslateExprString("a && b || c", 1)
	// && → and first, then || → or
	assertEqual(t, "a and b or c", got)
}

// ── E-006: negation in expression position ───────────────────────────────────

func TestTranslateExprString_Negation(t *testing.T) {
	got, trans, _ := TranslateExprString("!acknowledged", 1)
	assertEqual(t, "not acknowledged", got)
	assertRule(t, trans, "E-006")
}

func TestTranslateExprString_NegationInCompound(t *testing.T) {
	got, _, _ := TranslateExprString(`db_count == "0" && !analytics_still_exists`, 1)
	assertEqual(t, `db_count == "0" and not analytics_still_exists`, got)
}

func TestTranslateExprString_NotEqualUntouched(t *testing.T) {
	// != must NOT be rewritten to "not =".
	got, _, _ := TranslateExprString("status != 200", 1)
	assertEqual(t, "status != 200", got)
}

// ── E-007: contains infix in expression position ─────────────────────────────

func TestTranslateExprString_ContainsSingleString(t *testing.T) {
	got, trans, _ := TranslateExprString(`pod_json contains "OOMKilled"`, 1)
	assertEqual(t, `str.contains(pod_json, "OOMKilled")`, got)
	assertRule(t, trans, "E-007")
}

func TestTranslateExprString_ContainsWithOr(t *testing.T) {
	got, _, _ := TranslateExprString(
		`pod_json contains "CrashLoopBackOff" || pod_json contains "ImagePullBackOff"`, 1)
	assertEqual(t,
		`str.contains(pod_json, "CrashLoopBackOff") or str.contains(pod_json, "ImagePullBackOff")`,
		got)
}

func TestTranslateExprString_AlreadyStrContainsUntouched(t *testing.T) {
	// E-007 guard: already-translated str.contains must not be double-processed.
	in := `str.contains(pod_json, "OOMKilled")`
	got, _, _ := TranslateExprString(in, 1)
	assertEqual(t, in, got)
}

// ── E-008: iterate.over jq-style path ────────────────────────────────────────

func TestTranslateOverValue_JQRoot(t *testing.T) {
	got, ruleID, changed := TranslateOverValue("$.services")
	assertEqual(t, "services", got)
	assertEqual(t, "E-008", ruleID)
	if !changed {
		t.Error("E-008: expected changed=true")
	}
}

func TestTranslateOverValue_AlreadyBare(t *testing.T) {
	got, _, changed := TranslateOverValue("services")
	assertEqual(t, "services", got)
	if changed {
		t.Error("E-008: expected changed=false for bare identifier")
	}
}

func TestTranslateOverValue_JQRootWithPath(t *testing.T) {
	got, _, changed := TranslateOverValue("$.items.list")
	assertEqual(t, "items.list", got)
	if !changed {
		t.Error("E-008: expected changed=true for $.items.list")
	}
}

// ── GIS rules NOT applied to expression positions by mistake ─────────────────

func TestTranslateExprString_NoGISOnNonString(t *testing.T) {
	// Shell && in a command arg should never reach TranslateExprString —
	// only expression-position keys go through that path.
	// This test verifies the rule itself doesn't accidentally fire on an
	// expression-position value that has no && (should be identity).
	in := "amount > 50000"
	got, _, _ := TranslateExprString(in, 1)
	assertEqual(t, in, got)
}

// ── helpers ──────────────────────────────────────────────────────────────────

func assertEqual(t *testing.T, want, got string) {
	t.Helper()
	if want != got {
		t.Errorf("\nwant: %q\n got: %q", want, got)
	}
}

func assertRule(t *testing.T, trans []Translation, ruleID string) {
	t.Helper()
	for _, tr := range trans {
		if tr.RuleID == ruleID {
			return
		}
	}
	ids := make([]string, len(trans))
	for i, tr := range trans {
		ids[i] = tr.RuleID
	}
	t.Errorf("expected rule %q in translations %v", ruleID, strings.Join(ids, ", "))
}

func assertNoWarns(t *testing.T, warns []Warning) {
	t.Helper()
	if len(warns) > 0 {
		t.Errorf("expected no warnings, got %d: %+v", len(warns), warns)
	}
}
