package cmd

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ormasoftchile/gert/internal/migrateexpr"
)

type migrateExprFlags struct {
	dryRun bool
	diff   bool
	report string
	strict bool
}

func migrateExprCmd() *cobra.Command {
	var flags migrateExprFlags

	cmd := &cobra.Command{
		Use:   "migrate-expr [flags] <path>...",
		Short: "Migrate legacy expression syntax to GXL/GIS/GCP",
		Long: `migrate-expr rewrites YAML runbook files from legacy text/template and
expr-lang syntax to the GERT-native expression languages GXL, GIS, and GCP.

Translation rules applied (per Stream D audit):
  E-001..003  {{ .var }}, {{ .a.b }}, {{ .a[0] }}  →  ${var}, ${a.b}, ${a[0]}
  E-004       && in expression position             →  and
  E-005       || in expression position             →  or
  E-006       !X / !(...) in expression position    →  not X / not (...)
  E-007       X contains Y in expression position    →  str/list.contains or WARN
  E-008       over: "$.IDENT"                       →  over: IDENT
  E-009/OI-GIS-01  legacy $${...} escape            →  \${...}
  E-010       {{ .x | default "v" }}               →  WARN (requires inputs block)
  E-011       {{ now }}                             →  ${now()}
  E-011b      {{ FUNCNAME }}  (other functions)     →  WARN (deferred)

Expression-position fields (GXL required): when, condition, until
GIS-interpolation fields: all other string-valued YAML scalars

Examples:
  gert migrate-expr runbooks/            # rewrite all YAML files under dir
  gert migrate-expr --dry-run schema.yaml
  gert migrate-expr --dry-run --diff schema.yaml
  gert migrate-expr --strict --report out.json runbooks/
`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// With --dry-run the diff default is true; otherwise false.
			// The user may override explicitly.
			if flags.dryRun && !cmd.Flags().Changed("diff") {
				flags.diff = true
			}
			return runMigrateExpr(args, flags)
		},
	}

	cmd.Flags().BoolVar(&flags.dryRun, "dry-run", false,
		"show planned changes without writing files")
	cmd.Flags().BoolVar(&flags.diff, "diff", false,
		"print unified diffs (default true when --dry-run is set)")
	cmd.Flags().StringVar(&flags.report, "report", "",
		"write a per-file JSON report of patterns translated, counts, and deferrals")
	cmd.Flags().BoolVar(&flags.strict, "strict", false,
		"exit non-zero if any pattern cannot be automatically translated")

	return cmd
}

type fileReport struct {
	File       string                `json:"file"`
	Translated int                   `json:"translated"`
	Deferred   int                   `json:"deferred"`
	Warnings   []migrateexpr.Warning `json:"warnings,omitempty"`
	Rules      map[string]int        `json:"rules"`
}

