# Phase 7：组队校验框架 实现计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现组队校验框架——包括角色接口定义（Guard/Verifier/Dispatcher/Broadcaster）、消息类型、首领校验逻辑、冗余校验与扩展复核协议、组间反馈机制、铸造集成协议、以及 UTXO/UTCO 缓存接口。

**Architecture:** `internal/verification` 包，定义校验团队的角色接口、消息协议和核心验证逻辑。传输层由外部 P2P 库处理，本模块专注于数据结构和协议流程。依赖 `pkg/types`、`pkg/crypto` 和 `internal/tx`。

**Tech Stack:** Go 1.25+, pkg/types, pkg/crypto, internal/tx

---

## 前置依赖

本 Phase 假设 Phase 1、Phase 3 和 Phase 4 已完成，以下类型和函数已可用：

```go
// pkg/types
type Hash512 [64]byte              // SHA-512（64 字节）
type Hash384 [48]byte              // SHA3-384
type PubKeyHash [48]byte           // 公钥哈希（SHA3-384）
type OutputConfig byte             // 输出配置字节
const HashLen = 64
const Hash384Len = 48
const PubKeyHashLen = 48
const BlocksPerYear = 87661        // 每年区块数
const OutTypeCoin OutputConfig = 1          // 币金类型
const OutTypeCredit OutputConfig = 2        // 凭信类型
const OutTypeProof OutputConfig = 3         // 存证类型
const OutTypeMediator OutputConfig = 4      // 中介类型
const OutDestroy OutputConfig = 1 << 5      // 销毁标记
func (h Hash512) IsZero() bool
func (h Hash512) String() string
func (h Hash512) Equal(other Hash512) bool
func (p PubKeyHash) IsZero() bool

// pkg/crypto
func SHA512Sum(data []byte) types.Hash512
func SHA3_512Sum(data []byte) types.Hash512
func SHA3_384Sum(data []byte) types.Hash384

// internal/tx
type TxHeader struct { ... }
func (h *TxHeader) TxID() types.Hash512
type LeadInput struct { Year uint16; TxID types.Hash512; OutIndex uint16 }
type RestInput struct { ... }
type CoinOutput struct { ... }
type CreditOutput struct { ... }

// internal/utxo
type OutPoint struct { TxID types.Hash512; Index uint16 }
type UTXOEntry struct { ... }
```

> **注意：** 如果前置 Phase 的具体 API 与以上描述有差异，请在实现时以实际导出符号为准，适当调整本计划中的调用方式。

---

## Task 1: 消息类型定义 (internal/verification/message.go)

**Files:**
- Create: `internal/verification/message.go`
- Test: `internal/verification/message_test.go`

本 Task 定义校验团队内部和团队之间通信所需的消息类型枚举、`Message` 通用消息结构、`NodeID` 节点标识类型，以及各消息类型对应的 payload 结构体。

### Step 1: 写失败测试

创建 `internal/verification/message_test.go`：

```go
package verification

import (
	"testing"
	"time"
)

// --- MessageType 枚举值唯一性测试 ---

func TestMessageType_Unique(t *testing.T) {
	// 收集所有已定义的消息类型
	allTypes := []MessageType{
		MsgSubmitTx,
		MsgVerifyTask,
		MsgVerifyResult,
		MsgVerifiedTxSync,
		MsgBlockProof,
		MsgBlockSummary,
		MsgTxSync,
		MsgMintRequest,
		MsgMintInfo,
		MsgCoinbaseSubmit,
		MsgCheckRootData,
		MsgSignedCheckRoot,
		MsgBlockPublished,
		MsgInterTeamFeedback,
		MsgBlacklistNotify,
	}

	seen := make(map[MessageType]string)
	names := []string{
		"MsgSubmitTx",
		"MsgVerifyTask",
		"MsgVerifyResult",
		"MsgVerifiedTxSync",
		"MsgBlockProof",
		"MsgBlockSummary",
		"MsgTxSync",
		"MsgMintRequest",
		"MsgMintInfo",
		"MsgCoinbaseSubmit",
		"MsgCheckRootData",
		"MsgSignedCheckRoot",
		"MsgBlockPublished",
		"MsgInterTeamFeedback",
		"MsgBlacklistNotify",
	}

	for i, mt := range allTypes {
		if prev, ok := seen[mt]; ok {
			t.Errorf("duplicate MessageType value %d: %s and %s", mt, prev, names[i])
		}
		seen[mt] = names[i]
	}
}

// 测试消息类型值非零（0 保留为无效/未知）
func TestMessageType_NonZero(t *testing.T) {
	allTypes := []struct {
		name string
		val  MessageType
	}{
		{"MsgSubmitTx", MsgSubmitTx},
		{"MsgVerifyTask", MsgVerifyTask},
		{"MsgVerifyResult", MsgVerifyResult},
		{"MsgVerifiedTxSync", MsgVerifiedTxSync},
		{"MsgBlockProof", MsgBlockProof},
		{"MsgBlockSummary", MsgBlockSummary},
		{"MsgTxSync", MsgTxSync},
		{"MsgMintRequest", MsgMintRequest},
		{"MsgMintInfo", MsgMintInfo},
		{"MsgCoinbaseSubmit", MsgCoinbaseSubmit},
		{"MsgCheckRootData", MsgCheckRootData},
		{"MsgSignedCheckRoot", MsgSignedCheckRoot},
		{"MsgBlockPublished", MsgBlockPublished},
		{"MsgInterTeamFeedback", MsgInterTeamFeedback},
		{"MsgBlacklistNotify", MsgBlacklistNotify},
	}
	for _, tt := range allTypes {
		t.Run(tt.name, func(t *testing.T) {
			if tt.val == 0 {
				t.Errorf("%s should not be zero", tt.name)
			}
		})
	}
}

// 测试 MessageType.String() 返回可读名称
func TestMessageType_String(t *testing.T) {
	tests := []struct {
		mt   MessageType
		want string
	}{
		{MsgSubmitTx, "SubmitTx"},
		{MsgVerifyTask, "VerifyTask"},
		{MsgVerifyResult, "VerifyResult"},
		{MsgVerifiedTxSync, "VerifiedTxSync"},
		{MsgBlockProof, "BlockProof"},
		{MsgBlockSummary, "BlockSummary"},
		{MsgTxSync, "TxSync"},
		{MsgMintRequest, "MintRequest"},
		{MsgMintInfo, "MintInfo"},
		{MsgCoinbaseSubmit, "CoinbaseSubmit"},
		{MsgCheckRootData, "CheckRootData"},
		{MsgSignedCheckRoot, "SignedCheckRoot"},
		{MsgBlockPublished, "BlockPublished"},
		{MsgInterTeamFeedback, "InterTeamFeedback"},
		{MsgBlacklistNotify, "BlacklistNotify"},
		{MessageType(0), "Unknown(0)"},
		{MessageType(255), "Unknown(255)"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.mt.String()
			if got != tt.want {
				t.Errorf("MessageType(%d).String() = %q, want %q", tt.mt, got, tt.want)
			}
		})
	}
}

// --- NodeID 测试 ---

func TestNodeID_IsZero(t *testing.T) {
	var zero NodeID
	if !zero.IsZero() {
		t.Error("zero NodeID.IsZero() should return true")
	}

	nonZero := NodeID{0x01}
	if nonZero.IsZero() {
		t.Error("non-zero NodeID.IsZero() should return false")
	}
}

func TestNodeID_String(t *testing.T) {
	var id NodeID
	id[0] = 0xab
	id[1] = 0xcd

	s := id.String()
	// 应返回 64 字符的十六进制字符串
	if len(s) != NodeIDLen*2 {
		t.Errorf("NodeID.String() length = %d, want %d", len(s), NodeIDLen*2)
	}
	if s[:4] != "abcd" {
		t.Errorf("NodeID.String() prefix = %q, want %q", s[:4], "abcd")
	}
}

// --- Message 构造与字段测试 ---

func TestNewMessage(t *testing.T) {
	var sender NodeID
	sender[0] = 0x42

	payload := []byte("test payload data")
	msg := NewMessage(MsgSubmitTx, sender, payload)

	if msg.Type != MsgSubmitTx {
		t.Errorf("Type = %d, want %d", msg.Type, MsgSubmitTx)
	}
	if msg.SenderID != sender {
		t.Error("SenderID mismatch")
	}
	if string(msg.Payload) != string(payload) {
		t.Errorf("Payload = %q, want %q", msg.Payload, payload)
	}
	if msg.Timestamp == 0 {
		t.Error("Timestamp should be set automatically")
	}
	// 时间戳应在合理范围内（最近 1 秒）
	now := time.Now().UnixMilli()
	if msg.Timestamp < now-1000 || msg.Timestamp > now+1000 {
		t.Errorf("Timestamp %d not within 1s of now %d", msg.Timestamp, now)
	}
}

func TestNewMessage_NilPayload(t *testing.T) {
	msg := NewMessage(MsgBlockProof, NodeID{}, nil)
	if msg.Payload != nil {
		t.Error("nil payload should remain nil")
	}
	if msg.Type != MsgBlockProof {
		t.Errorf("Type = %d, want %d", msg.Type, MsgBlockProof)
	}
}

func TestMessage_Validate(t *testing.T) {
	validSender := NodeID{0x01}

	tests := []struct {
		name    string
		msg     *Message
		wantErr bool
	}{
		{
			name:    "valid message",
			msg:     &Message{Type: MsgSubmitTx, SenderID: validSender, Payload: []byte{0x01}, Timestamp: time.Now().UnixMilli()},
			wantErr: false,
		},
		{
			name:    "zero type",
			msg:     &Message{Type: 0, SenderID: validSender, Payload: []byte{0x01}, Timestamp: 1},
			wantErr: true,
		},
		{
			name:    "zero sender",
			msg:     &Message{Type: MsgSubmitTx, SenderID: NodeID{}, Payload: []byte{0x01}, Timestamp: 1},
			wantErr: true,
		},
		{
			name:    "zero timestamp",
			msg:     &Message{Type: MsgSubmitTx, SenderID: validSender, Payload: []byte{0x01}, Timestamp: 0},
			wantErr: true,
		},
		{
			name:    "nil payload allowed",
			msg:     &Message{Type: MsgBlockPublished, SenderID: validSender, Payload: nil, Timestamp: 1},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/verification/ -run "TestMessageType|TestNodeID|TestNewMessage|TestMessage"
```

预期输出：编译失败，`MessageType`、`NodeID`、`Message`、`NewMessage` 等未定义。

### Step 3: 写最小实现

创建 `internal/verification/message.go`：

```go
package verification

import (
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

// NodeIDLen 节点标识的字节长度。
const NodeIDLen = 32

// NodeID 节点唯一标识（32 字节）。
type NodeID [NodeIDLen]byte

// IsZero 检查 NodeID 是否全零。
func (n NodeID) IsZero() bool {
	for _, b := range n {
		if b != 0 {
			return false
		}
	}
	return true
}

// String 返回 NodeID 的十六进制字符串表示。
func (n NodeID) String() string {
	return hex.EncodeToString(n[:])
}

// MessageType 消息类型枚举（byte 类型）。
// 0 保留为无效/未知类型。
type MessageType byte

// 消息类型常量定义。
const (
	MsgSubmitTx          MessageType = 1  // 提交交易（Guard → Dispatcher）
	MsgVerifyTask        MessageType = 2  // 分配校验任务（Dispatcher → Verifier）
	MsgVerifyResult      MessageType = 3  // 校验结果（Verifier → Dispatcher）
	MsgVerifiedTxSync    MessageType = 4  // 已验证交易同步（Dispatcher → Broadcaster）
	MsgBlockProof        MessageType = 5  // 区块证明广播
	MsgBlockSummary      MessageType = 6  // 区块概要
	MsgTxSync            MessageType = 7  // 交易同步请求
	MsgMintRequest       MessageType = 8  // 铸造申请（Minter → Broadcaster）
	MsgMintInfo          MessageType = 9  // 铸造信息（Broadcaster → Minter）
	MsgCoinbaseSubmit    MessageType = 10 // 提交 Coinbase（Minter → Broadcaster）
	MsgCheckRootData     MessageType = 11 // 校验根数据（Broadcaster → Minter）
	MsgSignedCheckRoot   MessageType = 12 // 签名的 CheckRoot（Minter → Broadcaster）
	MsgBlockPublished    MessageType = 13 // 区块发布通知
	MsgInterTeamFeedback MessageType = 14 // 组间反馈
	MsgBlacklistNotify   MessageType = 15 // 黑名单通知
)

// messageTypeNames 消息类型名称映射。
var messageTypeNames = map[MessageType]string{
	MsgSubmitTx:          "SubmitTx",
	MsgVerifyTask:        "VerifyTask",
	MsgVerifyResult:      "VerifyResult",
	MsgVerifiedTxSync:    "VerifiedTxSync",
	MsgBlockProof:        "BlockProof",
	MsgBlockSummary:      "BlockSummary",
	MsgTxSync:            "TxSync",
	MsgMintRequest:       "MintRequest",
	MsgMintInfo:          "MintInfo",
	MsgCoinbaseSubmit:    "CoinbaseSubmit",
	MsgCheckRootData:     "CheckRootData",
	MsgSignedCheckRoot:   "SignedCheckRoot",
	MsgBlockPublished:    "BlockPublished",
	MsgInterTeamFeedback: "InterTeamFeedback",
	MsgBlacklistNotify:   "BlacklistNotify",
}

// String 返回消息类型的可读名称。
func (mt MessageType) String() string {
	if name, ok := messageTypeNames[mt]; ok {
		return name
	}
	return fmt.Sprintf("Unknown(%d)", mt)
}

// Message 通用消息结构，用于团队内部和团队之间的通信。
// 传输层（外部 P2P 库）负责消息的序列化和网络传输。
type Message struct {
	Type      MessageType // 消息类型
	Payload   []byte      // 消息载荷（具体内容由 Type 决定）
	SenderID  NodeID      // 发送者节点 ID
	Timestamp int64       // 发送时间戳（Unix 毫秒）
}

// 消息验证错误
var (
	ErrInvalidMsgType  = errors.New("invalid message type")
	ErrZeroSenderID    = errors.New("sender id is zero")
	ErrZeroTimestamp    = errors.New("message timestamp is zero")
)

// NewMessage 创建一条新消息，自动设置当前时间戳。
func NewMessage(msgType MessageType, sender NodeID, payload []byte) *Message {
	return &Message{
		Type:      msgType,
		Payload:   payload,
		SenderID:  sender,
		Timestamp: time.Now().UnixMilli(),
	}
}

// Validate 对消息执行基本字段验证。
func (m *Message) Validate() error {
	if m.Type == 0 {
		return ErrInvalidMsgType
	}
	if m.SenderID.IsZero() {
		return ErrZeroSenderID
	}
	if m.Timestamp == 0 {
		return ErrZeroTimestamp
	}
	return nil
}
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/verification/ -run "TestMessageType|TestNodeID|TestNewMessage|TestMessage"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/verification/message.go internal/verification/message_test.go
git commit -m "feat(verification): add message types, NodeID and Message struct"
```

---

## Task 2: 角色接口与辅助类型定义 (internal/verification/roles.go)

**Files:**
- Create: `internal/verification/roles.go`
- Test: `internal/verification/roles_test.go`

本 Task 定义校验团队四大角色的 Go 接口（Guard、Verifier、Dispatcher、Broadcaster），以及所有辅助结构体类型（TxEnvelope、VerifyTask、VerifyResult、铸造相关类型等）。

### Step 1: 写失败测试

创建 `internal/verification/roles_test.go`：

