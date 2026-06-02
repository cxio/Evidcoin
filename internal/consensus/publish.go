package consensus

// 区块发布三段状态机（第 12 章 §2，proposal 12 §2）。
//
// 三段不实现 P2P 线格式与网络传输，只定义状态类型与迁移逻辑。
// 完整合法性判断完成前，择优池候选者不应停止发布区块（调用方职责）。
//
// 证明包（阶段 1 广播）字段由第 13 章 DEC-0601 定义；
// 区块概要（阶段 2）TxID 截前 16 字节优化由第 15 章 DEC-0602 定义；
// 本文件只建模三段状态，不重复字段编码。

// PublishStage 是区块发布的三个阶段。
type PublishStage uint8

const (
	// PublishStageIdle 表示尚未开始发布（零值）。
	PublishStageIdle PublishStage = 0
	// PublishStageBlockProof 是阶段 1：广播区块证明。
	// 触发条件：Coinbase 合法 + 区块头合法，即先行转播。
	PublishStageBlockProof PublishStage = 1
	// PublishStageBlockSummary 是阶段 2：发布区块概要。
	// 内容：全部 TxID 序列（可优化为每个 TxID 前 16 字节）。
	PublishStageBlockSummary PublishStage = 2
	// PublishStageTxData 是阶段 3：同步交易数据。
	// 内容：补足各节点缺失的少量交易体。
	PublishStageTxData PublishStage = 3
)

// PublishState 表示某区块在本节点的发布状态。
// 阶段严格递增（Idle → BlockProof → BlockSummary → TxData），
// 不可回退；调用方负责保证并发安全。
type PublishState struct {
	stage PublishStage
}

// Stage 返回当前发布阶段。
func (s *PublishState) Stage() PublishStage { return s.stage }

// AdvanceTo 将状态推进到 next 阶段。若 next 不大于当前阶段（即回退或重复），
// 返回 ErrPublishStageInvalid；否则更新并返回 nil。
func (s *PublishState) AdvanceTo(next PublishStage) error {
	if next <= s.stage {
		return ErrPublishStageInvalid
	}
	s.stage = next
	return nil
}

// MustAdvanceTo 同 AdvanceTo，但在无效迁移时 panic（仅供测试与断言场景）。
func (s *PublishState) MustAdvanceTo(next PublishStage) {
	if err := s.AdvanceTo(next); err != nil {
		panic(err)
	}
}

// IsComplete 报告是否已完成阶段 3（交易数据同步）。
func (s *PublishState) IsComplete() bool {
	return s.stage == PublishStageTxData
}

// NewPublishState 创建处于 Idle 阶段的发布状态。
func NewPublishState() *PublishState {
	return &PublishState{stage: PublishStageIdle}
}
