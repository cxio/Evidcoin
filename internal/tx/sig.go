package tx

// UnlockData 输入项解锁数据（不参与 TxID 计算）。
type UnlockData struct {
	// InputIndex 对应的输入序位
	InputIndex int
	// Flag 签名授权标志
	Flag SigFlag
	// Kind 解锁类型（单签/多签）
	Kind UnlockKind
	// Single 单签名解锁数据（Kind == UnlockSingle 时有效）
	Single *SingleSigUnlock
	// Multi 多签名解锁数据（Kind == UnlockMulti 时有效）
	Multi *MultiSigUnlock
}

// UnlockKind 解锁类型。
type UnlockKind byte

const (
	// UnlockSingle 单签名解锁
	UnlockSingle UnlockKind = 1
	// UnlockMulti 多重签名解锁
	UnlockMulti UnlockKind = 2
)

// SingleSigUnlock 单签名解锁数据。
type SingleSigUnlock struct {
	// Signature ML-DSA-65 签名字节
	Signature []byte
	// PublicKey 对应的公钥字节
	PublicKey []byte
}

// MultiSigUnlock 多重签名解锁数据。
type MultiSigUnlock struct {
	// Signatures m 个签名字节列表
	Signatures [][]byte
	// PublicKeys m 个对应公钥字节列表
	PublicKeys [][]byte
	// Complement N-m 个未参与签名的公钥哈希列表
	Complement [][32]byte
}

// M 返回多签所需最小签名数（由 Signatures 数量推导）。
func (m *MultiSigUnlock) M() int {
	return len(m.Signatures)
}

// N 返回多签总参与者数（M + len(Complement)）。
func (m *MultiSigUnlock) N() int {
	return m.M() + len(m.Complement)
}