```go
package verification

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// --- TxEnvelope 构造测试 ---

func TestNewTxEnvelope(t *testing.T) {
	var txID types.Hash512
	txID[0] = 0xaa

	rawTx := []byte{0x01, 0x02, 0x03}
	env := NewTxEnvelope(txID, rawTx)

	if env.TxID != txID {
		t.Error("TxID mismatch")
	}
	if string(env.RawTx) != string(rawTx) {
		t.Errorf("RawTx = %x, want %x", env.RawTx, rawTx)
	}
	if env.ReceivedAt == 0 {
		t.Error("ReceivedAt should be set automatically")
	}
}

func TestNewTxEnvelope_Fields(t *testing.T) {
	var txID types.Hash512
	txID[63] = 0xff

	env := NewTxEnvelope(txID, []byte{0xab})

	// 新创建的信封输入项数组默认为空
	if len(env.CoinInputAmounts) != 0 {
		t.Errorf("CoinInputAmounts should be empty, got %d", len(env.CoinInputAmounts))
	}
	if env.LeaderAmount != 0 {
		t.Errorf("LeaderAmount should be 0, got %d", env.LeaderAmount)
	}
	if env.HasCreditDestroy {
		t.Error("HasCreditDestroy should be false by default")
	}
}

// --- VerifyResult 状态值测试 ---

func TestVerifyStatus_String(t *testing.T) {
	tests := []struct {
		status VerifyStatus
		want   string
	}{
		{StatusValid, "Valid"},
		{StatusInvalid, "Invalid"},
		{StatusOverloaded, "Overloaded"},
		{VerifyStatus(99), "Unknown(99)"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.status.String()
			if got != tt.want {
				t.Errorf("VerifyStatus(%d).String() = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestVerifyStatus_Values(t *testing.T) {
	// 确保三种状态值不同
	if StatusValid == StatusInvalid {
		t.Error("StatusValid and StatusInvalid should be different")
	}
	if StatusValid == StatusOverloaded {
		t.Error("StatusValid and StatusOverloaded should be different")
	}
	if StatusInvalid == StatusOverloaded {
		t.Error("StatusInvalid and StatusOverloaded should be different")
	}
}

// --- VerifyResult 构造测试 ---

func TestNewVerifyResult(t *testing.T) {
	var txID types.Hash512
	txID[0] = 0x01
	var verifier NodeID
	verifier[0] = 0x42

	result := NewVerifyResult(txID, verifier, StatusValid, "")
	if result.TxID != txID {
		t.Error("TxID mismatch")
	}
	if result.VerifierID != verifier {
		t.Error("VerifierID mismatch")
	}
	if result.Status != StatusValid {
		t.Errorf("Status = %d, want %d", result.Status, StatusValid)
	}
	if result.Reason != "" {
		t.Errorf("Reason = %q, want empty", result.Reason)
	}
}

func TestNewVerifyResult_Invalid(t *testing.T) {
	var txID types.Hash512
	txID[0] = 0x02

	result := NewVerifyResult(txID, NodeID{0x01}, StatusInvalid, "script execution failed")
	if result.Status != StatusInvalid {
		t.Errorf("Status = %d, want %d", result.Status, StatusInvalid)
	}
	if result.Reason != "script execution failed" {
		t.Errorf("Reason = %q, want %q", result.Reason, "script execution failed")
	}
}

// --- VerifyTask 构造测试 ---

func TestNewVerifyTask(t *testing.T) {
	var txID types.Hash512
	txID[0] = 0xcc

	env := NewTxEnvelope(txID, []byte{0x01})
	task := NewVerifyTask(env)

	if task.Envelope == nil {
		t.Fatal("Envelope should not be nil")
	}
	if task.Envelope.TxID != txID {
		t.Error("Envelope.TxID mismatch")
	}
	if task.AssignedAt == 0 {
		t.Error("AssignedAt should be set automatically")
	}
}

// --- OutPoint 类型测试（验证从 types 或 utxo 包引用） ---

func TestOutPoint_Construction(t *testing.T) {
	var txID types.Hash512
	txID[0] = 0xdd
	op := OutPoint{TxID: txID, Index: 7}

	if op.TxID != txID {
		t.Error("TxID mismatch")
	}
	if op.Index != 7 {
		t.Errorf("Index = %d, want 7", op.Index)
	}
}

func TestOutPoint_Key(t *testing.T) {
	var txID types.Hash512
	txID[0] = 0xee
	op := OutPoint{TxID: txID, Index: 3}
	key := op.Key()

	if key == "" {
		t.Error("Key() should not be empty")
	}

	// 相同的 OutPoint 应该有相同的 Key
	op2 := OutPoint{TxID: txID, Index: 3}
	if op.Key() != op2.Key() {
		t.Error("same OutPoint should produce same Key()")
	}

	// 不同的 OutPoint 应该有不同的 Key
	op3 := OutPoint{TxID: txID, Index: 4}
	if op.Key() == op3.Key() {
		t.Error("different OutPoint should produce different Key()")
	}
}

// --- 接口编译测试 ---

// 以下测试用 mock 类型确认接口定义可正确编译。
// 不测试行为逻辑，只确保接口方法签名正确。

type mockGuard struct{}

func (m *mockGuard) LeaderVerify(tx *TxEnvelope) (bool, error) { return true, nil }
func (m *mockGuard) SubmitToDispatcher(tx *TxEnvelope) error   { return nil }
func (m *mockGuard) ForwardToGuards(tx *TxEnvelope) error      { return nil }
func (m *mockGuard) IsBlacklisted(op OutPoint) bool            { return false }
func (m *mockGuard) AddBlacklist(op OutPoint) error            { return nil }

type mockVerifier struct{}

func (m *mockVerifier) RequestTask() (*VerifyTask, error)              { return nil, nil }
func (m *mockVerifier) Verify(task *VerifyTask) (*VerifyResult, error) { return nil, nil }
func (m *mockVerifier) DeliverToGuards(tx *TxEnvelope) error           { return nil }

type mockDispatcher struct{}

func (m *mockDispatcher) AcceptTx(tx *TxEnvelope) error                          { return nil }
func (m *mockDispatcher) AssignTask(verifierID NodeID) (*VerifyTask, error)       { return nil, nil }
func (m *mockDispatcher) CollectResult(result *VerifyResult) error                { return nil }
func (m *mockDispatcher) SyncVerified(broadcasterCh chan<- *TxEnvelope) error     { return nil }

type mockBroadcaster struct{}

func (m *mockBroadcaster) AcceptMintRequest(req *MintRequest) (*MintInfo, error)            { return nil, nil }
func (m *mockBroadcaster) AcceptCoinbase(cb *CoinbaseSubmission) (*CheckRootData, error)    { return nil, nil }
func (m *mockBroadcaster) AcceptSignedCheckRoot(sig *SignedCheckRoot) error                  { return nil }
func (m *mockBroadcaster) PublishBlock() error                                               { return nil }
func (m *mockBroadcaster) ReceiveBlock(proof *BlockProof) error                              { return nil }
func (m *mockBroadcaster) SyncMissingTxs(summary *BlockSummary) error                       { return nil }

// 测试 mock 类型满足接口约束
func TestInterfaces_Compile(t *testing.T) {
	var _ Guard = (*mockGuard)(nil)
	var _ Verifier = (*mockVerifier)(nil)
	var _ Dispatcher = (*mockDispatcher)(nil)
	var _ Broadcaster = (*mockBroadcaster)(nil)

	t.Log("all interfaces compile correctly")
}

// --- ServiceAddresses 测试 ---

func TestServiceAddresses_HasDepots(t *testing.T) {
	sa := ServiceAddresses{
		Depots:  types.PubKeyHash{0x01},
		Blockqs: types.PubKeyHash{},
		Stun:    types.PubKeyHash{},
	}
	if sa.Depots.IsZero() {
		t.Error("Depots should not be zero")
	}
	if !sa.Blockqs.IsZero() {
		t.Error("Blockqs should be zero")
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/verification/ -run "TestNewTxEnvelope|TestVerifyStatus|TestNewVerifyResult|TestNewVerifyTask|TestOutPoint|TestInterfaces|TestServiceAddresses"
```

预期输出：编译失败，`TxEnvelope`、`VerifyStatus`、`Guard`、`Verifier`、`Dispatcher`、`Broadcaster` 等未定义。

### Step 3: 写最小实现

创建 `internal/verification/roles.go`：

```go
package verification

import (
	"errors"
	"fmt"
	"time"

	"github.com/cxio/evidcoin/pkg/types"
)

// OutPoint 唯一标识一个交易输出。
// 此类型在 verification 包中本地定义，避免对 internal/utxo 包的循环依赖。
type OutPoint struct {
	TxID  types.Hash512 // 交易 ID
	Index uint16        // 输出序号
}

// Key 返回用作 map 键的字符串表示。
func (o OutPoint) Key() string {
	return fmt.Sprintf("%s:%d", o.TxID.String(), o.Index)
}

// VerifyStatus 校验结果状态枚举。
type VerifyStatus byte

// 校验结果状态常量。
const (
	StatusValid      VerifyStatus = 1 // 校验通过
	StatusInvalid    VerifyStatus = 2 // 校验失败
	StatusOverloaded VerifyStatus = 3 // 校验者过载，拒绝处理
)

// verifyStatusNames 状态名称映射。
var verifyStatusNames = map[VerifyStatus]string{
	StatusValid:      "Valid",
	StatusInvalid:    "Invalid",
	StatusOverloaded: "Overloaded",
}

// String 返回校验状态的可读名称。
func (s VerifyStatus) String() string {
	if name, ok := verifyStatusNames[s]; ok {
		return name
	}
	return fmt.Sprintf("Unknown(%d)", s)
}

// TxEnvelope 交易信封，封装交易数据及其元信息。
// 在校验团队内部传递时使用。
type TxEnvelope struct {
	TxID             types.Hash512 // 交易 ID
	RawTx            []byte        // 原始交易字节
	ReceivedAt       int64         // 接收时间戳（Unix 毫秒）
	LeaderAmount     int64         // 首领输入的币金数量
	LeaderCoinAge    uint64        // 首领输入的币权（币龄*币量）
	CoinInputAmounts []int64       // 所有币金输入的金额列表
	CoinInputAges    []uint64      // 所有币金输入的币权列表
	HasCreditDestroy bool          // 是否包含凭信提前销毁
	TotalFee         int64         // 交易手续费
	TotalStakes      uint64        // 总币权销毁量
}

// NewTxEnvelope 创建一个新的交易信封，自动设置接收时间。
func NewTxEnvelope(txID types.Hash512, rawTx []byte) *TxEnvelope {
	return &TxEnvelope{
		TxID:       txID,
		RawTx:      rawTx,
		ReceivedAt: time.Now().UnixMilli(),
	}
}

// VerifyTask 校验任务，由 Dispatcher 分配给 Verifier。
type VerifyTask struct {
	Envelope   *TxEnvelope // 待校验的交易信封
	AssignedAt int64       // 分配时间戳（Unix 毫秒）
}

// NewVerifyTask 创建一个新的校验任务。
func NewVerifyTask(env *TxEnvelope) *VerifyTask {
	return &VerifyTask{
		Envelope:   env,
		AssignedAt: time.Now().UnixMilli(),
	}
}

// VerifyResult 校验结果，由 Verifier 报告给 Dispatcher。
type VerifyResult struct {
	TxID       types.Hash512 // 交易 ID
	VerifierID NodeID        // 校验者节点 ID
	Status     VerifyStatus  // 校验状态
	Reason     string        // 失败原因（仅在 StatusInvalid 时有值）
}

// NewVerifyResult 创建一条校验结果。
func NewVerifyResult(txID types.Hash512, verifierID NodeID, status VerifyStatus, reason string) *VerifyResult {
	return &VerifyResult{
		TxID:       txID,
		VerifierID: verifierID,
		Status:     status,
		Reason:     reason,
	}
}

// ServiceAddresses 公共服务奖励地址集合。
// 包含数据驿站、区块查询和 STUN 中继三种服务的奖励地址。
type ServiceAddresses struct {
	Depots  types.PubKeyHash // 数据驿站服务地址
	Blockqs types.PubKeyHash // 区块查询服务地址
	Stun    types.PubKeyHash // STUN 中继服务地址
}

// MintRequest 铸造申请，由铸造者发送给 Broadcaster。
type MintRequest struct {
	Proof          []byte        // 择优池凭证
	CandidateIndex int           // 候选者排名
	MinterPub      []byte        // 铸造者公钥
}

// MintInfo 铸造信息，由 Broadcaster 返回给铸造者。
type MintInfo struct {
	TotalFees       int64            // 区块总交易费
	RewardAddresses ServiceAddresses // 公共服务奖励地址
	MintAmount      int64            // 铸造奖励金额
	FeeBurned       int64            // 销毁的手续费（如有）
}

// CoinbaseSubmission 铸造者提交的 Coinbase 交易。
type CoinbaseSubmission struct {
	CoinbaseTx []byte        // Coinbase 交易原始字节
	CoinbaseID types.Hash512 // Coinbase 交易 ID
}

// CheckRootData 校验根数据，由 Broadcaster 返回给铸造者。
// 包含 Coinbase 的 Merkle 路径、交易树根、UTXO/UTCO 指纹和 CheckRoot。
type CheckRootData struct {
	CoinbaseMerklePath []types.Hash512 // Coinbase 在交易哈希树中的验证路径
	TreeRoot           types.Hash512   // 交易哈希树根
	UTXOFingerprint    types.Hash512   // 当前 UTXO 集指纹
	UTCOFingerprint    types.Hash512   // 当前 UTCO 集指纹
	CheckRoot          types.Hash512   // 校验根 = Hash(TreeRoot || UTXOFp || UTCOFp)
}

// SignedCheckRoot 铸造者签名的 CheckRoot。
type SignedCheckRoot struct {
	Signature []byte // 对 CheckRoot 的签名
	MinterPub []byte // 铸造者公钥
}

// BlockHeader 简化的区块头结构（用于校验协议内部传递）。
// 完整的区块头定义在 internal/blockchain 包中。
type BlockHeader struct {
	Version       uint16        // 版本号
	Height        uint64        // 区块高度
	Timestamp     int64         // 出块时间戳
	PrevBlockID   types.Hash512 // 前一区块 ID
	CheckRoot     types.Hash512 // 校验根
	MinterPub     []byte        // 铸造者公钥
	MinterSig     []byte        // 铸造者签名
}

// BlockProof 区块证明，用于三阶段区块发布的第一阶段。
type BlockProof struct {
	CoinbaseTx      []byte          // Coinbase 交易原始字节
	MerklePath      []types.Hash512 // Coinbase Merkle 验证路径
	MinterSignature []byte          // 铸造者签名
	Header          BlockHeader     // 区块头
}

// BlockSummary 区块概要，用于三阶段区块发布的第二阶段。
// 包含截断的交易 ID 列表（每个取前 16 字节）。
type BlockSummary struct {
	Height         uint64     // 区块高度
	TruncatedTxIDs [][16]byte // 截断的交易 ID（前 16 字节）
}

// Guard 守卫者接口——外部交易的唯一入口。
// Guards 对入站交易执行首领校验，并将通过的交易提交给 Dispatcher。
type Guard interface {
	// LeaderVerify 对交易执行首领校验。
	// 仅校验首笔输入（leader input），判断交易是否可能合法。
	LeaderVerify(tx *TxEnvelope) (bool, error)

	// SubmitToDispatcher 将首领校验通过的交易提交给 Dispatcher。
	SubmitToDispatcher(tx *TxEnvelope) error

	// ForwardToGuards 将交易转发给其他团队的 Guards 以加速全网广播。
	ForwardToGuards(tx *TxEnvelope) error

	// IsBlacklisted 检查指定输出点是否在黑名单中。
	IsBlacklisted(op OutPoint) bool

	// AddBlacklist 将指定输出点添加到黑名单（冻结 24 小时）。
	AddBlacklist(op OutPoint) error
}

// Verifier 校验员接口——执行完整交易校验。
type Verifier interface {
	// RequestTask 向 Dispatcher 请求一个校验任务。
	RequestTask() (*VerifyTask, error)

	// Verify 对分配的任务执行完整校验。
	Verify(task *VerifyTask) (*VerifyResult, error)

	// DeliverToGuards 将已验证通过的交易发送给其他团队的 Guards。
	DeliverToGuards(tx *TxEnvelope) error
}

// Dispatcher 调度员接口——管理交易分配和结果收集。
type Dispatcher interface {
	// AcceptTx 接收已过首领校验的交易，存入待校验池。
	AcceptTx(tx *TxEnvelope) error

	// AssignTask 为指定 Verifier 分配一笔校验任务。
	AssignTask(verifierID NodeID) (*VerifyTask, error)

	// CollectResult 收集 Verifier 的校验结果。
	CollectResult(result *VerifyResult) error

	// SyncVerified 将已验证通过的交易同步给 Broadcaster。
	SyncVerified(broadcasterCh chan<- *TxEnvelope) error
}

// Broadcaster 广播者接口——管理铸造流程和区块发布。
type Broadcaster interface {
	// AcceptMintRequest 接受铸造申请，返回铸造信息。
	AcceptMintRequest(req *MintRequest) (*MintInfo, error)

	// AcceptCoinbase 接受 Coinbase 交易，返回校验根数据。
	AcceptCoinbase(cb *CoinbaseSubmission) (*CheckRootData, error)

	// AcceptSignedCheckRoot 接受铸造者签名的 CheckRoot。
	AcceptSignedCheckRoot(sig *SignedCheckRoot) error

	// PublishBlock 发布当前区块到网络。
	PublishBlock() error

	// ReceiveBlock 接收来自其他团队发布的区块证明。
	ReceiveBlock(proof *BlockProof) error

	// SyncMissingTxs 根据区块概要同步缺失的交易。
	SyncMissingTxs(summary *BlockSummary) error
}

// UTXOLookup UTXO 查询接口，供首领校验使用。
// 具体实现由缓存层提供。
type UTXOLookup interface {
	// Lookup 查询指定输出点的 UTXO 条目。
	// 若不存在返回 nil 和 ErrNotFound。
	Lookup(op OutPoint) (*UTXOEntryInfo, error)
}

// UTXOEntryInfo UTXO 条目简要信息（供校验使用）。
type UTXOEntryInfo struct {
	Amount     int64            // 金额
	Address    types.PubKeyHash // 接收者公钥哈希
	Height     uint64           // 创建时区块高度
	IsCoinbase bool             // 是否 Coinbase 产出
	LockScript []byte           // 锁定脚本
	Config     byte             // 输出配置字节
}

// 常用错误定义
var (
	ErrNotFound       = errors.New("entry not found")
	ErrAlreadyExists  = errors.New("entry already exists")
	ErrNilEnvelope    = errors.New("tx envelope is nil")
	ErrNilTask        = errors.New("verify task is nil")
)
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/verification/ -run "TestNewTxEnvelope|TestVerifyStatus|TestNewVerifyResult|TestNewVerifyTask|TestOutPoint|TestInterfaces|TestServiceAddresses"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/verification/roles.go internal/verification/roles_test.go
git commit -m "feat(verification): add role interfaces (Guard/Verifier/Dispatcher/Broadcaster) and auxiliary types"
```

---

