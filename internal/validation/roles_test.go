package validation

import (
	"testing"
	"time"

	"github.com/cxio/evidcoin/pkg/types"
)

// stubManager 实现 Manager 接口，用于接口满足检查。
type stubManager struct{}

func (s *stubManager) SubmitTask(task Task) error                { return nil }
func (s *stubManager) AssignTasks(id string) ([]Task, error)     { return nil, nil }
func (s *stubManager) ReceiveResult(r TaskResult) error          { return nil }
func (s *stubManager) GuardianDeliveries(id string) []types.TxID { return nil }

// stubGuardian 实现 Guardian 接口，用于接口满足检查。
type stubGuardian struct{}

func (s *stubGuardian) ReceiveExternal(txData []byte) error                           { return nil }
func (s *stubGuardian) ForwardToManager(task Task) error                              { return nil }
func (s *stubGuardian) NotifyIllegal(txID types.TxID, sourceValidatorID string) error { return nil }

// stubValidator 实现 Validator 接口，用于接口满足检查。
type stubValidator struct{}

func (s *stubValidator) RequestTasks() ([]Task, error)                     { return nil, nil }
func (s *stubValidator) SubmitResult(r TaskResult) error                   { return nil }
func (s *stubValidator) ForwardValid(txID types.TxID, txData []byte) error { return nil }

// TestInterfaceSatisfaction 确认桩类型满足角色接口约束，防止接口变更后遗漏实现。
func TestInterfaceSatisfaction(t *testing.T) {
	var _ Manager = (*stubManager)(nil)
	var _ Guardian = (*stubGuardian)(nil)
	var _ Validator = (*stubValidator)(nil)
}

// stubTask 构造一个最简 Task，用于接口调用冒烟。
func TestManagerSubmitTask(t *testing.T) {
	m := &stubManager{}
	task := Task{
		TxID:       types.TxID{},
		Kind:       TaskFullValidation,
		AssignedAt: time.Now(),
		TxData:     []byte{0x01, 0x02},
	}
	if err := m.SubmitTask(task); err != nil {
		t.Fatalf("SubmitTask unexpected error: %v", err)
	}
}
