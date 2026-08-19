package engine

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"

	enginepkg "github.com/ormasoftchile/gert/pkg/engine"
	"github.com/ormasoftchile/gert/pkg/trace"
)

func TestEngine_UnhandledFailureStopsBeforeLaterToolsAndBranches(t *testing.T) {
	traceWriter := &fakeTraceWriter{}
	registry := newFakeExecutorRegistry()
	registry.Register("failing-tool", &failingExecutor{err: errors.New("first tool failed")})

	var laterToolCalls, branchCalls int
	registry.Register("later-tool", stepExecutorFunc(func(_ context.Context, step enginepkg.ResolvedStep, _ map[string]any) (*enginepkg.StepResult, error) {
		laterToolCalls++
		return &enginepkg.StepResult{StepID: step.ID, Status: enginepkg.StepStatusCompleted}, nil
	}))
	registry.Register("branch", stepExecutorFunc(func(_ context.Context, step enginepkg.ResolvedStep, _ map[string]any) (*enginepkg.StepResult, error) {
		branchCalls++
		return &enginepkg.StepResult{StepID: step.ID, Status: enginepkg.StepStatusCompleted}, nil
	}))

	cfg := makeTestConfig()
	cfg.Executors = registry
	cfg.TraceWriter = traceWriter
	handle, err := New(cfg).Start(context.Background(), enginepkg.ValidatedForTest(makeTestPlan(
		enginepkg.ResolvedStep{ID: "get-incident", Kind: "failing-tool", Spec: &cliStepSpec{}},
		enginepkg.ResolvedStep{ID: "recommend-tsg", Kind: "later-tool", Spec: &cliStepSpec{}},
		enginepkg.ResolvedStep{ID: "fallback-branch", Kind: "branch", Spec: &cliStepSpec{}},
	)), enginepkg.RunOptions{})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	if _, err := handle.Next(context.Background()); err == nil {
		t.Fatal("unhandled failed step must terminate the run")
	}
	if state := handle.State(); state.Status != enginepkg.RunStatusFailed {
		t.Fatalf("run status = %s, want failed", state.Status)
	}
	if _, err := handle.Next(context.Background()); !errors.Is(err, io.EOF) {
		t.Fatalf("Next after terminal failure = %v, want EOF", err)
	}
	if laterToolCalls != 0 || branchCalls != 0 {
		t.Fatalf("later tool calls = %d, branch calls = %d; neither may execute", laterToolCalls, branchCalls)
	}

	for _, event := range traceWriter.collect() {
		if event.Kind == trace.EventKindRunCompleted {
			t.Fatal("failed run emitted run/completed")
		}
		if event.Kind == trace.EventKindStepStarted {
			var payload struct {
				StepID string `json:"step_id"`
			}
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				t.Fatalf("unmarshal step/started payload: %v", err)
			}
			if payload.StepID != "get-incident" {
				t.Fatalf("later step/started emitted for %q", payload.StepID)
			}
		}
	}
}