## Task 3: 首领校验逻辑 (internal/verification/leader.go)

**Files:**
- Create: `internal/verification/leader.go`
- Test: `internal/verification/leader_test.go`

本 Task 实现首领校验核心逻辑——`LeaderVerifier` 结构体，包含首领输入验证、黑名单管理、过期清理，以及 50% 概率随机抉择策略。

### Step 1: 写失败测试

创建 `internal/verification/leader_test.go`：

```go
package verification

import (
	"math"
	"sync"
	"testing"
	"time"

	"github.com/cxio/evidcoin/pkg/types"
)

// mockUTXOLookup 用于测试的 UTXO 查询 mock
type mockUTXOLookup struct {
	mu      sync.RWMutex
	entries map[string]*UTXOEntryInfo
}

func newMockUTXOLookup() *mockUTXOLookup {
	return &mockUTXOLookup{
		entries: make(map[string]*UTXOEntryInfo),
	}
}

func (m *mockUTXOLookup) Lookup(op OutPoint) (*UTXOEntryInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if e, ok := m.entries[op.Key()]; ok {
		return e, nil
	}
	return nil, ErrNotFound
}

func (m *mockUTXOLookup) addEntry(op OutPoint, entry *UTXOEntryInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries[op.Key()] = entry
}

// 辅助函数：构建有效的首领校验交易信封
func makeLeaderTestEnvelope(leaderAmount int64, leaderAge uint64, coinAmounts []int64, coinAges []uint64) *TxEnvelope {
	var txID types.Hash512
	txID[0] = 0x01
	env := &TxEnvelope{
		TxID:             txID,
		RawTx:            []byte{0x01},
		ReceivedAt:       time.Now().UnixMilli(),
		LeaderAmount:     leaderAmount,
		LeaderCoinAge:    leaderAge,
		CoinInputAmounts: coinAmounts,
		CoinInputAges:    coinAges,
	}
	return env
}

// --- LeaderVerifier 构造测试 ---

func TestNewLeaderVerifier(t *testing.T) {
	lookup := newMockUTXOLookup()
	lv := NewLeaderVerifier(lookup)

	if lv == nil {
		t.Fatal("NewLeaderVerifier() returned nil")
	}
}

// --- 首领校验：有效输入通过 ---

func TestLeaderVerifier_Verify_ValidLeader(t *testing.T) {
	lookup := newMockUTXOLookup()
	leaderOP := OutPoint{TxID: types.Hash512{0x10}, Index: 0}
	lookup.addEntry(leaderOP, &UTXOEntryInfo{
		Amount:     5000,
		Address:    types.PubKeyHash{0x01},
		Height:     100,
		LockScript: []byte{0x76, 0xa9},
		Config:     byte(types.OutTypeCoin),
	})

	lv := NewLeaderVerifier(lookup)

	// 首领输入有最高币权
	env := makeLeaderTestEnvelope(5000, 500000, []int64{5000, 1000}, []uint64{500000, 100000})
	env.TxID = types.Hash512{0x10}

	result, err := lv.Verify(env, leaderOP)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !result.Valid {
		t.Errorf("Verify() Valid = false, want true; Reason = %q", result.Reason)
	}
}

// --- 首领校验：黑名单输入被拒 ---

func TestLeaderVerifier_Verify_Blacklisted(t *testing.T) {
	lookup := newMockUTXOLookup()
	leaderOP := OutPoint{TxID: types.Hash512{0x20}, Index: 0}
	lookup.addEntry(leaderOP, &UTXOEntryInfo{
		Amount:     5000,
		Address:    types.PubKeyHash{0x01},
		Height:     100,
		LockScript: []byte{0x76, 0xa9},
		Config:     byte(types.OutTypeCoin),
	})

	lv := NewLeaderVerifier(lookup)
	lv.AddToBlacklist(leaderOP)

	env := makeLeaderTestEnvelope(5000, 500000, []int64{5000}, []uint64{500000})

	result, err := lv.Verify(env, leaderOP)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if result.Valid {
		t.Error("Verify() should reject blacklisted leader input")
	}
	if result.Reason == "" {
		t.Error("Reason should explain why rejected")
	}
}

// --- 首领校验：UTXO 不存在被拒 ---

func TestLeaderVerifier_Verify_UTXONotFound(t *testing.T) {
	lookup := newMockUTXOLookup()
	// 不添加任何条目
	lv := NewLeaderVerifier(lookup)

	leaderOP := OutPoint{TxID: types.Hash512{0x30}, Index: 0}
	env := makeLeaderTestEnvelope(5000, 500000, []int64{5000}, []uint64{500000})

	result, err := lv.Verify(env, leaderOP)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if result.Valid {
		t.Error("Verify() should reject when UTXO not found")
	}
}

// --- 首领校验：非 Coin 类型被拒 ---

func TestLeaderVerifier_Verify_NonCoinType(t *testing.T) {
	lookup := newMockUTXOLookup()
	leaderOP := OutPoint{TxID: types.Hash512{0x40}, Index: 0}
	lookup.addEntry(leaderOP, &UTXOEntryInfo{
		Amount:     5000,
		Address:    types.PubKeyHash{0x01},
		Height:     100,
		LockScript: []byte{0x76, 0xa9},
		Config:     byte(types.OutTypeCredit), // 凭信类型，不是 Coin
	})

	lv := NewLeaderVerifier(lookup)
	env := makeLeaderTestEnvelope(5000, 500000, []int64{5000}, []uint64{500000})

	result, err := lv.Verify(env, leaderOP)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if result.Valid {
		t.Error("Verify() should reject non-Coin leader input")
	}
}

// --- 首领校验：非最高币权被拒 ---

func TestLeaderVerifier_Verify_NotHighestCoinAge(t *testing.T) {
	lookup := newMockUTXOLookup()
	leaderOP := OutPoint{TxID: types.Hash512{0x50}, Index: 0}
	lookup.addEntry(leaderOP, &UTXOEntryInfo{
		Amount:     1000,
		Address:    types.PubKeyHash{0x01},
		Height:     100,
		LockScript: []byte{0x76, 0xa9},
		Config:     byte(types.OutTypeCoin),
	})

	lv := NewLeaderVerifier(lookup)

	// 首领输入的币权 100000 小于另一个 coin 输入的 500000
	env := makeLeaderTestEnvelope(1000, 100000, []int64{1000, 5000}, []uint64{100000, 500000})

	result, err := lv.Verify(env, leaderOP)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if result.Valid {
		t.Error("Verify() should reject leader input without highest coin-age")
	}
}

// --- 黑名单管理 ---

func TestLeaderVerifier_Blacklist_AddAndCheck(t *testing.T) {
	lookup := newMockUTXOLookup()
	lv := NewLeaderVerifier(lookup)

	op := OutPoint{TxID: types.Hash512{0x60}, Index: 0}
	if lv.IsBlacklisted(op) {
		t.Error("should not be blacklisted before adding")
	}

	lv.AddToBlacklist(op)
	if !lv.IsBlacklisted(op) {
		t.Error("should be blacklisted after adding")
	}
}

func TestLeaderVerifier_Blacklist_DifferentOutPoints(t *testing.T) {
	lookup := newMockUTXOLookup()
	lv := NewLeaderVerifier(lookup)

	op1 := OutPoint{TxID: types.Hash512{0x70}, Index: 0}
	op2 := OutPoint{TxID: types.Hash512{0x70}, Index: 1}

	lv.AddToBlacklist(op1)

	if !lv.IsBlacklisted(op1) {
		t.Error("op1 should be blacklisted")
	}
	if lv.IsBlacklisted(op2) {
		t.Error("op2 should not be blacklisted")
	}
}

// --- 黑名单过期清理 ---

func TestLeaderVerifier_Blacklist_Expiry(t *testing.T) {
	lookup := newMockUTXOLookup()
	lv := NewLeaderVerifier(lookup)

	op := OutPoint{TxID: types.Hash512{0x80}, Index: 0}

	// 手动设置一个已过期的黑名单条目（1 小时前过期）
	lv.setBlacklistEntry(op, time.Now().Add(-1*time.Hour))

	// 清理前还在黑名单中（因为 IsBlacklisted 不自动清理）
	lv.CleanExpiredBlacklist()

	if lv.IsBlacklisted(op) {
		t.Error("expired entry should be cleaned")
	}
}

func TestLeaderVerifier_Blacklist_NotExpired(t *testing.T) {
	lookup := newMockUTXOLookup()
	lv := NewLeaderVerifier(lookup)

	op := OutPoint{TxID: types.Hash512{0x90}, Index: 0}
	lv.AddToBlacklist(op)

	// 清理不应移除未过期条目
	lv.CleanExpiredBlacklist()

	if !lv.IsBlacklisted(op) {
		t.Error("non-expired entry should not be cleaned")
	}
}

// --- 随机抉择统计分布测试 ---

func TestLeaderVerifier_ShouldVerifyFirst_Distribution(t *testing.T) {
	lookup := newMockUTXOLookup()
	lv := NewLeaderVerifier(lookup)

	const iterations = 10000
	verifyFirst := 0

	for i := 0; i < iterations; i++ {
		if lv.ShouldVerifyFirst() {
			verifyFirst++
		}
	}

	// 50% 概率，允许 5% 误差范围
	ratio := float64(verifyFirst) / float64(iterations)
	if math.Abs(ratio-0.5) > 0.05 {
		t.Errorf("ShouldVerifyFirst() ratio = %.3f, want ~0.5 (within 5%%)", ratio)
	}
}

// --- LeaderResult 构造测试 ---

func TestLeaderResult_Valid(t *testing.T) {
	r := &LeaderResult{Valid: true, Reason: ""}
	if !r.Valid {
		t.Error("Valid should be true")
	}
}

func TestLeaderResult_Invalid(t *testing.T) {
	r := &LeaderResult{Valid: false, Reason: "blacklisted"}
	if r.Valid {
		t.Error("Valid should be false")
	}
	if r.Reason != "blacklisted" {
		t.Errorf("Reason = %q, want %q", r.Reason, "blacklisted")
	}
}

// --- nil 输入保护测试 ---

func TestLeaderVerifier_Verify_NilEnvelope(t *testing.T) {
	lookup := newMockUTXOLookup()
	lv := NewLeaderVerifier(lookup)

	_, err := lv.Verify(nil, OutPoint{})
	if err == nil {
		t.Error("Verify(nil) should return error")
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/verification/ -run "TestNewLeaderVerifier|TestLeaderVerifier|TestLeaderResult"
```

预期输出：编译失败，`LeaderVerifier`、`NewLeaderVerifier`、`LeaderResult` 等未定义。

### Step 3: 写最小实现

创建 `internal/verification/leader.go`：

```go
package verification

import (
	"math/rand/v2"
	"sync"
	"time"

	"github.com/cxio/evidcoin/pkg/types"
)

// BlacklistDuration 黑名单冻结时长（24 小时）。
const BlacklistDuration = 24 * time.Hour

// LeaderResult 首领校验结果。
type LeaderResult struct {
	Valid  bool   // 是否通过
	Reason string // 拒绝原因（仅在 Valid=false 时有值）
}

// LeaderVerifier 首领校验器，执行交易首领输入的轻量预检。
// 线程安全，内部使用互斥锁保护黑名单。
type LeaderVerifier struct {
	mu         sync.RWMutex
	blacklist  map[string]time.Time // key: OutPoint.Key(), value: 到期时间
	utxoLookup UTXOLookup           // UTXO 查询接口
}

// NewLeaderVerifier 创建新的首领校验器。
func NewLeaderVerifier(lookup UTXOLookup) *LeaderVerifier {
	return &LeaderVerifier{
		blacklist:  make(map[string]time.Time),
		utxoLookup: lookup,
	}
}

// Verify 对交易信封执行首领校验。
// leaderOP 为首领输入引用的输出点。
//
// 校验步骤：
//  1. 检查首领输入是否在黑名单中
//  2. 验证首领输入的 UTXO 存在性
//  3. 验证首领输入是 Coin 类型
//  4. 验证首领输入在所有 coin 输入中有最高币权
func (lv *LeaderVerifier) Verify(env *TxEnvelope, leaderOP OutPoint) (*LeaderResult, error) {
	if env == nil {
		return nil, ErrNilEnvelope
	}

	// 步骤 1: 检查黑名单
	if lv.IsBlacklisted(leaderOP) {
		return &LeaderResult{
			Valid:  false,
			Reason: "leader input is blacklisted",
		}, nil
	}

	// 步骤 2: 验证 UTXO 存在性
	entry, err := lv.utxoLookup.Lookup(leaderOP)
	if err != nil {
		return &LeaderResult{
			Valid:  false,
			Reason: "leader input utxo not found",
		}, nil
	}

	// 步骤 3: 验证 Coin 类型
	// OutputConfig 低 4 位为类型
	entryType := types.OutputConfig(entry.Config) & 0x0f
	if entryType != types.OutTypeCoin {
		return &LeaderResult{
			Valid:  false,
			Reason: "leader input is not coin type",
		}, nil
	}

	// 步骤 4: 验证最高币权
	// 首领输入的币权必须在所有 coin 输入中最高
	if len(env.CoinInputAges) > 0 {
		for i, age := range env.CoinInputAges {
			// 跳过首领自身（首领是第一个 coin 输入）
			if i == 0 {
				continue
			}
			if age > env.LeaderCoinAge {
				return &LeaderResult{
					Valid:  false,
					Reason: "leader input does not have highest coin-age",
				}, nil
			}
		}
	}

	return &LeaderResult{Valid: true}, nil
}

// IsBlacklisted 检查指定输出点是否在黑名单中且未过期。
func (lv *LeaderVerifier) IsBlacklisted(op OutPoint) bool {
	lv.mu.RLock()
	defer lv.mu.RUnlock()

	expiry, ok := lv.blacklist[op.Key()]
	if !ok {
		return false
	}
	return time.Now().Before(expiry)
}

// AddToBlacklist 将输出点添加到黑名单，冻结 24 小时。
func (lv *LeaderVerifier) AddToBlacklist(op OutPoint) {
	lv.mu.Lock()
	defer lv.mu.Unlock()
	lv.blacklist[op.Key()] = time.Now().Add(BlacklistDuration)
}

// setBlacklistEntry 设置黑名单条目的到期时间（供测试使用）。
func (lv *LeaderVerifier) setBlacklistEntry(op OutPoint, expiry time.Time) {
	lv.mu.Lock()
	defer lv.mu.Unlock()
	lv.blacklist[op.Key()] = expiry
}

// CleanExpiredBlacklist 清理所有已过期的黑名单条目。
func (lv *LeaderVerifier) CleanExpiredBlacklist() {
	lv.mu.Lock()
	defer lv.mu.Unlock()

	now := time.Now()
	for key, expiry := range lv.blacklist {
		if now.After(expiry) {
			delete(lv.blacklist, key)
		}
	}
}

// ShouldVerifyFirst 返回是否应先执行首领校验再转发。
// 随机抉择策略：50% 概率先校验后转发，50% 直接转发。
func (lv *LeaderVerifier) ShouldVerifyFirst() bool {
	return rand.IntN(2) == 0
}
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/verification/ -run "TestNewLeaderVerifier|TestLeaderVerifier|TestLeaderResult"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/verification/leader.go internal/verification/leader_test.go
git commit -m "feat(verification): add leader verification logic with blacklist and random forwarding strategy"
```

---

## Task 4: 冗余校验与扩展复核 (internal/verification/redundancy.go)

**Files:**
- Create: `internal/verification/redundancy.go`
- Test: `internal/verification/redundancy_test.go`

本 Task 实现 `TaskDispatcher` 结构体——管理交易的冗余校验分配、结果收集、扩展复核（两级）以及校验员绩效统计。

### Step 1: 写失败测试

创建 `internal/verification/redundancy_test.go`：

