package tx

// 介管脚本（Mediator）属存证信元（类型 3，第 07 章 §6）。介管脚本是脚本跳转
// （GOTO/EMBED）的目标输出脚本，可插入监管逻辑（如财务监听）。它不能作为输入项、
// 不进入 UTXO/UTCO，仅以存证类输出形式参与区块哈希。

// NewMediatorOutput 构造一个承载介管脚本的存证类输出。
// 介管脚本本身置于输出公共头的 LockScript；存证类无信元载荷字段（Payload 为空）。
// 返回的输出 InState() 恒为 false（不可作为后续输入源）。
func NewMediatorOutput(script []byte) Output {
	return Output{
		Type:       TypeProof,
		LockScript: script,
	}
}