// runMigrateExpr is the entry point for the subcommand.
func runMigrateExpr(paths []string, flags migrateExprFlags) error {
	var files []string
	for _, p := range paths {
		collected, err := collectYAMLFiles(p)
		if err != nil {
			return fmt.Errorf("collecting files under %q: %w", p, err)
		}
		files = append(files, collected...)
	}

	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "no YAML files found")
		return nil
	}

	var reports []fileReport
	hasDeferred := false
	totalTranslated := 0

	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("reading %s: %w", f, err)
		}

		result, err := migrateexpr.TranslateFile(src)
		if err != nil {
			// Non-fatal: report and continue.
			fmt.Fprintf(os.Stderr, "WARN: %s: %v\n", f, err)
			continue
		}

		if len(result.Warnings) > 0 {
			for _, w := range result.Warnings {
				fmt.Fprintf(os.Stderr, "WARN [%s:%d] %s: %s\n",
					f, w.Line, w.RuleID, w.Message)
			}
			hasDeferred = true
		}

		rep := fileReport{
			File:       f,
			Translated: result.Translated,
			Deferred:   result.Deferred,
			Warnings:   result.Warnings,
			Rules:      result.ByRule,
		}
		reports = append(reports, rep)
		totalTranslated += result.Translated

		if flags.dryRun {
			if flags.diff && string(src) != string(result.Output) {
				printUnifiedDiff(f, string(src), string(result.Output))
			}
		} else if string(src) != string(result.Output) {
			if err := os.WriteFile(f, result.Output, 0o644); err != nil {
				return fmt.Errorf("writing %s: %w", f, err)
			}
		}
	}

	// Run post-migration verification grep.
	fmt.Fprintln(os.Stderr, "\n=== Post-migration verification ===")
	remaining := verifyClean(files)
	for _, v := range remaining {
		fmt.Fprintln(os.Stderr, "REMAIN:", v)
	}
	if len(remaining) == 0 {
		fmt.Fprintln(os.Stderr, "All patterns clean. ✓")
	}

	// Summary.
	fmt.Fprintf(os.Stderr, "\nSummary: %d file(s), %d translation(s) applied\n",
		len(files), totalTranslated)

	// Write JSON report if requested.
	if flags.report != "" {
		b, err := json.MarshalIndent(reports, "", "  ")
		if err != nil {
			return fmt.Errorf("encoding report: %w", err)
		}
		if err := os.WriteFile(flags.report, append(b, '\n'), 0o644); err != nil {
			return fmt.Errorf("writing report %s: %w", flags.report, err)
		}
	}

	if flags.strict && hasDeferred {
		return fmt.Errorf("strict mode: %d deferred pattern(s) require manual review", countDeferred(reports))
	}

	return nil
}

// collectYAMLFiles returns all .yaml / .yml files under path (recursive if dir).
func collectYAMLFiles(path string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		if isYAML(path) {
			return []string{path}, nil
		}
		return nil, nil
	}

	var files []string
	err = filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && isYAML(p) {
			files = append(files, p)
		}
		return nil
	})
	return files, err
}

func isYAML(p string) bool {
	ext := strings.ToLower(filepath.Ext(p))
	return ext == ".yaml" || ext == ".yml"
}

// verifyClean re-greps for forbidden patterns after translation.
// Returns a list of violation descriptions; empty means all-clear.
func verifyClean(files []string) []string {
	var violations []string
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		vs := migrateexpr.VerifyClean(f, src)
		violations = append(violations, vs...)
	}
	return violations
}

func countDeferred(reports []fileReport) int {
	total := 0
	for _, r := range reports {
		total += r.Deferred
	}
	return total
}

// printUnifiedDiff prints a minimal unified diff between old and new content.
func printUnifiedDiff(filename, old, new string) {
	fmt.Printf("--- a/%s\n+++ b/%s\n", filename, filename)

	oldLines := strings.Split(old, "\n")
	newLines := strings.Split(new, "\n")

	// Simple line-by-line diff with 3 lines of context.
	const ctx = 3
	var changedOld []int
	maxLen := len(oldLines)
	if len(newLines) > maxLen {
		maxLen = len(newLines)
	}
	for i := 0; i < maxLen; i++ {
		ov, nv := "", ""
		if i < len(oldLines) {
			ov = oldLines[i]
		}
		if i < len(newLines) {
			nv = newLines[i]
		}
		if ov != nv {
			changedOld = append(changedOld, i)
		}
	}

	// Build hunk ranges with context.
	printed := make(map[int]bool)
	for _, ci := range changedOld {
		lo := ci - ctx
		if lo < 0 {
			lo = 0
		}
		hi := ci + ctx
		if hi >= maxLen {
			hi = maxLen - 1
		}
		hunkStart := lo
		// Check if we need a header.
		if !printed[lo-1] {
			fmt.Printf("@@ -%d +%d @@\n", hunkStart+1, hunkStart+1)
		}
		for i := lo; i <= hi; i++ {
			if printed[i] {
				continue
			}
			ov, nv := "", ""
			if i < len(oldLines) {
				ov = oldLines[i]
			}
			if i < len(newLines) {
				nv = newLines[i]
			}
			if ov == nv {
				fmt.Printf(" %s\n", ov)
			} else {
				if i < len(oldLines) {
					fmt.Printf("-%s\n", ov)
				}
				if i < len(newLines) {
					fmt.Printf("+%s\n", nv)
				}
			}
			printed[i] = true
		}
	}
}