```go
package verification

import (
	"testing"
	"time"

	"github.com/cxio/evidcoin/pkg/types"
)

// 辅助函数：创建测试信封
func makeTestEnvelopeWithID(id byte) *TxEnvelope {
	var txID types.Hash512
	txID[0] = id
	return &TxEnvelope{
		TxID:       txID,
		RawTx:      []byte{id},
		ReceivedAt: time.Now().UnixMilli(),
	}
}

// --- TaskDispatcher 构造测试 ---

func TestNewTaskDispatcher(t *testing.T) {
	td := NewTaskDispatcher(2)
	if td == nil {
		t.Fatal("NewTaskDispatcher() returned nil")
	}
	if td.RedundancyLevel() != 2 {
		t.Errorf("RedundancyLevel() = %d, want 2", td.RedundancyLevel())
	}
}

func TestNewTaskDispatcher_MinRedundancy(t *testing.T) {
	// 冗余级别最小为 2
	td := NewTaskDispatcher(1)
	if td.RedundancyLevel() != 2 {
		t.Errorf("RedundancyLevel() = %d, want 2 (minimum)", td.RedundancyLevel())
	}

	td = NewTaskDispatcher(0)
	if td.RedundancyLevel() != 2 {
		t.Errorf("RedundancyLevel() = %d, want 2 (minimum)", td.RedundancyLevel())
	}
}

// --- 提交交易 ---

func TestTaskDispatcher_SubmitTx(t *testing.T) {
	td := NewTaskDispatcher(2)
	env := makeTestEnvelopeWithID(0x01)

	err := td.SubmitTx(env)
	if err != nil {
		t.Fatalf("SubmitTx() error = %v", err)
	}
	if td.PendingCount() != 1 {
		t.Errorf("PendingCount() = %d, want 1", td.PendingCount())
	}
}

func TestTaskDispatcher_SubmitTx_Duplicate(t *testing.T) {
	td := NewTaskDispatcher(2)
	env := makeTestEnvelopeWithID(0x02)

	_ = td.SubmitTx(env)
	err := td.SubmitTx(env)
	if err == nil {
		t.Error("SubmitTx() duplicate should return error")
	}
}

func TestTaskDispatcher_SubmitTx_Nil(t *testing.T) {
	td := NewTaskDispatcher(2)
	err := td.SubmitTx(nil)
	if err == nil {
		t.Error("SubmitTx(nil) should return error")
	}
}

// --- 任务分配 ---

func TestTaskDispatcher_GetTask(t *testing.T) {
	td := NewTaskDispatcher(2)
	env := makeTestEnvelopeWithID(0x03)
	_ = td.SubmitTx(env)

	verifier := NodeID{0x01}
	task, err := td.GetTask(verifier)
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if task == nil {
		t.Fatal("GetTask() returned nil task")
	}
	if task.Envelope.TxID != env.TxID {
		t.Error("task Envelope TxID mismatch")
	}
}

func TestTaskDispatcher_GetTask_Empty(t *testing.T) {
	td := NewTaskDispatcher(2)
	verifier := NodeID{0x01}

	task, err := td.GetTask(verifier)
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	// 没有待分配任务时返回 nil
	if task != nil {
		t.Error("GetTask() should return nil when no tasks pending")
	}
}

func TestTaskDispatcher_GetTask_NotAssignedTwice(t *testing.T) {
	td := NewTaskDispatcher(2)
	env := makeTestEnvelopeWithID(0x04)
	_ = td.SubmitTx(env)

	verifier := NodeID{0x01}
	// 第一次应该得到任务
	task1, _ := td.GetTask(verifier)
	if task1 == nil {
		t.Fatal("first GetTask() should return a task")
	}

	// 同一 Verifier 不应再次得到相同交易
	task2, _ := td.GetTask(verifier)
	if task2 != nil {
		t.Error("same verifier should not get same tx twice")
	}
}

// --- 结果提交：2 个都通过 → 接受 ---

func TestTaskDispatcher_AllValid(t *testing.T) {
	td := NewTaskDispatcher(2)
	env := makeTestEnvelopeWithID(0x10)
	_ = td.SubmitTx(env)

	v1, v2 := NodeID{0x01}, NodeID{0x02}
	td.GetTask(v1)
	td.GetTask(v2)

	// 两个 Verifier 都报告有效
	r1 := NewVerifyResult(env.TxID, v1, StatusValid, "")
	r2 := NewVerifyResult(env.TxID, v2, StatusValid, "")

	err := td.SubmitResult(r1)
	if err != nil {
		t.Fatalf("SubmitResult(r1) error = %v", err)
	}
	err = td.SubmitResult(r2)
	if err != nil {
		t.Fatalf("SubmitResult(r2) error = %v", err)
	}

	// 应在已验证列表中
	verified := td.GetVerifiedTxs()
	found := false
	for _, v := range verified {
		if v.TxID == env.TxID {
			found = true
			break
		}
	}
	if !found {
		t.Error("all-valid tx should be in verified list")
	}
}

// --- 结果提交：1 通过 1 失败 → 触发扩展复核 ---

func TestTaskDispatcher_OneInvalid_TriggersReview(t *testing.T) {
	td := NewTaskDispatcher(2)
	env := makeTestEnvelopeWithID(0x11)
	_ = td.SubmitTx(env)

	v1, v2 := NodeID{0x01}, NodeID{0x02}
	td.GetTask(v1)
	td.GetTask(v2)

	r1 := NewVerifyResult(env.TxID, v1, StatusValid, "")
	r2 := NewVerifyResult(env.TxID, v2, StatusInvalid, "script failed")

	td.SubmitResult(r1)
	td.SubmitResult(r2)

	// 不应在已验证列表中（进入了复核）
	verified := td.GetVerifiedTxs()
	for _, v := range verified {
		if v.TxID == env.TxID {
			t.Error("disputed tx should not be in verified list before review")
		}
	}

	// 应在待复核列表中
	pt := td.GetPendingTx(env.TxID)
	if pt == nil {
		t.Fatal("tx should still be in pending")
	}
	if pt.ReviewLevel != 1 {
		t.Errorf("ReviewLevel = %d, want 1", pt.ReviewLevel)
	}
}

// --- Level 1 复核：全部通过 → 接受 ---

func TestTaskDispatcher_Level1Review_AllValid(t *testing.T) {
	td := NewTaskDispatcher(2)
	env := makeTestEnvelopeWithID(0x20)
	_ = td.SubmitTx(env)

	// 初始冗余校验：一个通过一个失败
	v1, v2 := NodeID{0x01}, NodeID{0x02}
	td.GetTask(v1)
	td.GetTask(v2)
	td.SubmitResult(NewVerifyResult(env.TxID, v1, StatusValid, ""))
	td.SubmitResult(NewVerifyResult(env.TxID, v2, StatusInvalid, "fail"))

	// 注册绩效更好的 Verifiers 用于复核
	v3, v4 := NodeID{0x03}, NodeID{0x04}
	td.RegisterVerifier(v3)
	td.RegisterVerifier(v4)
	td.UpdateStats(v3, true) // 给 v3 和 v4 设置好的绩效
	td.UpdateStats(v4, true)

	// Level 1 复核：两个都通过
	task3, _ := td.GetTask(v3)
	task4, _ := td.GetTask(v4)
	if task3 == nil || task4 == nil {
		t.Fatal("review verifiers should get tasks")
	}
	td.SubmitResult(NewVerifyResult(env.TxID, v3, StatusValid, ""))
	td.SubmitResult(NewVerifyResult(env.TxID, v4, StatusValid, ""))

	// 零错误报告 → 接受
	verified := td.GetVerifiedTxs()
	found := false
	for _, v := range verified {
		if v.TxID == env.TxID {
			found = true
			break
		}
	}
	if !found {
		t.Error("level 1 review all-valid should accept tx")
	}
}

// --- Level 1 复核：超半数失败 → 拒绝 ---

func TestTaskDispatcher_Level1Review_MajorityInvalid(t *testing.T) {
	td := NewTaskDispatcher(2)
	env := makeTestEnvelopeWithID(0x21)
	_ = td.SubmitTx(env)

	// 初始冗余校验触发复核
	v1, v2 := NodeID{0x01}, NodeID{0x02}
	td.GetTask(v1)
	td.GetTask(v2)
	td.SubmitResult(NewVerifyResult(env.TxID, v1, StatusValid, ""))
	td.SubmitResult(NewVerifyResult(env.TxID, v2, StatusInvalid, "fail"))

	// Level 1 复核
	v3, v4 := NodeID{0x03}, NodeID{0x04}
	td.RegisterVerifier(v3)
	td.RegisterVerifier(v4)
	td.UpdateStats(v3, true)
	td.UpdateStats(v4, true)

	td.GetTask(v3)
	td.GetTask(v4)
	td.SubmitResult(NewVerifyResult(env.TxID, v3, StatusInvalid, "fail"))
	td.SubmitResult(NewVerifyResult(env.TxID, v4, StatusInvalid, "fail"))

	// 超半数失败 → 拒绝
	rejected := td.GetRejectedTxs()
	found := false
	for _, r := range rejected {
		if r.TxID == env.TxID {
			found = true
			break
		}
	}
	if !found {
		t.Error("level 1 review majority-invalid should reject tx")
	}
}

// --- Level 2 复核：全部通过 → 接受 ---

func TestTaskDispatcher_Level2Review_AllValid(t *testing.T) {
	td := NewTaskDispatcher(2)
	env := makeTestEnvelopeWithID(0x30)
	_ = td.SubmitTx(env)

	// 初始冗余校验触发复核
	v1, v2 := NodeID{0x01}, NodeID{0x02}
	td.GetTask(v1)
	td.GetTask(v2)
	td.SubmitResult(NewVerifyResult(env.TxID, v1, StatusValid, ""))
	td.SubmitResult(NewVerifyResult(env.TxID, v2, StatusInvalid, "fail"))

	// Level 1 复核：不足半数失败 → 升级 Level 2
	v3, v4 := NodeID{0x03}, NodeID{0x04}
	td.RegisterVerifier(v3)
	td.RegisterVerifier(v4)
	td.UpdateStats(v3, true)
	td.UpdateStats(v4, true)

	td.GetTask(v3)
	td.GetTask(v4)
	// 1 个有效 1 个无效（不足半数无效）
	td.SubmitResult(NewVerifyResult(env.TxID, v3, StatusValid, ""))
	td.SubmitResult(NewVerifyResult(env.TxID, v4, StatusInvalid, "fail"))

	// 验证进入 Level 2
	pt := td.GetPendingTx(env.TxID)
	if pt == nil {
		t.Fatal("tx should still be pending for level 2")
	}
	if pt.ReviewLevel != 2 {
		t.Errorf("ReviewLevel = %d, want 2", pt.ReviewLevel)
	}

	// Level 2 复核
	v5, v6 := NodeID{0x05}, NodeID{0x06}
	td.RegisterVerifier(v5)
	td.RegisterVerifier(v6)
	td.UpdateStats(v5, true)
	td.UpdateStats(v6, true)

	td.GetTask(v5)
	td.GetTask(v6)
	td.SubmitResult(NewVerifyResult(env.TxID, v5, StatusValid, ""))
	td.SubmitResult(NewVerifyResult(env.TxID, v6, StatusValid, ""))

	// 全部有效 → 接受
	verified := td.GetVerifiedTxs()
	found := false
	for _, v := range verified {
		if v.TxID == env.TxID {
			found = true
			break
		}
	}
	if !found {
		t.Error("level 2 review all-valid should accept tx")
	}
}

// --- Level 2 复核：任何失败 → 拒绝 ---

func TestTaskDispatcher_Level2Review_AnyInvalid(t *testing.T) {
	td := NewTaskDispatcher(2)
	env := makeTestEnvelopeWithID(0x31)
	_ = td.SubmitTx(env)

	// 初始冗余校验触发复核
	v1, v2 := NodeID{0x01}, NodeID{0x02}
	td.GetTask(v1)
	td.GetTask(v2)
	td.SubmitResult(NewVerifyResult(env.TxID, v1, StatusValid, ""))
	td.SubmitResult(NewVerifyResult(env.TxID, v2, StatusInvalid, "fail"))

	// Level 1: 不足半数失败 → 升级 Level 2
	v3, v4 := NodeID{0x03}, NodeID{0x04}
	td.RegisterVerifier(v3)
	td.RegisterVerifier(v4)
	td.UpdateStats(v3, true)
	td.UpdateStats(v4, true)

	td.GetTask(v3)
	td.GetTask(v4)
	td.SubmitResult(NewVerifyResult(env.TxID, v3, StatusValid, ""))
	td.SubmitResult(NewVerifyResult(env.TxID, v4, StatusInvalid, "fail"))

	// Level 2: 任何失败 → 拒绝
	v5, v6 := NodeID{0x05}, NodeID{0x06}
	td.RegisterVerifier(v5)
	td.RegisterVerifier(v6)
	td.UpdateStats(v5, true)
	td.UpdateStats(v6, true)

	td.GetTask(v5)
	td.GetTask(v6)
	td.SubmitResult(NewVerifyResult(env.TxID, v5, StatusValid, ""))
	td.SubmitResult(NewVerifyResult(env.TxID, v6, StatusInvalid, "fail"))

	rejected := td.GetRejectedTxs()
	found := false
	for _, r := range rejected {
		if r.TxID == env.TxID {
			found = true
			break
		}
	}
	if !found {
		t.Error("level 2 review any-invalid should reject tx")
	}
}

// --- Verifier 绩效记录 ---

func TestTaskDispatcher_VerifierStats(t *testing.T) {
	td := NewTaskDispatcher(2)
	v := NodeID{0x10}
	td.RegisterVerifier(v)

	td.UpdateStats(v, true)
	td.UpdateStats(v, true)
	td.UpdateStats(v, false)

	stats := td.GetVerifierStats(v)
	if stats == nil {
		t.Fatal("GetVerifierStats() returned nil")
	}
	if stats.TotalTasks != 3 {
		t.Errorf("TotalTasks = %d, want 3", stats.TotalTasks)
	}
	if stats.Correct != 2 {
		t.Errorf("Correct = %d, want 2", stats.Correct)
	}
	if stats.Incorrect != 1 {
		t.Errorf("Incorrect = %d, want 1", stats.Incorrect)
	}
}

func TestTaskDispatcher_GetVerifierStats_Unknown(t *testing.T) {
	td := NewTaskDispatcher(2)
	stats := td.GetVerifierStats(NodeID{0xff})
	if stats != nil {
		t.Error("unknown verifier should return nil stats")
	}
}

// --- Overloaded 状态处理 ---

func TestTaskDispatcher_Overloaded_Ignored(t *testing.T) {
	td := NewTaskDispatcher(2)
	env := makeTestEnvelopeWithID(0x40)
	_ = td.SubmitTx(env)

	v1, v2, v3 := NodeID{0x01}, NodeID{0x02}, NodeID{0x03}
	td.GetTask(v1)
	td.GetTask(v2)

	// v1 报告过载，不计入有效结果
	td.SubmitResult(NewVerifyResult(env.TxID, v1, StatusOverloaded, ""))
	// v2 报告有效
	td.SubmitResult(NewVerifyResult(env.TxID, v2, StatusValid, ""))

	// 应继续等待（冗余未满足），需要新的 Verifier
	verified := td.GetVerifiedTxs()
	for _, v := range verified {
		if v.TxID == env.TxID {
			t.Error("tx should not be verified yet (only 1 effective result)")
		}
	}

	// 再分配给 v3
	td.GetTask(v3)
	td.SubmitResult(NewVerifyResult(env.TxID, v3, StatusValid, ""))

	verified = td.GetVerifiedTxs()
	found := false
	for _, v := range verified {
		if v.TxID == env.TxID {
			found = true
			break
		}
	}
	if !found {
		t.Error("tx should be verified after replacement verifier confirms")
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/verification/ -run "TestNewTaskDispatcher|TestTaskDispatcher"
```

预期输出：编译失败，`TaskDispatcher`、`NewTaskDispatcher`、`PendingTx` 等未定义。

### Step 3: 写最小实现

创建 `internal/verification/redundancy.go`：

