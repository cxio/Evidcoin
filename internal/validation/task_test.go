package validation

import (
	"testing"
	"time"

	"github.com/cxio/evidcoin/pkg/types"
)

func TestTaskKindConstants(t *testing.T) {
	tests := []struct {
		kind TaskKind
		want TaskKind
	}{
		{TaskFullValidation, 1},
		{TaskReview1, 2},
		{TaskReview2, 3},
	}
	for _, tc := range tests {
		if tc.kind != tc.want {
			t.Errorf("TaskKind value: got %d, want %d", tc.kind, tc.want)
		}
	}
}

func TestVerdictConstants(t *testing.T) {
	tests := []struct {
		v    Verdict
		want Verdict
	}{
		{VerdictLegal, 1},
		{VerdictIllegal, 2},
		{VerdictRejected, 3},
		{VerdictError, 4},
	}
	for _, tc := range tests {
		if tc.v != tc.want {
			t.Errorf("Verdict value: got %d, want %d", tc.v, tc.want)
		}
	}
}

func TestTaskConstruction(t *testing.T) {
	now := time.Now()
	txID := types.TxID{0xAB}
	task := Task{
		TxID:       txID,
		Kind:       TaskFullValidation,
		AssignedAt: now,
		TxData:     []byte{0x01, 0x02, 0x03},
	}
	if task.TxID != txID {
		t.Errorf("Task.TxID: got %x, want %x", task.TxID, txID)
	}
	if task.Kind != TaskFullValidation {
		t.Errorf("Task.Kind: got %d, want %d", task.Kind, TaskFullValidation)
	}
	if !task.AssignedAt.Equal(now) {
		t.Errorf("Task.AssignedAt: got %v, want %v", task.AssignedAt, now)
	}
	if len(task.TxData) != 3 {
		t.Errorf("Task.TxData len: got %d, want 3", len(task.TxData))
	}
}

func TestTaskResultConstruction(t *testing.T) {
	txID := types.TxID{0x01}
	r := TaskResult{
		TxID:        txID,
		ValidatorID: "val-1",
		Verdict:     VerdictIllegal,
		Reason:      "invalid signature",
	}
	if r.TxID != txID {
		t.Errorf("TaskResult.TxID mismatch")
	}
	if r.ValidatorID != "val-1" {
		t.Errorf("TaskResult.ValidatorID: got %q, want %q", r.ValidatorID, "val-1")
	}
	if r.Verdict != VerdictIllegal {
		t.Errorf("TaskResult.Verdict: got %d, want %d", r.Verdict, VerdictIllegal)
	}
}
