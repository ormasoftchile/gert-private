package serve

import (
	"context"
	"errors"
	"testing"

	"github.com/ormasoftchile/gert/pkg/engine"
)

type terminalFailureHandle struct {
	*fakeRunHandle
	nextCalls int
}

func (h *terminalFailureHandle) Next(_ context.Context) (*engine.StepResult, error) {
	h.nextCalls++
	h.mu.Lock()
	h.state.Status = engine.RunStatusFailed
	h.mu.Unlock()
	return nil, errors.New("unhandled step failure")
}

func TestAdvanceRunStopsOnTerminalEngineFailure(t *testing.T) {
	server := newTestServer(t)
	handle := &terminalFailureHandle{fakeRunHandle: newFakeRunHandle("failed-run")}
	entry := &RunEntry{ID: "failed-run", Handle: handle, State: engine.RunStatusRunning}
	server.registry.Add(entry)

	server.wg.Add(1)
	server.advanceRun(context.Background(), entry)

	if handle.nextCalls != 1 {
		t.Fatalf("advanceRun called Next %d times, want 1 after terminal failure", handle.nextCalls)
	}
	if entry.State != engine.RunStatusFailed {
		t.Fatalf("entry state = %s, want failed", entry.State)
	}
}
