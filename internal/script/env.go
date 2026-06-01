package script

// env.go 定义脚本 VM 运行时的扩展注入接口。
// 参考：DEC-0503，docs/proposal/10.Script-System.md §4。

// SignatureChecker 签名验证注入接口，供 FN_CHECKSIG/FN_MCHECKSIG 使用。
// message、signature、publicKey 均为原始字节。
type SignatureChecker interface {
	// CheckSig 验证单签。chkType=算法类型，authFlag=授权标志。
	CheckSig(chkType byte, authFlag byte, message, signature, publicKey []byte) (bool, error)
	// CheckMultiSig 验证多签。
	CheckMultiSig(chkType byte, authFlag byte, message [][]byte, sigs [][]byte, pubKeys [][]byte, baseHashes [][]byte) (bool, error)
}

// WitnessProvider 见证数据注入接口，供 SYS_CHKPASS 使用。
// 返回 nil 表示无见证数据（ErrWitnessMissing）。
type WitnessProvider interface {
	// GetWitness 返回见证数据字节；nil 表示无见证。
	GetWitness() []byte
	// IsCoinbase 返回当前是否为 Coinbase 交易。
	IsCoinbase() bool
}

// 环境命名空间常量（DEC-0503）。
const (
	EnvNsSystem = "sys"   // 系统域
	EnvNsTx     = "tx"    // 交易域
	EnvNsCheck  = "check" // 校验域
)