```go
package verification

import (
	"sort"
	"sync"
	"time"

	"github.com/cxio/evidcoin/pkg/types"
)

// PendingTx 待校验交易，跟踪分配和结果状态。
type PendingTx struct {
	Envelope    *TxEnvelope     // 交易信封
	AssignedTo  []NodeID        // 已分配的校验者列表
	Results     []*VerifyResult // 收到的校验结果
	ReviewLevel int             // 当前复核级别（0=初始, 1=一级复核, 2=二级复核）
}

// VerifierStats 校验者绩效统计。
type VerifierStats struct {
	TotalTasks int // 总任务数
	Correct    int // 正确次数
	Incorrect  int // 错误次数
}

// Accuracy 计算校验准确率（0.0~1.0）。
func (s *VerifierStats) Accuracy() float64 {
	if s.TotalTasks == 0 {
		return 0
	}
	return float64(s.Correct) / float64(s.TotalTasks)
}

// TaskDispatcher 任务调度器，管理交易的冗余校验分配和结果收集。
// 线程安全，内部使用互斥锁保护所有状态。
type TaskDispatcher struct {
	mu              sync.Mutex
	pendingTxs      map[types.Hash512]*PendingTx // 待校验交易池
	verifiedTxs     []*TxEnvelope                // 已验证通过的交易
	rejectedTxs     []*TxEnvelope                // 已拒绝的交易
	verifierStats   map[NodeID]*VerifierStats     // 校验员绩效
	redundancyLevel int                           // 冗余校验级别
}

// NewTaskDispatcher 创建新的任务调度器。
// redundancy 为每笔交易分配的最小校验者数量，最小为 2。
func NewTaskDispatcher(redundancy int) *TaskDispatcher {
	if redundancy < 2 {
		redundancy = 2
	}
	return &TaskDispatcher{
		pendingTxs:      make(map[types.Hash512]*PendingTx),
		verifiedTxs:     make([]*TxEnvelope, 0),
		rejectedTxs:     make([]*TxEnvelope, 0),
		verifierStats:   make(map[NodeID]*VerifierStats),
		redundancyLevel: redundancy,
	}
}

// RedundancyLevel 返回当前冗余级别设置。
func (td *TaskDispatcher) RedundancyLevel() int {
	return td.redundancyLevel
}

// PendingCount 返回待校验交易数量。
func (td *TaskDispatcher) PendingCount() int {
	td.mu.Lock()
	defer td.mu.Unlock()
	return len(td.pendingTxs)
}

// SubmitTx 提交交易到待校验池。
func (td *TaskDispatcher) SubmitTx(env *TxEnvelope) error {
	if env == nil {
		return ErrNilEnvelope
	}
	td.mu.Lock()
	defer td.mu.Unlock()

	if _, exists := td.pendingTxs[env.TxID]; exists {
		return ErrAlreadyExists
	}

	td.pendingTxs[env.TxID] = &PendingTx{
		Envelope:   env,
		AssignedTo: make([]NodeID, 0),
		Results:    make([]*VerifyResult, 0),
	}
	return nil
}

// GetTask 为 Verifier 分配一笔待校验交易。
// 选择尚未被该 Verifier 校验、且分配数未满的交易。
// Verifier 不知道任务是常规校验还是复核任务。
// 无可用任务时返回 (nil, nil)。
func (td *TaskDispatcher) GetTask(verifierID NodeID) (*VerifyTask, error) {
	td.mu.Lock()
	defer td.mu.Unlock()

	for _, pt := range td.pendingTxs {
		// 检查是否已分配给该 Verifier
		alreadyAssigned := false
		for _, assigned := range pt.AssignedTo {
			if assigned == verifierID {
				alreadyAssigned = true
				break
			}
		}
		if alreadyAssigned {
			continue
		}

		// 确定当前阶段需要的分配数量
		needed := td.redundancyLevel
		// 计算当前有效分配数（排除 Overloaded 的）
		effectiveAssigned := td.countEffectiveAssignments(pt)
		if effectiveAssigned >= needed {
			// 已经分配足够，但可能还在等结果
			// 检查是否需要替补（有 Overloaded 的）
			effectiveResults := td.countEffectiveResults(pt)
			if effectiveResults >= needed {
				continue
			}
		}

		// 分配
		pt.AssignedTo = append(pt.AssignedTo, verifierID)
		return &VerifyTask{
			Envelope:   pt.Envelope,
			AssignedAt: time.Now().UnixMilli(),
		}, nil
	}

	return nil, nil
}

// countEffectiveAssignments 计算有效分配数（排除已返回 Overloaded 的）。
func (td *TaskDispatcher) countEffectiveAssignments(pt *PendingTx) int {
	overloaded := make(map[NodeID]bool)
	for _, r := range pt.Results {
		if r.Status == StatusOverloaded {
			overloaded[r.VerifierID] = true
		}
	}
	count := 0
	for _, a := range pt.AssignedTo {
		if !overloaded[a] {
			count++
		}
	}
	return count
}

// countEffectiveResults 计算有效结果数（排除 Overloaded 的）。
func (td *TaskDispatcher) countEffectiveResults(pt *PendingTx) int {
	count := 0
	for _, r := range pt.Results {
		if r.Status != StatusOverloaded {
			count++
		}
	}
	return count
}

// SubmitResult 收集 Verifier 的校验结果，并根据冗余策略决定交易状态。
func (td *TaskDispatcher) SubmitResult(result *VerifyResult) error {
	if result == nil {
		return ErrNilTask
	}
	td.mu.Lock()
	defer td.mu.Unlock()

	pt, exists := td.pendingTxs[result.TxID]
	if !exists {
		return ErrNotFound
	}

	// 记录结果
	pt.Results = append(pt.Results, result)

	// 统计有效结果
	effectiveResults := td.countEffectiveResults(pt)
	if effectiveResults < td.redundancyLevel {
		// 结果还不够，继续等待
		return nil
	}

	// 根据当前复核级别评估结果
	td.evaluateResults(pt)
	return nil
}

// evaluateResults 评估当前阶段的校验结果。
func (td *TaskDispatcher) evaluateResults(pt *PendingTx) {
	// 按阶段收集有效结果
	results := td.getCurrentStageResults(pt)
	if len(results) < td.redundancyLevel {
		return
	}

	invalidCount := 0
	for _, r := range results {
		if r.Status == StatusInvalid {
			invalidCount++
		}
	}

	switch pt.ReviewLevel {
	case 0:
		// 初始冗余校验
		if invalidCount == 0 {
			// 全部通过 → 接受
			td.acceptTx(pt)
		} else {
			// 至少一个失败 → 进入 Level 1 复核
			pt.ReviewLevel = 1
			// 重置当前阶段结果计数（保留历史，后续分配新 Verifiers）
		}

	case 1:
		// 一级复核
		if invalidCount == 0 {
			// 零错误报告 → 接受
			td.acceptTx(pt)
		} else if invalidCount > len(results)/2 {
			// 超半数失败 → 拒绝
			td.rejectTx(pt)
		} else {
			// 不足半数失败 → 升级 Level 2
			pt.ReviewLevel = 2
		}

	case 2:
		// 二级复核
		if invalidCount > 0 {
			// 任何错误 → 拒绝
			td.rejectTx(pt)
		} else {
			// 全部有效 → 接受
			td.acceptTx(pt)
		}
	}
}

// getCurrentStageResults 获取当前阶段的有效结果。
// 根据 ReviewLevel 和 AssignedTo 列表中各阶段分配的 Verifiers 来筛选。
func (td *TaskDispatcher) getCurrentStageResults(pt *PendingTx) []*VerifyResult {
	// 按阶段划分：每个阶段分配 redundancyLevel 个 Verifier
	startIdx := pt.ReviewLevel * td.redundancyLevel
	endIdx := startIdx + td.redundancyLevel
	if endIdx > len(pt.AssignedTo) {
		endIdx = len(pt.AssignedTo)
	}

	// 当前阶段分配的 Verifier ID 集合
	stageVerifiers := make(map[NodeID]bool)
	for i := startIdx; i < endIdx; i++ {
		stageVerifiers[pt.AssignedTo[i]] = true
	}

	// 筛选有效结果
	var results []*VerifyResult
	for _, r := range pt.Results {
		if stageVerifiers[r.VerifierID] && r.Status != StatusOverloaded {
			results = append(results, r)
		}
	}
	return results
}

// acceptTx 将交易移入已验证列表。
func (td *TaskDispatcher) acceptTx(pt *PendingTx) {
	td.verifiedTxs = append(td.verifiedTxs, pt.Envelope)
	delete(td.pendingTxs, pt.Envelope.TxID)
}

// rejectTx 将交易移入已拒绝列表。
func (td *TaskDispatcher) rejectTx(pt *PendingTx) {
	td.rejectedTxs = append(td.rejectedTxs, pt.Envelope)
	delete(td.pendingTxs, pt.Envelope.TxID)
}

// GetVerifiedTxs 获取已验证通过的交易列表。
func (td *TaskDispatcher) GetVerifiedTxs() []*TxEnvelope {
	td.mu.Lock()
	defer td.mu.Unlock()
	result := make([]*TxEnvelope, len(td.verifiedTxs))
	copy(result, td.verifiedTxs)
	return result
}

// GetRejectedTxs 获取已拒绝的交易列表。
func (td *TaskDispatcher) GetRejectedTxs() []*TxEnvelope {
	td.mu.Lock()
	defer td.mu.Unlock()
	result := make([]*TxEnvelope, len(td.rejectedTxs))
	copy(result, td.rejectedTxs)
	return result
}

// GetPendingTx 获取指定交易的待校验信息。
func (td *TaskDispatcher) GetPendingTx(txID types.Hash512) *PendingTx {
	td.mu.Lock()
	defer td.mu.Unlock()
	return td.pendingTxs[txID]
}

// RegisterVerifier 注册一个新的校验者。
func (td *TaskDispatcher) RegisterVerifier(id NodeID) {
	td.mu.Lock()
	defer td.mu.Unlock()
	if _, exists := td.verifierStats[id]; !exists {
		td.verifierStats[id] = &VerifierStats{}
	}
}

// UpdateStats 更新校验者绩效统计。
func (td *TaskDispatcher) UpdateStats(verifierID NodeID, correct bool) {
	td.mu.Lock()
	defer td.mu.Unlock()

	stats, exists := td.verifierStats[verifierID]
	if !exists {
		stats = &VerifierStats{}
		td.verifierStats[verifierID] = stats
	}

	stats.TotalTasks++
	if correct {
		stats.Correct++
	} else {
		stats.Incorrect++
	}
}

// GetVerifierStats 获取指定校验者的绩效统计。
func (td *TaskDispatcher) GetVerifierStats(verifierID NodeID) *VerifierStats {
	td.mu.Lock()
	defer td.mu.Unlock()
	return td.verifierStats[verifierID]
}

// SelectBestVerifiers 选择绩效最好的 n 个校验者（用于扩展复核）。
// 排除已参与当前交易的校验者。
func (td *TaskDispatcher) SelectBestVerifiers(exclude []NodeID, n int) []NodeID {
	td.mu.Lock()
	defer td.mu.Unlock()

	excludeMap := make(map[NodeID]bool)
	for _, id := range exclude {
		excludeMap[id] = true
	}

	// 收集候选者
	type candidate struct {
		id       NodeID
		accuracy float64
	}
	var candidates []candidate
	for id, stats := range td.verifierStats {
		if excludeMap[id] {
			continue
		}
		candidates = append(candidates, candidate{id: id, accuracy: stats.Accuracy()})
	}

	// 按准确率降序排序
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].accuracy > candidates[j].accuracy
	})

	// 取前 n 个
	result := make([]NodeID, 0, n)
	for i := 0; i < len(candidates) && i < n; i++ {
		result = append(result, candidates[i].id)
	}
	return result
}
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/verification/ -run "TestNewTaskDispatcher|TestTaskDispatcher"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/verification/redundancy.go internal/verification/redundancy_test.go
git commit -m "feat(verification): add TaskDispatcher with redundant verification and two-level extended review"
```

---

## Task 5: 铸造集成协议 (internal/verification/minting.go)

**Files:**
- Create: `internal/verification/minting.go`
- Test: `internal/verification/minting_test.go`

本 Task 实现铸造集成协议——包括铸造流程状态管理、Coinbase 验证、交易排序优先级、不收录交易排除逻辑。

### Step 1: 写失败测试

创建 `internal/verification/minting_test.go`：

```go
package verification

import (
	"sort"
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// --- MintingProtocol 构造测试 ---

func TestNewMintingProtocol(t *testing.T) {
	mp := NewMintingProtocol()
	if mp == nil {
		t.Fatal("NewMintingProtocol() returned nil")
	}
}

// --- ValidateMintRequest 测试 ---

func TestValidateMintRequest_Valid(t *testing.T) {
	mp := NewMintingProtocol()

	req := &MintRequest{
		Proof:          []byte{0x01, 0x02, 0x03},
		CandidateIndex: 0,
		MinterPub:      []byte{0x04, 0x05},
	}

	err := mp.ValidateMintRequest(req)
	if err != nil {
		t.Errorf("ValidateMintRequest() error = %v", err)
	}
}

func TestValidateMintRequest_NilProof(t *testing.T) {
	mp := NewMintingProtocol()

	req := &MintRequest{
		Proof:          nil,
		CandidateIndex: 0,
		MinterPub:      []byte{0x01},
	}

	err := mp.ValidateMintRequest(req)
	if err == nil {
		t.Error("ValidateMintRequest() should reject nil proof")
	}
}

func TestValidateMintRequest_NilMinterPub(t *testing.T) {
	mp := NewMintingProtocol()

	req := &MintRequest{
		Proof:          []byte{0x01},
		CandidateIndex: 0,
		MinterPub:      nil,
	}

	err := mp.ValidateMintRequest(req)
	if err == nil {
		t.Error("ValidateMintRequest() should reject nil minter pubkey")
	}
}

func TestValidateMintRequest_NegativeIndex(t *testing.T) {
	mp := NewMintingProtocol()

	req := &MintRequest{
		Proof:          []byte{0x01},
		CandidateIndex: -1,
		MinterPub:      []byte{0x01},
	}

	err := mp.ValidateMintRequest(req)
	if err == nil {
		t.Error("ValidateMintRequest() should reject negative candidate index")
	}
}

// --- PrepareMintInfo 测试 ---

func TestPrepareMintInfo_BasicFields(t *testing.T) {
	mp := NewMintingProtocol()

	env1 := &TxEnvelope{
		TxID:     types.Hash512{0x01},
		TotalFee: 100,
	}
	env2 := &TxEnvelope{
		TxID:     types.Hash512{0x02},
		TotalFee: 200,
	}

	addrs := ServiceAddresses{
		Depots:  types.PubKeyHash{0x10},
		Blockqs: types.PubKeyHash{0x20},
		Stun:    types.PubKeyHash{0x30},
	}

	info := mp.PrepareMintInfo([]*TxEnvelope{env1, env2}, 100, addrs)

	if info.TotalFees != 300 {
		t.Errorf("TotalFees = %d, want 300", info.TotalFees)
	}
	if info.RewardAddresses != addrs {
		t.Error("RewardAddresses mismatch")
	}
	if info.MintAmount <= 0 {
		t.Errorf("MintAmount = %d, should be positive", info.MintAmount)
	}
}

func TestPrepareMintInfo_EmptyTxs(t *testing.T) {
	mp := NewMintingProtocol()
	addrs := ServiceAddresses{}

	info := mp.PrepareMintInfo(nil, 0, addrs)

	if info.TotalFees != 0 {
		t.Errorf("TotalFees = %d, want 0", info.TotalFees)
	}
	// 即使没有交易，铸造奖励仍应存在
	if info.MintAmount <= 0 {
		t.Errorf("MintAmount = %d, should be positive even with no txs", info.MintAmount)
	}
}

// --- ValidateCoinbase 测试 ---

func TestValidateCoinbase_Valid(t *testing.T) {
	mp := NewMintingProtocol()

	info := &MintInfo{
		TotalFees:  300,
		MintAmount: 5000,
	}

	cb := &CoinbaseSubmission{
		CoinbaseTx: []byte{0x01, 0x02, 0x03, 0x04},
		CoinbaseID: types.Hash512{0xaa},
	}

	err := mp.ValidateCoinbase(cb, info)
	if err != nil {
		t.Errorf("ValidateCoinbase() error = %v", err)
	}
}

func TestValidateCoinbase_EmptyTx(t *testing.T) {
	mp := NewMintingProtocol()
	info := &MintInfo{TotalFees: 100, MintAmount: 5000}

	cb := &CoinbaseSubmission{
		CoinbaseTx: nil,
		CoinbaseID: types.Hash512{0xbb},
	}

	err := mp.ValidateCoinbase(cb, info)
	if err == nil {
		t.Error("ValidateCoinbase() should reject empty coinbase tx")
	}
}

func TestValidateCoinbase_ZeroCoinbaseID(t *testing.T) {
	mp := NewMintingProtocol()
	info := &MintInfo{TotalFees: 100, MintAmount: 5000}

	cb := &CoinbaseSubmission{
		CoinbaseTx: []byte{0x01},
		CoinbaseID: types.Hash512{},
	}

	err := mp.ValidateCoinbase(cb, info)
	if err == nil {
		t.Error("ValidateCoinbase() should reject zero coinbase ID")
	}
}

// --- PrepareCheckRootData 测试 ---

func TestPrepareCheckRootData(t *testing.T) {
	mp := NewMintingProtocol()

	coinbaseID := types.Hash512{0x01}
	txIDs := []types.Hash512{{0x02}, {0x03}}
	utxoFP := types.Hash512{0x10}
	utcoFP := types.Hash512{0x20}

	data := mp.PrepareCheckRootData(coinbaseID, txIDs, utxoFP, utcoFP)

	if data.UTXOFingerprint != utxoFP {
		t.Error("UTXOFingerprint mismatch")
	}
	if data.UTCOFingerprint != utcoFP {
		t.Error("UTCOFingerprint mismatch")
	}
	if data.TreeRoot.IsZero() {
		t.Error("TreeRoot should not be zero")
	}
	if data.CheckRoot.IsZero() {
		t.Error("CheckRoot should not be zero")
	}
	if len(data.CoinbaseMerklePath) == 0 {
		t.Error("CoinbaseMerklePath should not be empty")
	}
}

func TestPrepareCheckRootData_Deterministic(t *testing.T) {
	mp := NewMintingProtocol()

	coinbaseID := types.Hash512{0x01}
	txIDs := []types.Hash512{{0x02}, {0x03}}
	utxoFP := types.Hash512{0x10}
	utcoFP := types.Hash512{0x20}

	d1 := mp.PrepareCheckRootData(coinbaseID, txIDs, utxoFP, utcoFP)
	d2 := mp.PrepareCheckRootData(coinbaseID, txIDs, utxoFP, utcoFP)

	if d1.CheckRoot != d2.CheckRoot {
		t.Error("PrepareCheckRootData() should be deterministic")
	}
	if d1.TreeRoot != d2.TreeRoot {
		t.Error("TreeRoot should be deterministic")
	}
}

func TestPrepareCheckRootData_DifferentInputs(t *testing.T) {
	mp := NewMintingProtocol()

	d1 := mp.PrepareCheckRootData(types.Hash512{0x01}, []types.Hash512{{0x02}}, types.Hash512{0x10}, types.Hash512{0x20})
	d2 := mp.PrepareCheckRootData(types.Hash512{0x01}, []types.Hash512{{0x02}}, types.Hash512{0x11}, types.Hash512{0x20})

	if d1.CheckRoot == d2.CheckRoot {
		t.Error("different UTXO fingerprints should produce different CheckRoot")
	}
}

// --- ValidateCheckRootSignature 测试 ---

func TestValidateCheckRootSignature_NonEmpty(t *testing.T) {
	mp := NewMintingProtocol()

	sig := &SignedCheckRoot{
		Signature: []byte{0x01, 0x02, 0x03},
		MinterPub: []byte{0x04, 0x05},
	}

	// 简化验证：只要签名和公钥非空即可
	err := mp.ValidateCheckRootSignature(sig, types.Hash512{0xaa})
	if err != nil {
		t.Errorf("ValidateCheckRootSignature() error = %v", err)
	}
}

func TestValidateCheckRootSignature_EmptySig(t *testing.T) {
	mp := NewMintingProtocol()

	sig := &SignedCheckRoot{
		Signature: nil,
		MinterPub: []byte{0x01},
	}

	err := mp.ValidateCheckRootSignature(sig, types.Hash512{0xaa})
	if err == nil {
		t.Error("should reject empty signature")
	}
}

func TestValidateCheckRootSignature_EmptyPub(t *testing.T) {
	mp := NewMintingProtocol()

	sig := &SignedCheckRoot{
		Signature: []byte{0x01},
		MinterPub: nil,
	}

	err := mp.ValidateCheckRootSignature(sig, types.Hash512{0xaa})
	if err == nil {
		t.Error("should reject empty minter pubkey")
	}
}

// --- TxInclusionPriority 排序测试 ---

func TestTxInclusionPriority_Sort(t *testing.T) {
	// 构造不同优先级的交易
	txHighCredit := &TxEnvelope{
		TxID:             types.Hash512{0x01},
		HasCreditDestroy: true,
		TotalStakes:      100,
		TotalFee:         10,
	}
	txHighStakes := &TxEnvelope{
		TxID:             types.Hash512{0x02},
		HasCreditDestroy: false,
		TotalStakes:      500,
		TotalFee:         20,
	}
	txHighFee := &TxEnvelope{
		TxID:             types.Hash512{0x03},
		HasCreditDestroy: false,
		TotalStakes:      100,
		TotalFee:         1000,
	}
	txLow := &TxEnvelope{
		TxID:             types.Hash512{0x04},
		HasCreditDestroy: false,
		TotalStakes:      10,
		TotalFee:         1,
	}

	txs := []*TxEnvelope{txLow, txHighFee, txHighStakes, txHighCredit}

	sort.Slice(txs, func(i, j int) bool {
		return TxInclusionPriority(txs[i], txs[j])
	})

	// 排序结果：凭信销毁 > 高币权 > 高手续费 > 低优先级
	if txs[0].TxID != txHighCredit.TxID {
		t.Errorf("first should be credit-destroy tx, got %x", txs[0].TxID[0])
	}
	if txs[1].TxID != txHighStakes.TxID {
		t.Errorf("second should be high-stakes tx, got %x", txs[1].TxID[0])
	}
	if txs[2].TxID != txHighFee.TxID {
		t.Errorf("third should be high-fee tx, got %x", txs[2].TxID[0])
	}
	if txs[3].TxID != txLow.TxID {
		t.Errorf("fourth should be low-priority tx, got %x", txs[3].TxID[0])
	}
}

func TestTxInclusionPriority_SamePriority(t *testing.T) {
	tx1 := &TxEnvelope{
		TxID:             types.Hash512{0x01},
		HasCreditDestroy: false,
		TotalStakes:      100,
		TotalFee:         50,
	}
	tx2 := &TxEnvelope{
		TxID:             types.Hash512{0x02},
		HasCreditDestroy: false,
		TotalStakes:      100,
		TotalFee:         50,
	}

	// 完全相同优先级，不应 panic
	_ = TxInclusionPriority(tx1, tx2)
	_ = TxInclusionPriority(tx2, tx1)
}

// --- ShouldExcludeTx 不收录交易排除测试 ---

func TestShouldExcludeTx_IntraTxCollision(t *testing.T) {
	mp := NewMintingProtocol()

	// 构造两个输入的 TxID 前 20 字节相同的交易
	var txID1, txID2 types.Hash512
	for i := 0; i < 20; i++ {
		txID1[i] = byte(i)
		txID2[i] = byte(i) // 前 20 字节相同
	}
	txID1[20] = 0xaa // 后续字节不同
	txID2[20] = 0xbb

	inputs := []types.Hash512{txID1, txID2}

	exclude, reason := mp.ShouldExcludeTx(inputs, nil)
	if !exclude {
		t.Error("should exclude tx with intra-tx TxID prefix collision")
	}
	if reason == "" {
		t.Error("reason should not be empty")
	}
}

func TestShouldExcludeTx_NoCollision(t *testing.T) {
	mp := NewMintingProtocol()

	var txID1, txID2 types.Hash512
	txID1[0] = 0x01
	txID2[0] = 0x02

	inputs := []types.Hash512{txID1, txID2}

	exclude, _ := mp.ShouldExcludeTx(inputs, nil)
	if exclude {
		t.Error("should not exclude tx without collision")
	}
}

func TestShouldExcludeTx_LeafLayerCollision(t *testing.T) {
	mp := NewMintingProtocol()

	var txID1 types.Hash512
	txID1[0] = 0x42

	// 模拟 UTXO 叶子层中已有交易的 TxID 前 8 字节
	var existingPrefix [8]byte
	copy(existingPrefix[:], txID1[:8])
	leafPrefixes := [][8]byte{existingPrefix}

	inputs := []types.Hash512{txID1}

	exclude, reason := mp.ShouldExcludeTx(inputs, leafPrefixes)
	if !exclude {
		t.Error("should exclude tx with leaf-layer TxID prefix collision")
	}
	if reason == "" {
		t.Error("reason should not be empty")
	}
}

func TestShouldExcludeTx_NoLeafCollision(t *testing.T) {
	mp := NewMintingProtocol()

	var txID1 types.Hash512
	txID1[0] = 0x42

	var existingPrefix [8]byte
	existingPrefix[0] = 0x99 // 不同前缀

	leafPrefixes := [][8]byte{existingPrefix}
	inputs := []types.Hash512{txID1}

	exclude, _ := mp.ShouldExcludeTx(inputs, leafPrefixes)
	if exclude {
		t.Error("should not exclude tx without leaf collision")
	}
}

func TestShouldExcludeTx_SingleInput(t *testing.T) {
	mp := NewMintingProtocol()

	inputs := []types.Hash512{{0x01}}

	exclude, _ := mp.ShouldExcludeTx(inputs, nil)
	if exclude {
		t.Error("single input tx should not have intra-tx collision")
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/verification/ -run "TestNewMintingProtocol|TestValidateMintRequest|TestPrepareMintInfo|TestValidateCoinbase|TestPrepareCheckRootData|TestValidateCheckRootSignature|TestTxInclusionPriority|TestShouldExcludeTx"
```

