package tx

// 自定义类（Custom）输出（Config bit7=1，第 06 章 §6 / 第 07 章 §6）：私有应用借公共
// 网络传递信息的机制，携带最长 127 字节私有标识 ID 供专属客户端识别。自定义类输出不
// 进入 UTXO/UTCO，不能作为后续输入源；节点仅做编码合法性校验，不做语义校验。

// NewCustomOutput 构造一个自定义类输出。id 为私有标识（≤127 字节），payload 为应用
// 自定义载荷字节，lockScript 为可选锁定脚本。当 id 超过 127 字节时返回 ErrCustomIDTooLong。
// 返回的输出 InState() 恒为 false（不入状态集、不可作输入源）。
func NewCustomOutput(id, payload, lockScript []byte) (Output, error) {
	if len(id) > maxCustomIDLen {
		return Output{}, ErrCustomIDTooLong
	}
	return Output{
		IsCustom:   true,
		CustomID:   id,
		Payload:    payload,
		LockScript: lockScript,
	}, nil
}
