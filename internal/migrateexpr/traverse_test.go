package migrateexpr

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTranslateFile_GISInterpolation exercises E-001/E-002/E-003 on a full file.
func TestTranslateFile_GISInterpolation(t *testing.T) {
	src, err := os.ReadFile("testdata/gis_interp_input.yaml")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	res, err := TranslateFile(src)
	if err != nil {
		t.Fatalf("TranslateFile: %v", err)
	}

	out := string(res.Output)

	// All {{ .var }} patterns should be gone.
	if strings.Contains(out, "{{") {
		t.Error("output still contains {{ }} patterns")
	}

	// Verify specific translations.
	mustContain(t, out, "${task_name}", "E-001 simple var")
	mustContain(t, out, "${host}", "E-001 simple var host")
	mustContain(t, out, "${port}", "E-001 simple var port")
	mustContain(t, out, "${config.region}", "E-002 GDP path")
	mustContain(t, out, "${items[0]}", "E-003 array index")

	// At least 6 translations.
	if res.Translated < 6 {
		t.Errorf("expected ≥6 translations, got %d", res.Translated)
	}

	// No deferred items.
	if res.Deferred != 0 {
		t.Errorf("expected 0 deferred, got %d", res.Deferred)
	}

	// Post-migration verify should find no violations.
	violations := VerifyClean("test", res.Output)
	if len(violations) > 0 {
		t.Errorf("VerifyClean found violations after translation: %v", violations)
	}
}

// TestTranslateFile_ExpressionRules exercises E-004..E-007 on a full file.
func TestTranslateFile_ExpressionRules(t *testing.T) {
	src, err := os.ReadFile("testdata/expr_rules_input.yaml")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	res, err := TranslateFile(src)
	if err != nil {
		t.Fatalf("TranslateFile: %v", err)
	}

	out := string(res.Output)

	// E-005: || in when: → or
	mustContain(t, out, "error_rate > 0.05 or p95_latency > 0.5", "E-005")
	mustNotContain(t, out, "||", "E-005 should remove all ||")

	// E-004: && in when: → and
	mustContain(t, out, "error_rate < 0.01 and p95_latency < 0.2", "E-004")
	mustNotContain(t, out, "&&", "E-004 should remove all &&")

	// E-006: !acknowledged → not acknowledged
	mustContain(t, out, "not acknowledged", "E-006")
	// Make sure != was not touched (no "not = " in output)
	mustNotContain(t, out, "not =", "E-006 must not touch !=")

	// E-007: contains infix → str.contains()
	mustContain(t, out, `str.contains(pod_json, "CrashLoopBackOff")`, "E-007 single")
	mustContain(t, out, `str.contains(pod_json, "ImagePullBackOff")`, "E-007 compound")
	mustContain(t, out, `str.contains(pod_json, "OOMKilled")`, "E-007 simple")

	// Post-migration verify.
	violations := VerifyClean("test", res.Output)
	if len(violations) > 0 {
		t.Errorf("VerifyClean found violations: %v", violations)
	}
}

// TestTranslateFile_MiscRules exercises E-008 and E-011.
func TestTranslateFile_MiscRules(t *testing.T) {
	src, err := os.ReadFile("testdata/misc_rules_input.yaml")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	res, err := TranslateFile(src)
	if err != nil {
		t.Fatalf("TranslateFile: %v", err)
	}

	out := string(res.Output)

	// E-008: over: "$.services" → over: services
	mustNotContain(t, out, `"$.services"`, "E-008 should remove jq-style $.services")
	mustNotContain(t, out, `$.`, "E-008 should remove all $. patterns")
	mustContain(t, out, "services", "E-008 bare identifier")

	// E-001: {{ .svc }} → ${svc} (in args)
	mustContain(t, out, "${svc}", "E-001 in iterate args")

	// E-011: {{ now }} → ${now()}
	mustContain(t, out, "${now()}", "E-011 now()")
	// Check that {{ now }} does not appear outside of comment lines in the output.
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(strings.TrimSpace(line), "#") {
			continue // skip comments
		}
		if strings.Contains(line, "{{ now }}") {
			t.Errorf("E-011: non-comment line still contains {{ now }}: %q", line)
		}
	}

	// Post-migration verify.
	violations := VerifyClean("test", res.Output)
	if len(violations) > 0 {
		t.Errorf("VerifyClean found violations: %v", violations)
	}
}

// TestTranslateFile_NoOp verifies that already-clean YAML is not modified.
func TestTranslateFile_NoOp(t *testing.T) {
	src := []byte(`apiVersion: runbook/v2
id: r12-approval-quorum
name: r12 Approval Quorum
inputs:
  threshold:
    type: integer
flow:
  - step:
      id: s1
      type: cli
      title: Check threshold
      command: echo
      args: ["${threshold}"]
      when: 'threshold > 5'
`)
	res, err := TranslateFile(src)
	if err != nil {
		t.Fatalf("TranslateFile: %v", err)
	}

	if res.Translated != 0 {
		t.Errorf("expected 0 translations for clean file, got %d", res.Translated)
	}

	violations := VerifyClean("test", res.Output)
	if len(violations) > 0 {
		t.Errorf("VerifyClean found violations in clean file: %v", violations)
	}
}

// TestTranslateFile_E010Warn verifies that template pipes produce warnings and
// leave the value untouched.
func TestTranslateFile_E010Warn(t *testing.T) {
	src := []byte(`apiVersion: runbook/v2
id: test-pipe
flow:
  - step:
      id: s1
      type: cli
      title: Env step
      command: echo
      args: ["{{ .env | default \"dev\" }}"]
`)
	res, err := TranslateFile(src)
	if err != nil {
		t.Fatalf("TranslateFile: %v", err)
	}

	// No auto-translation for E-010.
	if res.Translated != 0 {
		t.Errorf("E-010: expected 0 auto-translations, got %d", res.Translated)
	}

	// Warning must be emitted.
	found := false
	for _, w := range res.Warnings {
		if w.RuleID == "E-010" {
			found = true
		}
	}
	if !found {
		t.Error("E-010: expected warning in result")
	}

	if res.Deferred == 0 {
		t.Error("E-010: expected deferred count > 0")
	}
}

func TestTranslateFile_DogfoodMigratedRunbooksNoOp(t *testing.T) {
	root := filepath.Join("..", "..", "design", "gert", "testdata", "runbooks")
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".yaml" || ext == ".yml" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking migrated runbooks: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected migrated runbook YAML fixtures")
	}

	for _, file := range files {
		file := file
		t.Run(file, func(t *testing.T) {
			src, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("reading fixture: %v", err)
			}
			res, err := TranslateFile(src)
			if err != nil {
				t.Fatalf("TranslateFile: %v", err)
			}
			if res.Translated != 0 {
				t.Fatalf("expected already-migrated fixture to need 0 translations, got %d", res.Translated)
			}
			if res.Deferred != 0 {
				t.Fatalf("expected already-migrated fixture to need 0 deferrals, got %d: %+v", res.Deferred, res.Warnings)
			}
			if string(res.Output) != string(src) {
				t.Fatal("expected migrated fixture to remain byte-for-byte unchanged")
			}
		})
	}
}

// ── helpers ──────────────────────────────────────────────────────────────────

func mustContain(t *testing.T, s, substr, label string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("%s: output does not contain %q", label, substr)
	}
}

func mustNotContain(t *testing.T, s, substr, label string) {
	t.Helper()
	if strings.Contains(s, substr) {
		t.Errorf("%s: output unexpectedly contains %q", label, substr)
	}
}