预期输出：编译失败，`MintingProtocol`、`NewMintingProtocol`、`TxInclusionPriority`、`ShouldExcludeTx` 等未定义。

### Step 3: 写最小实现

创建 `internal/verification/minting.go`：

```go
package verification

import (
	"errors"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// 铸造相关常量
const (
	// InitialMintReward 初始铸造奖励（单位：最小币金）。
	// 实际系统中会随区块高度递减。
	InitialMintReward int64 = 5000000000

	// IntraTxPrefixLen 同笔交易内 TxID 碰撞检查的前缀长度（20 字节）。
	IntraTxPrefixLen = 20

	// LeafLayerPrefixLen UTXO/UTCO 叶子层 TxID 碰撞检查的前缀长度（8 字节）。
	LeafLayerPrefixLen = 8

	// StakesThresholdFactor 币权阈值倍数——候选区块的总币权销毁超过主块 3 倍则候选胜出。
	StakesThresholdFactor = 3
)

// 铸造相关错误
var (
	ErrNilMintRequest     = errors.New("mint request is nil")
	ErrEmptyProof         = errors.New("mint proof is empty")
	ErrEmptyMinterPub     = errors.New("minter public key is empty")
	ErrNegativeIndex      = errors.New("candidate index is negative")
	ErrEmptyCoinbaseTx    = errors.New("coinbase transaction is empty")
	ErrZeroCoinbaseID     = errors.New("coinbase id is zero")
	ErrEmptySignature     = errors.New("signature is empty")
	ErrIntraTxCollision   = errors.New("intra-transaction txid prefix collision")
	ErrLeafLayerCollision = errors.New("utxo/utco leaf-layer txid prefix collision")
)

// MintingProtocol 铸造集成协议管理器。
// 管理铸造流程的状态和验证逻辑。
type MintingProtocol struct{}

// NewMintingProtocol 创建新的铸造协议管理器。
func NewMintingProtocol() *MintingProtocol {
	return &MintingProtocol{}
}

// ValidateMintRequest 验证铸造申请的基本字段。
// 真正的择优池验证由共识层处理，这里只做字段合法性检查。
func (mp *MintingProtocol) ValidateMintRequest(req *MintRequest) error {
	if req == nil {
		return ErrNilMintRequest
	}
	if len(req.Proof) == 0 {
		return ErrEmptyProof
	}
	if len(req.MinterPub) == 0 {
		return ErrEmptyMinterPub
	}
	if req.CandidateIndex < 0 {
		return ErrNegativeIndex
	}
	return nil
}

// PrepareMintInfo 准备铸造信息——汇总交易费、计算铸造奖励。
func (mp *MintingProtocol) PrepareMintInfo(verifiedTxs []*TxEnvelope, height int, addrs ServiceAddresses) *MintInfo {
	var totalFees int64
	for _, tx := range verifiedTxs {
		totalFees += tx.TotalFee
	}

	// 铸造奖励：简化模型，实际会根据高度递减
	mintAmount := calcMintReward(height)

	return &MintInfo{
		TotalFees:       totalFees,
		RewardAddresses: addrs,
		MintAmount:      mintAmount,
		FeeBurned:       0,
	}
}

// calcMintReward 根据区块高度计算铸造奖励。
// 简化模型：每 BlocksPerYear 个区块奖励减半。
func calcMintReward(height int) int64 {
	reward := InitialMintReward
	halvings := height / int(types.BlocksPerYear)
	for i := 0; i < halvings && reward > 0; i++ {
		reward /= 2
	}
	if reward == 0 {
		reward = 1 // 最低奖励
	}
	return reward
}

// ValidateCoinbase 验证 Coinbase 交易的基本字段。
// 详细的金额验证由区块链核心层处理。
func (mp *MintingProtocol) ValidateCoinbase(cb *CoinbaseSubmission, info *MintInfo) error {
	if cb == nil {
		return ErrEmptyCoinbaseTx
	}
	if len(cb.CoinbaseTx) == 0 {
		return ErrEmptyCoinbaseTx
	}
	if cb.CoinbaseID.IsZero() {
		return ErrZeroCoinbaseID
	}
	return nil
}

// PrepareCheckRootData 准备校验根数据。
// 计算交易哈希树根、构建 Coinbase Merkle 路径、计算 CheckRoot。
//
// CheckRoot = SHA-512( TreeRoot || UTXOFingerprint || UTCOFingerprint )
func (mp *MintingProtocol) PrepareCheckRootData(
	coinbaseID types.Hash512,
	txIDs []types.Hash512,
	utxoFP, utcoFP types.Hash512,
) *CheckRootData {
	// 构建完整交易列表（Coinbase 在首位）
	allIDs := make([]types.Hash512, 0, len(txIDs)+1)
	allIDs = append(allIDs, coinbaseID)
	allIDs = append(allIDs, txIDs...)

	// 计算简化版交易哈希树根
	treeRoot := calcSimpleMerkleRoot(allIDs)

	// 构建 Coinbase 的 Merkle 路径（简化版）
	merklePath := buildSimpleMerklePath(allIDs, 0)

	// 计算 CheckRoot = SHA-512( TreeRoot || UTXOFp || UTCOFp )
	buf := make([]byte, types.HashLen*3)
	copy(buf[0:types.HashLen], treeRoot[:])
	copy(buf[types.HashLen:types.HashLen*2], utxoFP[:])
	copy(buf[types.HashLen*2:types.HashLen*3], utcoFP[:])
	checkRoot := crypto.SHA512Sum(buf)

	return &CheckRootData{
		CoinbaseMerklePath: merklePath,
		TreeRoot:           treeRoot,
		UTXOFingerprint:    utxoFP,
		UTCOFingerprint:    utcoFP,
		CheckRoot:          checkRoot,
	}
}

// calcSimpleMerkleRoot 计算简化的 Merkle 树根。
// 将所有哈希逐层两两合并直到只剩一个。
func calcSimpleMerkleRoot(hashes []types.Hash512) types.Hash512 {
	if len(hashes) == 0 {
		return types.Hash512{}
	}
	if len(hashes) == 1 {
		return hashes[0]
	}

	// 复制一份避免修改原数组
	current := make([]types.Hash512, len(hashes))
	copy(current, hashes)

	for len(current) > 1 {
		var next []types.Hash512
		for i := 0; i < len(current); i += 2 {
			if i+1 < len(current) {
				// 合并相邻两个
				buf := make([]byte, types.HashLen*2)
				copy(buf[:types.HashLen], current[i][:])
				copy(buf[types.HashLen:], current[i+1][:])
				next = append(next, crypto.SHA512Sum(buf))
			} else {
				// 奇数个，最后一个自己与自己合并
				buf := make([]byte, types.HashLen*2)
				copy(buf[:types.HashLen], current[i][:])
				copy(buf[types.HashLen:], current[i][:])
				next = append(next, crypto.SHA512Sum(buf))
			}
		}
		current = next
	}

	return current[0]
}

// buildSimpleMerklePath 构建指定叶子节点的 Merkle 验证路径。
func buildSimpleMerklePath(hashes []types.Hash512, targetIndex int) []types.Hash512 {
	if len(hashes) <= 1 {
		return nil
	}

	current := make([]types.Hash512, len(hashes))
	copy(current, hashes)
	idx := targetIndex
	var path []types.Hash512

	for len(current) > 1 {
		var next []types.Hash512
		newIdx := idx / 2

		for i := 0; i < len(current); i += 2 {
			var left, right types.Hash512
			left = current[i]
			if i+1 < len(current) {
				right = current[i+1]
			} else {
				right = current[i] // 奇数个，自己与自己
			}

			// 如果目标在这一对中，记录兄弟节点
			if i == idx || i+1 == idx {
				if i == idx {
					path = append(path, right)
				} else {
					path = append(path, left)
				}
			}

			buf := make([]byte, types.HashLen*2)
			copy(buf[:types.HashLen], left[:])
			copy(buf[types.HashLen:], right[:])
			next = append(next, crypto.SHA512Sum(buf))
		}

		current = next
		idx = newIdx
	}

	return path
}

// ValidateCheckRootSignature 验证铸造者对 CheckRoot 的签名。
// 简化实现：只检查签名和公钥非空。
// 实际签名验证由 pkg/crypto 提供。
func (mp *MintingProtocol) ValidateCheckRootSignature(sig *SignedCheckRoot, checkRoot types.Hash512) error {
	if sig == nil || len(sig.Signature) == 0 {
		return ErrEmptySignature
	}
	if len(sig.MinterPub) == 0 {
		return ErrEmptyMinterPub
	}
	// TODO: 使用 pkg/crypto 验证签名
	// return crypto.Verify(sig.MinterPub, checkRoot[:], sig.Signature)
	return nil
}

// TxInclusionPriority 交易收录优先级比较函数。
// 返回 true 表示 a 的优先级高于 b。
//
// 优先级规则：
//  1. 有凭信提前销毁的交易优先
//  2. 币权（币龄*币量）销毁更多的交易优先
//  3. 手续费更高的交易优先
func TxInclusionPriority(a, b *TxEnvelope) bool {
	// 规则 1：凭信提前销毁优先
	if a.HasCreditDestroy != b.HasCreditDestroy {
		return a.HasCreditDestroy
	}

	// 规则 2：更高币权销毁优先
	if a.TotalStakes != b.TotalStakes {
		return a.TotalStakes > b.TotalStakes
	}

	// 规则 3：更高手续费优先
	return a.TotalFee > b.TotalFee
}

// ShouldExcludeTx 检查交易是否应被排除收录。
// inputTxIDs 为交易所有输入引用的 TxID 列表。
// leafPrefixes 为 UTXO/UTCO 叶子层已有交易的 TxID 前 8 字节列表。
//
// 排除条件：
//  1. 同笔交易内 TxID 前 20 字节碰撞
//  2. UTXO/UTCO 叶子层 TxID 前 8 字节碰撞
func (mp *MintingProtocol) ShouldExcludeTx(inputTxIDs []types.Hash512, leafPrefixes [][8]byte) (bool, string) {
	// 检查条件 1：同笔交易内 TxID 前 20 字节碰撞
	if len(inputTxIDs) > 1 {
		prefixSet := make(map[[IntraTxPrefixLen]byte]bool)
		for _, txID := range inputTxIDs {
			var prefix [IntraTxPrefixLen]byte
			copy(prefix[:], txID[:IntraTxPrefixLen])
			if prefixSet[prefix] {
				return true, "intra-transaction txid prefix collision (first 20 bytes)"
			}
			prefixSet[prefix] = true
		}
	}

	// 检查条件 2：UTXO/UTCO 叶子层 TxID 前 8 字节碰撞
	if len(leafPrefixes) > 0 {
		leafSet := make(map[[LeafLayerPrefixLen]byte]bool)
		for _, p := range leafPrefixes {
			leafSet[p] = true
		}

		for _, txID := range inputTxIDs {
			var prefix [LeafLayerPrefixLen]byte
			copy(prefix[:], txID[:LeafLayerPrefixLen])
			if leafSet[prefix] {
				return true, "utxo/utco leaf-layer txid prefix collision (first 8 bytes)"
			}
		}
	}

	return false, ""
}
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/verification/ -run "TestNewMintingProtocol|TestValidateMintRequest|TestPrepareMintInfo|TestValidateCoinbase|TestPrepareCheckRootData|TestValidateCheckRootSignature|TestTxInclusionPriority|TestShouldExcludeTx"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/verification/minting.go internal/verification/minting_test.go
git commit -m "feat(verification): add minting protocol with tx priority sorting and exclusion rules"
```

