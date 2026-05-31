package tx

// AuthFlag 是授权种类 8 位标记（DEC-0102 §2 / conception 附.交易.md#授权种类）。
// 它进入签名消息的 SigScope（auth_flag 字节），并决定 CoveredInputs/CoveredOutputs
// 的覆盖子集。未被覆盖的部分完全不进入签名消息，可被第三方修改而不影响验证。
type AuthFlag uint8

const (
	// SigInAll 表示全部输入项（含解锁脚本）参与签名，独项（bit7）。
	SigInAll AuthFlag = 1 << 7
	// SigInSelf 表示仅当前输入项（含解锁脚本）参与签名，独项（bit6）。
	SigInSelf AuthFlag = 1 << 6
	// SigOutAll 表示全部输出项参与签名，主项（bit5）；须与辅项配合。
	SigOutAll AuthFlag = 1 << 5
	// SigOutSelf 表示与当前输入项同序位的输出项参与签名，主项（bit4）；须与辅项配合。
	SigOutSelf AuthFlag = 1 << 4
	// AuxScript 表示输出项锁定脚本，辅项（bit3）；须与主项配合。
	AuxScript AuthFlag = 1 << 3
	// AuxContent 表示输出项内容（除锁定脚本与接收者外的字段），辅项（bit2）。
	AuxContent AuthFlag = 1 << 2
	// AuxReceiver 表示输出项接收者字段，辅项（bit1）。
	AuxReceiver AuthFlag = 1 << 1
	// AuxOutput 表示输出项全部（脚本+内容+接收者），辅项（bit0）；与上面三辅项互斥。
	AuxOutput AuthFlag = 1 << 0
)

// maskAux 是除 OUTPUT 外的三个辅项位集合（SCRIPT|CONTENT|RECEIVER）。
const maskAux = AuxScript | AuxContent | AuxReceiver

// Byte 返回授权标记的原始字节值（即进入 SigScope 的 auth_flag 字节）。
func (f AuthFlag) Byte() byte { return byte(f) }

// Has 报告 f 是否设置了 x 中的任意位。
func (f AuthFlag) Has(x AuthFlag) bool { return f&x != 0 }

// hasMain 报告是否设置了主项（SIGOUT_ALL 或 SIGOUT_SELF）。
func (f AuthFlag) hasMain() bool { return f&(SigOutAll|SigOutSelf) != 0 }

// hasAux 报告是否设置了任意辅项（SCRIPT/CONTENT/RECEIVER/OUTPUT）。
func (f AuthFlag) hasAux() bool { return f&(maskAux|AuxOutput) != 0 }

// Validate 对授权标记做静态合法性检查（DEC-0102 §4）：
//   - OUTPUT 与 SCRIPT/CONTENT/RECEIVER 任一并存即互斥违例；
//   - 主项存在但无辅项配合 / 辅项存在但无主项配合，均拒绝。
//
// 独项（SIGIN_ALL/SIGIN_SELF）逻辑自完整，可独立或合并设置，不参与上述约束。
// 该规范化保证签名消息字节序列唯一对应一组 auth_flag。
func (f AuthFlag) Validate() error {
	if f.Has(AuxOutput) && f&maskAux != 0 {
		return ErrOutputAuxConflict
	}
	if f.hasMain() && !f.hasAux() {
		return ErrMainWithoutAux
	}
	if f.hasAux() && !f.hasMain() {
		return ErrAuxWithoutMain
	}
	return nil
}
