package hashtree

// 含序叶子辅助：序号作为 leaf payload 的内部前缀（大端），进入 leaf 哈希。
// 区块交易树用 3 字节序号（DEC-0004 §3.1），附件片组等用 2 字节序号。

// OrderedLeaf2 在 body 前追加 2 字节大端序号，形成叶子 payload，
// 该序号将成为叶哈希原像的一部分。
func OrderedLeaf2(seq uint16, body []byte) []byte {
	out := make([]byte, 0, 2+len(body))
	out = append(out, byte(seq>>8), byte(seq))
	return append(out, body...)
}

// OrderedLeaf3 在 body 前追加 3 字节大端序号（seq 的低 24 位），
// 用于区块交易树（即使单交易区块也使用序号 000）。
func OrderedLeaf3(seq uint32, body []byte) []byte {
	out := make([]byte, 0, 3+len(body))
	out = append(out, byte(seq>>16), byte(seq>>8), byte(seq))
	return append(out, body...)
}