---

## Task 6: 区块发布协议 (internal/verification/publish.go)

**Files:**
- Create: `internal/verification/publish.go`
- Test: `internal/verification/publish_test.go`

本 Task 实现三阶段区块发布协议——区块证明创建与验证、区块概要（截断的交易 ID）、本地已验证交易匹配与缺失检测。

### Step 1: 写失败测试

创建 `internal/verification/publish_test.go`：

```go
package verification

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// --- TruncateTxID 测试 ---

func TestTruncateTxID(t *testing.T) {
	var txID types.Hash512
	for i := range txID {
		txID[i] = byte(i)
	}

	truncated := TruncateTxID(txID)

	// 应取前 16 字节
	for i := 0; i < TruncatedIDLen; i++ {
		if truncated[i] != byte(i) {
			t.Errorf("truncated[%d] = %d, want %d", i, truncated[i], i)
		}
	}
}

func TestTruncateTxID_Deterministic(t *testing.T) {
	txID := types.Hash512{0xab, 0xcd}

	t1 := TruncateTxID(txID)
	t2 := TruncateTxID(txID)

	if t1 != t2 {
		t.Error("TruncateTxID should be deterministic")
	}
}

func TestTruncateTxID_DifferentInputs(t *testing.T) {
	txID1 := types.Hash512{0x01}
	txID2 := types.Hash512{0x02}

	t1 := TruncateTxID(txID1)
	t2 := TruncateTxID(txID2)

	if t1 == t2 {
		t.Error("different TxIDs should produce different truncated IDs")
	}
}

// --- CreateBlockSummary 测试 ---

func TestCreateBlockSummary(t *testing.T) {
	txIDs := []types.Hash512{
		{0x01, 0x02, 0x03},
		{0x04, 0x05, 0x06},
		{0x07, 0x08, 0x09},
	}

	summary := CreateBlockSummary(100, txIDs)

	if summary.Height != 100 {
		t.Errorf("Height = %d, want 100", summary.Height)
	}
	if len(summary.TruncatedTxIDs) != 3 {
		t.Errorf("TruncatedTxIDs length = %d, want 3", len(summary.TruncatedTxIDs))
	}

	// 验证截断正确
	for i, txID := range txIDs {
		expected := TruncateTxID(txID)
		if summary.TruncatedTxIDs[i] != expected {
			t.Errorf("TruncatedTxIDs[%d] mismatch", i)
		}
	}
}

func TestCreateBlockSummary_Empty(t *testing.T) {
	summary := CreateBlockSummary(50, nil)

	if summary.Height != 50 {
		t.Errorf("Height = %d, want 50", summary.Height)
	}
	if len(summary.TruncatedTxIDs) != 0 {
		t.Errorf("TruncatedTxIDs should be empty, got %d", len(summary.TruncatedTxIDs))
	}
}

// --- CreateBlockProof 测试 ---

func TestCreateBlockProof(t *testing.T) {
	coinbaseTx := []byte{0x01, 0x02}
	merklePath := []types.Hash512{{0x10}, {0x20}}
	sig := []byte{0x30, 0x40}
	header := BlockHeader{
		Version: 1,
		Height:  100,
	}

	proof := CreateBlockProof(coinbaseTx, merklePath, sig, header)

	if string(proof.CoinbaseTx) != string(coinbaseTx) {
		t.Error("CoinbaseTx mismatch")
	}
	if len(proof.MerklePath) != 2 {
		t.Errorf("MerklePath length = %d, want 2", len(proof.MerklePath))
	}
	if string(proof.MinterSignature) != string(sig) {
		t.Error("MinterSignature mismatch")
	}
	if proof.Header.Height != 100 {
		t.Errorf("Header.Height = %d, want 100", proof.Header.Height)
	}
}

func TestCreateBlockProof_NilFields(t *testing.T) {
	proof := CreateBlockProof(nil, nil, nil, BlockHeader{})

	if proof.CoinbaseTx != nil {
		t.Error("CoinbaseTx should be nil")
	}
	if proof.MerklePath != nil {
		t.Error("MerklePath should be nil")
	}
}

// --- VerifyBlockProof 测试 ---

func TestVerifyBlockProof_Valid(t *testing.T) {
	proof := &BlockProof{
		CoinbaseTx:      []byte{0x01},
		MerklePath:      []types.Hash512{{0x10}},
		MinterSignature: []byte{0x30},
		Header: BlockHeader{
			Version: 1,
			Height:  100,
		},
	}

	err := VerifyBlockProof(proof)
	if err != nil {
		t.Errorf("VerifyBlockProof() error = %v", err)
	}
}

func TestVerifyBlockProof_NilProof(t *testing.T) {
	err := VerifyBlockProof(nil)
	if err == nil {
		t.Error("VerifyBlockProof(nil) should return error")
	}
}

func TestVerifyBlockProof_EmptyCoinbase(t *testing.T) {
	proof := &BlockProof{
		CoinbaseTx:      nil,
		MerklePath:      []types.Hash512{{0x10}},
		MinterSignature: []byte{0x30},
		Header:          BlockHeader{Version: 1, Height: 100},
	}

	err := VerifyBlockProof(proof)
	if err == nil {
		t.Error("should reject proof with empty coinbase")
	}
}

func TestVerifyBlockProof_EmptySignature(t *testing.T) {
	proof := &BlockProof{
		CoinbaseTx:      []byte{0x01},
		MerklePath:      []types.Hash512{{0x10}},
		MinterSignature: nil,
		Header:          BlockHeader{Version: 1, Height: 100},
	}

	err := VerifyBlockProof(proof)
	if err == nil {
		t.Error("should reject proof with empty signature")
	}
}

func TestVerifyBlockProof_ZeroVersion(t *testing.T) {
	proof := &BlockProof{
		CoinbaseTx:      []byte{0x01},
		MerklePath:      []types.Hash512{{0x10}},
		MinterSignature: []byte{0x30},
		Header:          BlockHeader{Version: 0, Height: 100},
	}

	err := VerifyBlockProof(proof)
	if err == nil {
		t.Error("should reject proof with zero version")
	}
}

// --- MatchLocalTxs 测试 ---

func TestMatchLocalTxs_AllPresent(t *testing.T) {
	// 本地有所有交易
	txIDs := []types.Hash512{{0x01}, {0x02}, {0x03}}
	summary := CreateBlockSummary(100, txIDs)

	localVerified := make(map[[TruncatedIDLen]byte]bool)
	for _, txID := range txIDs {
		localVerified[TruncateTxID(txID)] = true
	}

	missing := MatchLocalTxs(summary, localVerified)
	if len(missing) != 0 {
		t.Errorf("missing = %v, want empty", missing)
	}
}

func TestMatchLocalTxs_SomeMissing(t *testing.T) {
	txIDs := []types.Hash512{{0x01}, {0x02}, {0x03}}
	summary := CreateBlockSummary(100, txIDs)

	// 只有第一笔在本地
	localVerified := make(map[[TruncatedIDLen]byte]bool)
	localVerified[TruncateTxID(txIDs[0])] = true

	missing := MatchLocalTxs(summary, localVerified)
	if len(missing) != 2 {
		t.Errorf("missing count = %d, want 2", len(missing))
	}
	// 应返回索引 1 和 2
	if missing[0] != 1 || missing[1] != 2 {
		t.Errorf("missing = %v, want [1, 2]", missing)
	}
}

func TestMatchLocalTxs_AllMissing(t *testing.T) {
	txIDs := []types.Hash512{{0x01}, {0x02}}
	summary := CreateBlockSummary(100, txIDs)

	localVerified := make(map[[TruncatedIDLen]byte]bool)

	missing := MatchLocalTxs(summary, localVerified)
	if len(missing) != 2 {
		t.Errorf("missing count = %d, want 2", len(missing))
	}
}

func TestMatchLocalTxs_EmptySummary(t *testing.T) {
	summary := CreateBlockSummary(100, nil)
	localVerified := make(map[[TruncatedIDLen]byte]bool)

	missing := MatchLocalTxs(summary, localVerified)
	if len(missing) != 0 {
		t.Errorf("missing = %v, want empty", missing)
	}
}

// --- TxSyncRequest / TxSyncResponse 测试 ---

func TestNewTxSyncRequest(t *testing.T) {
	req := NewTxSyncRequest([]int{1, 3, 5})

	if len(req.MissingIndices) != 3 {
		t.Errorf("MissingIndices length = %d, want 3", len(req.MissingIndices))
	}
	if req.MissingIndices[0] != 1 || req.MissingIndices[1] != 3 || req.MissingIndices[2] != 5 {
		t.Errorf("MissingIndices = %v, want [1, 3, 5]", req.MissingIndices)
	}
}

func TestNewTxSyncResponse(t *testing.T) {
	txs := [][]byte{{0x01}, {0x02, 0x03}}
	resp := NewTxSyncResponse(txs)

	if len(resp.Transactions) != 2 {
		t.Errorf("Transactions length = %d, want 2", len(resp.Transactions))
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/verification/ -run "TestTruncateTxID|TestCreateBlockSummary|TestCreateBlockProof|TestVerifyBlockProof|TestMatchLocalTxs|TestNewTxSync"
```

预期输出：编译失败，`TruncateTxID`、`CreateBlockSummary`、`VerifyBlockProof` 等未定义。

### Step 3: 写最小实现

创建 `internal/verification/publish.go`：

```go
package verification

import (
	"errors"

	"github.com/cxio/evidcoin/pkg/types"
)

// TruncatedIDLen 截断的交易 ID 长度（16 字节）。
const TruncatedIDLen = 16

// 区块发布相关错误
var (
	ErrNilBlockProof      = errors.New("block proof is nil")
	ErrEmptyBlockCoinbase = errors.New("block proof coinbase is empty")
	ErrEmptyBlockSig      = errors.New("block proof signature is empty")
	ErrZeroBlockVersion   = errors.New("block header version is zero")
)

// TruncateTxID 将完整的交易 ID（64 字节）截断为前 16 字节。
// 用于区块概要阶段的传输优化。
func TruncateTxID(txID types.Hash512) [TruncatedIDLen]byte {
	var truncated [TruncatedIDLen]byte
	copy(truncated[:], txID[:TruncatedIDLen])
	return truncated
}

// CreateBlockSummary 创建区块概要。
// 将所有交易 ID 截断为前 16 字节。
func CreateBlockSummary(height uint64, txIDs []types.Hash512) *BlockSummary {
	truncated := make([][TruncatedIDLen]byte, len(txIDs))
	for i, txID := range txIDs {
		truncated[i] = TruncateTxID(txID)
	}
	return &BlockSummary{
		Height:         height,
		TruncatedTxIDs: truncated,
	}
}

// CreateBlockProof 创建区块证明（第一阶段：最小证明数据）。
func CreateBlockProof(coinbaseTx []byte, merklePath []types.Hash512, sig []byte, header BlockHeader) *BlockProof {
	return &BlockProof{
		CoinbaseTx:      coinbaseTx,
		MerklePath:      merklePath,
		MinterSignature: sig,
		Header:          header,
	}
}

// VerifyBlockProof 验证区块证明的基本字段完整性。
// 包括 Coinbase 非空、签名非空、区块头版本有效。
// 详细的 Merkle 路径验证和签名验证由后续流程处理。
func VerifyBlockProof(proof *BlockProof) error {
	if proof == nil {
		return ErrNilBlockProof
	}
	if len(proof.CoinbaseTx) == 0 {
		return ErrEmptyBlockCoinbase
	}
	if len(proof.MinterSignature) == 0 {
		return ErrEmptyBlockSig
	}
	if proof.Header.Version == 0 {
		return ErrZeroBlockVersion
	}
	return nil
}

// MatchLocalTxs 将区块概要中的截断 ID 与本地已验证交易匹配。
// 返回缺失交易在概要中的索引列表。
func MatchLocalTxs(summary *BlockSummary, localVerified map[[TruncatedIDLen]byte]bool) []int {
	var missing []int
	for i, truncID := range summary.TruncatedTxIDs {
		if !localVerified[truncID] {
			missing = append(missing, i)
		}
	}
	return missing
}

// TxSyncRequest 交易同步请求（第三阶段）。
// 包含缺失交易在区块中的索引列表。
type TxSyncRequest struct {
	MissingIndices []int // 缺失交易索引
}

// NewTxSyncRequest 创建交易同步请求。
func NewTxSyncRequest(missingIndices []int) *TxSyncRequest {
	return &TxSyncRequest{
		MissingIndices: missingIndices,
	}
}

// TxSyncResponse 交易同步响应。
// 包含请求的完整交易数据。
type TxSyncResponse struct {
	Transactions [][]byte // 交易原始字节列表
}

// NewTxSyncResponse 创建交易同步响应。
func NewTxSyncResponse(txs [][]byte) *TxSyncResponse {
	return &TxSyncResponse{
		Transactions: txs,
	}
}
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/verification/ -run "TestTruncateTxID|TestCreateBlockSummary|TestCreateBlockProof|TestVerifyBlockProof|TestMatchLocalTxs|TestNewTxSync"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/verification/publish.go internal/verification/publish_test.go
git commit -m "feat(verification): add three-phase block publication protocol with truncated tx ID matching"
```

---

## Task 7: 组间反馈与缓存接口 (internal/verification/feedback.go, internal/verification/cache.go)

**Files:**
- Create: `internal/verification/feedback.go`
- Create: `internal/verification/cache.go`
- Test: `internal/verification/feedback_test.go`
- Test: `internal/verification/cache_test.go`

本 Task 实现组间反馈追踪机制（投递日志、反馈处理、重新复核请求触发）和缓存查询接口（UTXO/UTCO/Script 缓存接口及内存 mock 实现）。

### Step 1: 写失败测试

创建 `internal/verification/feedback_test.go`：

```go
package verification

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// --- FeedbackTracker 构造测试 ---

func TestNewFeedbackTracker(t *testing.T) {
	ft := NewFeedbackTracker()
	if ft == nil {
		t.Fatal("NewFeedbackTracker() returned nil")
	}
}

// --- 投递日志 ---

func TestFeedbackTracker_LogDelivery(t *testing.T) {
	ft := NewFeedbackTracker()

	txID := types.Hash512{0x01}
	verifier := NodeID{0x10}
	targetTeam := NodeID{0x20}

	ft.LogDelivery(txID, verifier, targetTeam)

	records := ft.GetDeliveryRecords(txID)
	if len(records) != 1 {
		t.Fatalf("delivery records count = %d, want 1", len(records))
	}
	if records[0].VerifierID != verifier {
		t.Error("VerifierID mismatch")
	}
	if records[0].TargetGuardTeam != targetTeam {
		t.Error("TargetGuardTeam mismatch")
	}
}

func TestFeedbackTracker_LogDelivery_Multiple(t *testing.T) {
	ft := NewFeedbackTracker()

	txID := types.Hash512{0x01}
	v1, v2 := NodeID{0x10}, NodeID{0x11}
	team1, team2 := NodeID{0x20}, NodeID{0x21}

	ft.LogDelivery(txID, v1, team1)
	ft.LogDelivery(txID, v2, team2)

	records := ft.GetDeliveryRecords(txID)
	if len(records) != 2 {
		t.Fatalf("delivery records count = %d, want 2", len(records))
	}
}

func TestFeedbackTracker_GetDeliveryRecords_Unknown(t *testing.T) {
	ft := NewFeedbackTracker()

	records := ft.GetDeliveryRecords(types.Hash512{0xff})
	if len(records) != 0 {
		t.Errorf("unknown txID should return empty records, got %d", len(records))
	}
}

// --- 反馈处理 ---

func TestFeedbackTracker_HandleFeedback_Found(t *testing.T) {
	ft := NewFeedbackTracker()

	txID := types.Hash512{0x02}
	verifier := NodeID{0x10}
	targetTeam := NodeID{0x20}

	ft.LogDelivery(txID, verifier, targetTeam)

	fb := &FeedbackMessage{
		TxID:       txID,
		Valid:      false,
		SenderTeam: targetTeam,
	}

	req := ft.HandleFeedback(fb)
	if req == nil {
		t.Fatal("HandleFeedback() should return re-review request")
	}
	if req.TxID != txID {
		t.Error("ReReviewRequest TxID mismatch")
	}
	if req.OrigVerifier != verifier {
		t.Error("OrigVerifier mismatch")
	}
}

func TestFeedbackTracker_HandleFeedback_NoRecord(t *testing.T) {
	ft := NewFeedbackTracker()

	fb := &FeedbackMessage{
		TxID:       types.Hash512{0x03},
		Valid:      false,
		SenderTeam: NodeID{0x20},
	}

	req := ft.HandleFeedback(fb)
	if req != nil {
		t.Error("HandleFeedback() should return nil when no delivery record exists")
	}
}

func TestFeedbackTracker_HandleFeedback_ValidTx(t *testing.T) {
	ft := NewFeedbackTracker()

	txID := types.Hash512{0x04}
	ft.LogDelivery(txID, NodeID{0x10}, NodeID{0x20})

	fb := &FeedbackMessage{
		TxID:       txID,
		Valid:      true, // 确认有效，无需重新复核
		SenderTeam: NodeID{0x20},
	}

	req := ft.HandleFeedback(fb)
	if req != nil {
		t.Error("HandleFeedback() should return nil for valid feedback")
	}
}

func TestFeedbackTracker_HandleFeedback_WrongTeam(t *testing.T) {
	ft := NewFeedbackTracker()

	txID := types.Hash512{0x05}
	ft.LogDelivery(txID, NodeID{0x10}, NodeID{0x20})

	fb := &FeedbackMessage{
		TxID:       txID,
		Valid:      false,
		SenderTeam: NodeID{0x99}, // 不是投递的目标团队
	}

	req := ft.HandleFeedback(fb)
	if req != nil {
		t.Error("HandleFeedback() should return nil when sender team doesn't match delivery target")
	}
}

// --- FeedbackMessage 构造测试 ---

func TestFeedbackMessage_Fields(t *testing.T) {
	fb := &FeedbackMessage{
		TxID:         types.Hash512{0x01},
		Valid:        false,
		SenderTeam:   NodeID{0x10},
		OrigVerifier: NodeID{0x20},
	}

	if fb.Valid {
		t.Error("Valid should be false")
	}
	if fb.SenderTeam.IsZero() {
		t.Error("SenderTeam should not be zero")
	}
}

// --- ReReviewRequest 测试 ---

func TestReReviewRequest_Fields(t *testing.T) {
	req := &ReReviewRequest{
		TxID:         types.Hash512{0x01},
		OrigVerifier: NodeID{0x10},
		ReportedBy:   NodeID{0x20},
	}

	if req.TxID.IsZero() {
		t.Error("TxID should not be zero")
	}
	if req.OrigVerifier.IsZero() {
		t.Error("OrigVerifier should not be zero")
	}
}
```

