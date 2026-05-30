package hashtree

// 含序叶子辅助：序号作为 leaf payload 的内部前缀（大端），进入 leaf 哈希。
// 区块交易树用 3 字节序号（DEC-0004 §3.1），附件片组等用 2 字节序号。

// OrderedLeaf2 prepends a 2-byte big-endian sequence number to body, forming a
// leaf payload whose sequence is part of the leaf hash preimage.
func OrderedLeaf2(seq uint16, body []byte) []byte {
	out := make([]byte, 0, 2+len(body))
	out = append(out, byte(seq>>8), byte(seq))
	return append(out, body...)
}

// OrderedLeaf3 prepends a 3-byte big-endian sequence number (low 24 bits of seq)
// to body, as used by the block transaction tree (even a single-tx block uses
// sequence 000).
func OrderedLeaf3(seq uint32, body []byte) []byte {
	out := make([]byte, 0, 3+len(body))
	out = append(out, byte(seq>>16), byte(seq>>8), byte(seq))
	return append(out, body...)
}