创建 `internal/verification/cache_test.go`：

```go
package verification

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// --- MemoryUTXOCache 测试 ---

func TestMemoryUTXOCache_StoreAndLookup(t *testing.T) {
	cache := NewMemoryUTXOCache()

	op := OutPoint{TxID: types.Hash512{0x01}, Index: 0}
	entry := &UTXOEntryInfo{
		Amount:     1000,
		Address:    types.PubKeyHash{0x10},
		Height:     100,
		LockScript: []byte{0x76},
		Config:     byte(types.OutTypeCoin),
	}

	cache.Store(op, entry)

	got, err := cache.Lookup(op)
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if got.Amount != 1000 {
		t.Errorf("Amount = %d, want 1000", got.Amount)
	}
	if got.Height != 100 {
		t.Errorf("Height = %d, want 100", got.Height)
	}
}

func TestMemoryUTXOCache_Lookup_NotFound(t *testing.T) {
	cache := NewMemoryUTXOCache()

	op := OutPoint{TxID: types.Hash512{0x99}, Index: 0}
	_, err := cache.Lookup(op)
	if err != ErrNotFound {
		t.Errorf("Lookup() error = %v, want ErrNotFound", err)
	}
}

func TestMemoryUTXOCache_Delete(t *testing.T) {
	cache := NewMemoryUTXOCache()

	op := OutPoint{TxID: types.Hash512{0x01}, Index: 0}
	entry := &UTXOEntryInfo{Amount: 1000, LockScript: []byte{0x76}}

	cache.Store(op, entry)
	cache.Delete(op)

	_, err := cache.Lookup(op)
	if err != ErrNotFound {
		t.Errorf("Lookup() after Delete() error = %v, want ErrNotFound", err)
	}
}

func TestMemoryUTXOCache_Count(t *testing.T) {
	cache := NewMemoryUTXOCache()

	if cache.Count() != 0 {
		t.Errorf("Count() = %d, want 0", cache.Count())
	}

	cache.Store(OutPoint{TxID: types.Hash512{0x01}, Index: 0}, &UTXOEntryInfo{Amount: 100, LockScript: []byte{0x76}})
	cache.Store(OutPoint{TxID: types.Hash512{0x02}, Index: 0}, &UTXOEntryInfo{Amount: 200, LockScript: []byte{0x76}})

	if cache.Count() != 2 {
		t.Errorf("Count() = %d, want 2", cache.Count())
	}
}

// --- MemoryUTCOCache 测试 ---

func TestMemoryUTCOCache_StoreAndLookup(t *testing.T) {
	cache := NewMemoryUTCOCache()

	op := OutPoint{TxID: types.Hash512{0x01}, Index: 0}
	entry := &UTCOEntryInfo{
		Address:         types.PubKeyHash{0x10},
		Height:          100,
		RemainingTransfers: 5,
		LockScript:      []byte{0x76},
		Config:          byte(types.OutTypeCredit),
	}

	cache.Store(op, entry)

	got, err := cache.Lookup(op)
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if got.RemainingTransfers != 5 {
		t.Errorf("RemainingTransfers = %d, want 5", got.RemainingTransfers)
	}
}

func TestMemoryUTCOCache_Lookup_NotFound(t *testing.T) {
	cache := NewMemoryUTCOCache()

	op := OutPoint{TxID: types.Hash512{0x99}, Index: 0}
	_, err := cache.Lookup(op)
	if err != ErrNotFound {
		t.Errorf("Lookup() error = %v, want ErrNotFound", err)
	}
}

// --- MemoryScriptCache 测试 ---

func TestMemoryScriptCache_StoreAndLookup(t *testing.T) {
	cache := NewMemoryScriptCache()

	ref := ScriptRef{TxID: types.Hash512{0x01}, OutputIndex: 0}
	script := []byte{0x76, 0xa9, 0x14}

	err := cache.Store(ref, script)
	if err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	got, err := cache.Lookup(ref)
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if string(got) != string(script) {
		t.Errorf("Lookup() = %x, want %x", got, script)
	}
}

func TestMemoryScriptCache_Lookup_NotFound(t *testing.T) {
	cache := NewMemoryScriptCache()

	ref := ScriptRef{TxID: types.Hash512{0x99}, OutputIndex: 0}
	_, err := cache.Lookup(ref)
	if err != ErrNotFound {
		t.Errorf("Lookup() error = %v, want ErrNotFound", err)
	}
}

func TestMemoryScriptCache_Evict(t *testing.T) {
	cache := NewMemoryScriptCache()

	ref := ScriptRef{TxID: types.Hash512{0x01}, OutputIndex: 0}
	cache.Store(ref, []byte{0x76})

	err := cache.Evict(ref)
	if err != nil {
		t.Fatalf("Evict() error = %v", err)
	}

	_, err = cache.Lookup(ref)
	if err != ErrNotFound {
		t.Errorf("Lookup() after Evict() error = %v, want ErrNotFound", err)
	}
}

func TestMemoryScriptCache_StoreEmpty(t *testing.T) {
	cache := NewMemoryScriptCache()

	ref := ScriptRef{TxID: types.Hash512{0x01}, OutputIndex: 0}
	err := cache.Store(ref, nil)
	if err == nil {
		t.Error("Store(nil) should return error")
	}
}

// --- ScriptRef 测试 ---

func TestScriptRef_Key(t *testing.T) {
	ref1 := ScriptRef{TxID: types.Hash512{0x01}, OutputIndex: 0}
	ref2 := ScriptRef{TxID: types.Hash512{0x01}, OutputIndex: 0}
	ref3 := ScriptRef{TxID: types.Hash512{0x01}, OutputIndex: 1}

	if ref1.Key() != ref2.Key() {
		t.Error("same ScriptRef should have same Key()")
	}
	if ref1.Key() == ref3.Key() {
		t.Error("different ScriptRef should have different Key()")
	}
}

// --- 接口满足测试 ---

func TestCacheInterfaces_Compile(t *testing.T) {
	var _ UTXOCacheLookup = (*MemoryUTXOCache)(nil)
	var _ UTCOCacheLookup = (*MemoryUTCOCache)(nil)
	var _ ScriptCacheLookup = (*MemoryScriptCache)(nil)

	t.Log("all cache interfaces compile correctly")
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/verification/ -run "TestNewFeedbackTracker|TestFeedbackTracker|TestFeedbackMessage|TestReReviewRequest|TestMemoryUTXOCache|TestMemoryUTCOCache|TestMemoryScriptCache|TestScriptRef|TestCacheInterfaces"
```

预期输出：编译失败，`FeedbackTracker`、`FeedbackMessage`、`MemoryUTXOCache` 等未定义。

### Step 3: 写最小实现

创建 `internal/verification/feedback.go`：

```go
package verification

import (
	"sync"

	"github.com/cxio/evidcoin/pkg/types"
)

// FeedbackMessage 组间反馈消息。
// 当接收 Guard 所属团队判定某交易无效时，向原始投递方发送反馈。
type FeedbackMessage struct {
	TxID         types.Hash512 // 交易 ID
	Valid        bool          // 交易是否有效
	SenderTeam   NodeID        // 发送反馈的团队 ID
	OrigVerifier NodeID        // 原始投递该交易的校验者
}

// DeliveryRecord 投递记录，记录一次跨团队交易投递。
type DeliveryRecord struct {
	VerifierID      NodeID // 执行投递的校验者 ID
	TargetGuardTeam NodeID // 目标 Guard 所属团队 ID
}

// ReReviewRequest 重新复核请求。
// 当收到组间负面反馈时，由 FeedbackTracker 生成，通知管理层启动扩展复核。
type ReReviewRequest struct {
	TxID         types.Hash512 // 交易 ID
	OrigVerifier NodeID        // 原始投递该交易的校验者
	ReportedBy   NodeID        // 发送反馈的团队 ID
}

// FeedbackTracker 组间反馈追踪器。
// 维护每笔交易的投递记录，用于在收到负面反馈时追溯到原始校验者。
// 线程安全。
type FeedbackTracker struct {
	mu          sync.RWMutex
	deliveryLog map[types.Hash512][]DeliveryRecord // key: 交易 ID
}

// NewFeedbackTracker 创建新的反馈追踪器。
func NewFeedbackTracker() *FeedbackTracker {
	return &FeedbackTracker{
		deliveryLog: make(map[types.Hash512][]DeliveryRecord),
	}
}

// LogDelivery 记录一次跨团队交易投递。
func (ft *FeedbackTracker) LogDelivery(txID types.Hash512, verifierID, targetTeam NodeID) {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	ft.deliveryLog[txID] = append(ft.deliveryLog[txID], DeliveryRecord{
		VerifierID:      verifierID,
		TargetGuardTeam: targetTeam,
	})
}

// GetDeliveryRecords 获取指定交易的投递记录。
func (ft *FeedbackTracker) GetDeliveryRecords(txID types.Hash512) []DeliveryRecord {
	ft.mu.RLock()
	defer ft.mu.RUnlock()

	records := ft.deliveryLog[txID]
	if len(records) == 0 {
		return nil
	}

	// 返回副本
	result := make([]DeliveryRecord, len(records))
	copy(result, records)
	return result
}

// HandleFeedback 处理组间反馈消息。
// 如果反馈为负面（Valid=false），查找投递记录，返回重新复核请求。
// 如果反馈为正面、无投递记录、或发送方不匹配，返回 nil。
func (ft *FeedbackTracker) HandleFeedback(fb *FeedbackMessage) *ReReviewRequest {
	if fb == nil || fb.Valid {
		return nil
	}

	ft.mu.RLock()
	defer ft.mu.RUnlock()

	records, exists := ft.deliveryLog[fb.TxID]
	if !exists || len(records) == 0 {
		return nil
	}

	// 查找匹配的投递记录（发送团队必须是投递的目标团队）
	for _, record := range records {
		if record.TargetGuardTeam == fb.SenderTeam {
			return &ReReviewRequest{
				TxID:         fb.TxID,
				OrigVerifier: record.VerifierID,
				ReportedBy:   fb.SenderTeam,
			}
		}
	}

	return nil
}
```

创建 `internal/verification/cache.go`：

```go
package verification

import (
	"errors"
	"fmt"
	"sync"

	"github.com/cxio/evidcoin/pkg/types"
)

// UTCOEntryInfo UTCO 条目简要信息（供校验使用）。
type UTCOEntryInfo struct {
	Address            types.PubKeyHash // 接收者公钥哈希
	Height             uint64           // 创建时区块高度
	RemainingTransfers int              // 剩余可转移次数
	LockScript         []byte           // 锁定脚本
	Config             byte             // 输出配置字节
}

// ScriptRef 外部脚本引用标识。
type ScriptRef struct {
	TxID        types.Hash512 // 脚本所在交易的 ID
	OutputIndex uint16        // 脚本所在输出的序号
}

// Key 返回 ScriptRef 用作 map 键的字符串。
func (r ScriptRef) Key() string {
	return fmt.Sprintf("%s:%d", r.TxID.String(), r.OutputIndex)
}

// UTXOCacheLookup UTXO 缓存查询接口。
type UTXOCacheLookup interface {
	// Lookup 查询指定输出点的 UTXO 条目。
	Lookup(op OutPoint) (*UTXOEntryInfo, error)
}

// UTCOCacheLookup UTCO 缓存查询接口。
type UTCOCacheLookup interface {
	// Lookup 查询指定输出点的 UTCO 条目。
	Lookup(op OutPoint) (*UTCOEntryInfo, error)
}

// CacheUpdater 缓存更新接口。
type CacheUpdater interface {
	// Update 根据交易更新缓存（添加新输出、标记已花费输出）。
	Update(tx *TxEnvelope) error
}

// ScriptCacheLookup 外部脚本缓存接口。
type ScriptCacheLookup interface {
	// Lookup 按脚本引用查询缓存的脚本字节码。
	Lookup(ref ScriptRef) ([]byte, error)

	// Store 缓存外部脚本。
	Store(ref ScriptRef, script []byte) error

	// Evict 清理指定的脚本缓存项。
	Evict(ref ScriptRef) error
}

// 缓存错误
var (
	ErrEmptyScript = errors.New("script is empty")
)

// --- 内存实现（用于测试） ---

// MemoryUTXOCache UTXO 缓存的内存实现。
type MemoryUTXOCache struct {
	mu      sync.RWMutex
	entries map[string]*UTXOEntryInfo
}

// NewMemoryUTXOCache 创建内存 UTXO 缓存。
func NewMemoryUTXOCache() *MemoryUTXOCache {
	return &MemoryUTXOCache{
		entries: make(map[string]*UTXOEntryInfo),
	}
}

// Lookup 查询 UTXO 条目。
func (c *MemoryUTXOCache) Lookup(op OutPoint) (*UTXOEntryInfo, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[op.Key()]
	if !exists {
		return nil, ErrNotFound
	}
	return entry, nil
}

// Store 存储 UTXO 条目。
func (c *MemoryUTXOCache) Store(op OutPoint, entry *UTXOEntryInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[op.Key()] = entry
}

// Delete 删除 UTXO 条目。
func (c *MemoryUTXOCache) Delete(op OutPoint) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, op.Key())
}

// Count 返回缓存条目数量。
func (c *MemoryUTXOCache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// MemoryUTCOCache UTCO 缓存的内存实现。
type MemoryUTCOCache struct {
	mu      sync.RWMutex
	entries map[string]*UTCOEntryInfo
}

// NewMemoryUTCOCache 创建内存 UTCO 缓存。
func NewMemoryUTCOCache() *MemoryUTCOCache {
	return &MemoryUTCOCache{
		entries: make(map[string]*UTCOEntryInfo),
	}
}

// Lookup 查询 UTCO 条目。
func (c *MemoryUTCOCache) Lookup(op OutPoint) (*UTCOEntryInfo, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[op.Key()]
	if !exists {
		return nil, ErrNotFound
	}
	return entry, nil
}

// Store 存储 UTCO 条目。
func (c *MemoryUTCOCache) Store(op OutPoint, entry *UTCOEntryInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[op.Key()] = entry
}

// Delete 删除 UTCO 条目。
func (c *MemoryUTCOCache) Delete(op OutPoint) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, op.Key())
}

// Count 返回缓存条目数量。
func (c *MemoryUTCOCache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// MemoryScriptCache 外部脚本缓存的内存实现。
type MemoryScriptCache struct {
	mu      sync.RWMutex
	scripts map[string][]byte
}

// NewMemoryScriptCache 创建内存脚本缓存。
func NewMemoryScriptCache() *MemoryScriptCache {
	return &MemoryScriptCache{
		scripts: make(map[string][]byte),
	}
}

// Lookup 查询脚本。
func (c *MemoryScriptCache) Lookup(ref ScriptRef) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	script, exists := c.scripts[ref.Key()]
	if !exists {
		return nil, ErrNotFound
	}

	// 返回副本
	result := make([]byte, len(script))
	copy(result, script)
	return result, nil
}

// Store 缓存脚本。
func (c *MemoryScriptCache) Store(ref ScriptRef, script []byte) error {
	if len(script) == 0 {
		return ErrEmptyScript
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 存储副本
	stored := make([]byte, len(script))
	copy(stored, script)
	c.scripts[ref.Key()] = stored
	return nil
}

// Evict 清理脚本缓存项。
func (c *MemoryScriptCache) Evict(ref ScriptRef) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.scripts, ref.Key())
	return nil
}
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/verification/ -run "TestNewFeedbackTracker|TestFeedbackTracker|TestFeedbackMessage|TestReReviewRequest|TestMemoryUTXOCache|TestMemoryUTCOCache|TestMemoryScriptCache|TestScriptRef|TestCacheInterfaces"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/verification/feedback.go internal/verification/feedback_test.go internal/verification/cache.go internal/verification/cache_test.go
git commit -m "feat(verification): add inter-team feedback tracking and UTXO/UTCO/Script cache interfaces with memory implementations"
```
