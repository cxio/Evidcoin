package script

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base32"
	"encoding/base64"
	"fmt"
	"hash"
	"io"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/mr-tron/base58"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/blake2s"
	"golang.org/x/crypto/sha3"
	"lukechampine.com/blake3"
)

// instr_function.go 实现函数段指令 [170-224] 的执行函数。
// 参考：docs/proposal/Instruction/16.Function-Instructions.md。

func init() {
	registerExec(FN_BASE58, execFN_BASE58)
	registerExec(FN_BASE32, execFN_BASE32)
	registerExec(FN_BASE64, execFN_BASE64)
	registerExec(FN_ADDRESS, execFN_ADDRESS)
	registerExec(FN_PUBHASH, execFN_PUBHASH)
	registerExec(FN_MPUBHASH, execFN_MPUBHASH)
	registerExec(FN_CHECKSIG, execFN_CHECKSIG)
	registerExec(FN_MCHECKSIG, execFN_MCHECKSIG)
	registerExec(FN_HASH224, execFN_HASH224)
	registerExec(FN_HASH256, execFN_HASH256)
	registerExec(FN_HASH384, execFN_HASH384)
	registerExec(FN_HASH512, execFN_HASH512)
	// opcode 182-222 保留，不注册
	registerExec(FN_PRINTF, execFN_PRINTF)
	registerExec(FN_X, execFN_X)
}

// execFN_BASE58 Base58 编解码（FN_BASE58，opcode 170）。
// 实参=Bytes → 编码为 String；实参=String → 解码为 Bytes。
func execFN_BASE58(vm *VM, _ *InstrFrame) error {
	v, err := vm.getOneArg()
	if err != nil {
		return err
	}
	switch v.Typ() {
	case TypeBytes:
		b, _ := v.AsBytes()
		encoded := base58.Encode(b)
		return vm.stack.Push(StringValue(encoded))
	case TypeString:
		s, _ := v.AsString()
		decoded, err := base58.Decode(s)
		if err != nil {
			return fmt.Errorf("%w: base58 decode: %v", ErrTypeMismatch, err)
		}
		return vm.stack.Push(BytesValue(decoded))
	default:
		return fmt.Errorf("%w: FN_BASE58 expects Bytes or String, got %s", ErrTypeMismatch, v.Typ())
	}
}

// base32NoPad 是无填充的 RFC4648 大写 Base32 编码器。
var base32NoPad = base32.StdEncoding.WithPadding(base32.NoPadding)

// execFN_BASE32 Base32 编解码（FN_BASE32，opcode 171）。
// 使用 RFC4648 大写无填充编码。
func execFN_BASE32(vm *VM, _ *InstrFrame) error {
	v, err := vm.getOneArg()
	if err != nil {
		return err
	}
	switch v.Typ() {
	case TypeBytes:
		b, _ := v.AsBytes()
		encoded := base32NoPad.EncodeToString(b)
		return vm.stack.Push(StringValue(encoded))
	case TypeString:
		s, _ := v.AsString()
		decoded, err := base32NoPad.DecodeString(s)
		if err != nil {
			return fmt.Errorf("%w: base32 decode: %v", ErrTypeMismatch, err)
		}
		return vm.stack.Push(BytesValue(decoded))
	default:
		return fmt.Errorf("%w: FN_BASE32 expects Bytes or String, got %s", ErrTypeMismatch, v.Typ())
	}
}

// execFN_BASE64 Base64 编解码（FN_BASE64，opcode 172）。
// 使用 URL 友好无填充编码（base64.RawURLEncoding）。
func execFN_BASE64(vm *VM, _ *InstrFrame) error {
	v, err := vm.getOneArg()
	if err != nil {
		return err
	}
	switch v.Typ() {
	case TypeBytes:
		b, _ := v.AsBytes()
		encoded := base64.RawURLEncoding.EncodeToString(b)
		return vm.stack.Push(StringValue(encoded))
	case TypeString:
		s, _ := v.AsString()
		decoded, err := base64.RawURLEncoding.DecodeString(s)
		if err != nil {
			return fmt.Errorf("%w: base64 decode: %v", ErrTypeMismatch, err)
		}
		return vm.stack.Push(BytesValue(decoded))
	default:
		return fmt.Errorf("%w: FN_BASE64 expects Bytes or String, got %s", ErrTypeMismatch, v.Typ())
	}
}

// netFromAttr 将附参网络标识转换为 crypto.Network。
// 0=Mainnet，1=Testnet，2=Devnet。
func netFromAttr(v uint64) crypto.Network {
	switch v {
	case 1:
		return crypto.Testnet
	case 2:
		return crypto.Devnet
	default:
		return crypto.Mainnet
	}
}

// execFN_ADDRESS 公钥哈希↔账户地址编解码（FN_ADDRESS，opcode 173）。
// Bytes(32 字节) → 编码为地址 String；String → 解码为 Bytes(公钥哈希)。
func execFN_ADDRESS(vm *VM, f *InstrFrame) error {
	v, err := vm.getOneArg()
	if err != nil {
		return err
	}
	switch v.Typ() {
	case TypeBytes:
		// 编码：附参=网络标识
		var netID uint64
		if len(f.AttrParams) > 0 {
			netID = readULEB128Param(f.AttrParams[0])
		}
		b, _ := v.AsBytes()
		var h [32]byte
		copy(h[:], b)
		addr, err := crypto.EncodeAddress(netFromAttr(netID), h)
		if err != nil {
			return fmt.Errorf("%w: FN_ADDRESS encode: %v", ErrTypeMismatch, err)
		}
		return vm.stack.Push(StringValue(addr))
	case TypeString:
		// 解码：忽略附参
		s, _ := v.AsString()
		_, h, err := crypto.DecodeAddress(s)
		if err != nil {
			return fmt.Errorf("%w: FN_ADDRESS decode: %v", ErrTypeMismatch, err)
		}
		return vm.stack.Push(BytesValue(h[:]))
	default:
		return fmt.Errorf("%w: FN_ADDRESS expects Bytes or String, got %s", ErrTypeMismatch, v.Typ())
	}
}

// execFN_PUBHASH 从公钥创建单签公钥哈希（FN_PUBHASH，opcode 174）。
// 实参=Bytes(pubKey) → Bytes(32 字节公钥哈希)。
func execFN_PUBHASH(vm *VM, _ *InstrFrame) error {
	v, err := vm.getOneArg()
	if err != nil {
		return err
	}
	b, err := v.AsBytes()
	if err != nil {
		return err
	}
	h := crypto.AddressHashSingle(b)
	return vm.stack.Push(BytesValue(h[:]))
}

// execFN_MPUBHASH 创建多签复合公钥哈希（FN_MPUBHASH，opcode 175）。
// 实参 1=Slice([baseHash,...])，实参 2=byte(m)，实参 3=byte(n) → Bytes(32 字节)。
func execFN_MPUBHASH(vm *VM, _ *InstrFrame) error {
	args, err := vm.getArgs(3)
	if err != nil {
		return err
	}
	baseSlice, err := args[0].AsSlice()
	if err != nil {
		return err
	}
	mByte, err := args[1].AsByte()
	if err != nil {
		return err
	}
	nByte, err := args[2].AsByte()
	if err != nil {
		return err
	}
	baseHashes := make([][]byte, len(baseSlice))
	for i, sv := range baseSlice {
		bh, err := sv.AsBytes()
		if err != nil {
			return err
		}
		baseHashes[i] = bh
	}
	h, err := crypto.AddressHashMultiFromBase(mByte, nByte, baseHashes)
	if err != nil {
		return fmt.Errorf("%w: FN_MPUBHASH: %v", ErrTypeMismatch, err)
	}
	return vm.stack.Push(BytesValue(h[:]))
}

// execFN_CHECKSIG 单签验证（FN_CHECKSIG，opcode 176）。
// 实参 1=chkType(Byte)，2=authFlag(Byte)，3=sig(Bytes)，4=pubKey(Bytes)。
func execFN_CHECKSIG(vm *VM, _ *InstrFrame) error {
	args, err := vm.getArgs(4)
	if err != nil {
		return err
	}
	chkType, err := args[0].AsByte()
	if err != nil {
		return err
	}
	authFlag, err := args[1].AsByte()
	if err != nil {
		return err
	}
	sig, err := args[2].AsBytes()
	if err != nil {
		return err
	}
	pubKey, err := args[3].AsBytes()
	if err != nil {
		return err
	}
	if vm.sigChecker == nil {
		return fmt.Errorf("%w: FN_CHECKSIG: no SignatureChecker injected", ErrTypeMismatch)
	}
	// message 从环境取（占位：使用空消息）
	ok, err := vm.sigChecker.CheckSig(chkType, authFlag, nil, sig, pubKey)
	if err != nil {
		return err
	}
	if ok {
		vm.SetSigned(0)
	}
	return vm.stack.Push(BoolValue(ok))
}

// execFN_MCHECKSIG 多签验证（FN_MCHECKSIG，opcode 177）。
// 实参 1=chkType(Byte)，2=authFlag(Byte)，3=[]sigs(Slice)，4=[]pubKeys(Slice)，5=[]baseHashes(Slice)。
func execFN_MCHECKSIG(vm *VM, _ *InstrFrame) error {
	args, err := vm.getArgs(5)
	if err != nil {
		return err
	}
	chkType, err := args[0].AsByte()
	if err != nil {
		return err
	}
	authFlag, err := args[1].AsByte()
	if err != nil {
		return err
	}
	if vm.sigChecker == nil {
		return fmt.Errorf("%w: FN_MCHECKSIG: no SignatureChecker injected", ErrTypeMismatch)
	}
	// 提取签名、公钥、基础哈希切片
	sigsSlice, err := args[2].AsSlice()
	if err != nil {
		return err
	}
	pubKeysSlice, err := args[3].AsSlice()
	if err != nil {
		return err
	}
	baseHashSlice, err := args[4].AsSlice()
	if err != nil {
		return err
	}
	toByteSlice := func(vs []Value) ([][]byte, error) {
		out := make([][]byte, len(vs))
		for i, v := range vs {
			b, err := v.AsBytes()
			if err != nil {
				return nil, err
			}
			out[i] = b
		}
		return out, nil
	}
	sigs, err := toByteSlice(sigsSlice)
	if err != nil {
		return err
	}
	pubKeys, err := toByteSlice(pubKeysSlice)
	if err != nil {
		return err
	}
	baseHashes, err := toByteSlice(baseHashSlice)
	if err != nil {
		return err
	}
	ok, err := vm.sigChecker.CheckMultiSig(chkType, authFlag, nil, sigs, pubKeys, baseHashes)
	if err != nil {
		return err
	}
	if ok {
		vm.SetSigned(0)
	}
	return vm.stack.Push(BoolValue(ok))
}

// hashAlgo 哈希算法标识常量（附参值）。
const (
	hashAlgoBLAKE3 = 0 // BLAKE3
	hashAlgoBLAKE2 = 1 // BLAKE2b/2s
	hashAlgoSHA2   = 2 // SHA2（SHA-2 系列）
	hashAlgoSHA3   = 3 // SHA3（SHA-3 系列）
)

// computeHash 按指定位数和算法标识对 data 执行哈希，返回 N/8 字节。
func computeHash(bits int, algo uint64, data []byte) ([]byte, error) {
	size := bits / 8
	var h hash.Hash
	switch algo {
	case hashAlgoBLAKE3:
		// blake3 支持任意输出长度
		xof := blake3.New(size, nil)
		xof.Write(data)
		return xof.Sum(nil), nil
	case hashAlgoBLAKE2:
		switch bits {
		case 224:
			// blake2b 支持 1-64 字节，28 字节可行
			hh, err := blake2b.New(28, nil)
			if err != nil {
				return nil, fmt.Errorf("script: blake2b-224: %w", err)
			}
			h = hh
		case 256:
			hh, err := blake2b.New256(nil)
			if err != nil {
				return nil, fmt.Errorf("script: blake2b-256: %w", err)
			}
			h = hh
		case 384:
			hh, err := blake2b.New384(nil)
			if err != nil {
				return nil, fmt.Errorf("script: blake2b-384: %w", err)
			}
			h = hh
		case 512:
			hh, err := blake2b.New512(nil)
			if err != nil {
				return nil, fmt.Errorf("script: blake2b-512: %w", err)
			}
			h = hh
		default:
			// 其余位数用 blake2s（最大 256 位）或退回 BLAKE3
			if bits <= 256 {
				hh, err := blake2s.New256(nil)
				if err != nil {
					return nil, fmt.Errorf("script: blake2s-256: %w", err)
				}
				h = hh
			} else {
				xof := blake3.New(size, nil)
				xof.Write(data)
				return xof.Sum(nil), nil
			}
		}
	case hashAlgoSHA2:
		switch bits {
		case 224:
			h = sha256.New224()
		case 256:
			h = sha256.New()
		case 384:
			h = sha512.New384()
		case 512:
			h = sha512.New()
		default:
			return nil, fmt.Errorf("script: SHA2 unsupported bits: %d", bits)
		}
	case hashAlgoSHA3:
		switch bits {
		case 224:
			h = sha3.New224()
		case 256:
			h = sha3.New256()
		case 384:
			h = sha3.New384()
		case 512:
			h = sha3.New512()
		default:
			return nil, fmt.Errorf("script: SHA3 unsupported bits: %d", bits)
		}
	default:
		return nil, fmt.Errorf("script: unknown hash algorithm: %d", algo)
	}
	io.Writer(h).Write(data)
	return h.Sum(nil), nil
}

// execHashN 通用哈希指令执行器（N 位）。
func execHashN(vm *VM, f *InstrFrame, bits int) error {
	var algo uint64
	if len(f.AttrParams) > 0 {
		algo = readULEB128Param(f.AttrParams[0])
	}
	v, err := vm.getOneArg()
	if err != nil {
		return err
	}
	data, err := v.AsBytes()
	if err != nil {
		return err
	}
	result, err := computeHash(bits, algo, data)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrTypeMismatch, err)
	}
	return vm.stack.Push(BytesValue(result))
}

// execFN_HASH224 224 位哈希（FN_HASH224，opcode 178）。
func execFN_HASH224(vm *VM, f *InstrFrame) error { return execHashN(vm, f, 224) }

// execFN_HASH256 256 位哈希（FN_HASH256，opcode 179）。
func execFN_HASH256(vm *VM, f *InstrFrame) error { return execHashN(vm, f, 256) }

// execFN_HASH384 384 位哈希（FN_HASH384，opcode 180）。
func execFN_HASH384(vm *VM, f *InstrFrame) error { return execHashN(vm, f, 384) }

// execFN_HASH512 512 位哈希（FN_HASH512，opcode 181）。
func execFN_HASH512(vm *VM, f *InstrFrame) error { return execHashN(vm, f, 512) }

// execFN_PRINTF 格式化打印（FN_PRINTF，opcode 223）。
// 实参 1=format String，其余为参数；压入格式化结果 String。
func execFN_PRINTF(vm *VM, _ *InstrFrame) error {
	// 从实参区读取所有参数（不定数量）
	var fmtStr string
	var fmtArgs []interface{}

	if vm.args.Len() > 0 {
		// 实参区有值：第一个为格式串
		fv, err := vm.args.Dequeue()
		if err != nil {
			return err
		}
		s, err := fv.AsString()
		if err != nil {
			return err
		}
		fmtStr = s
		for vm.args.Len() > 0 {
			av, _ := vm.args.Dequeue()
			fmtArgs = append(fmtArgs, valueToInterface(av))
		}
	} else {
		// 从数据栈弹出格式串（栈顶）
		fv, err := vm.stack.Pop()
		if err != nil {
			return err
		}
		s, err := fv.AsString()
		if err != nil {
			return err
		}
		fmtStr = s
	}

	result := fmt.Sprintf(fmtStr, fmtArgs...)
	return vm.stack.Push(StringValue(result))
}

// valueToInterface 将 Value 转为 interface{}（用于 fmt.Sprintf 参数）。
func valueToInterface(v Value) interface{} {
	switch v.Typ() {
	case TypeNil:
		return nil
	case TypeBool:
		b, _ := v.AsBool()
		return b
	case TypeByte:
		b, _ := v.AsByte()
		return b
	case TypeInt:
		n, _ := v.AsInt()
		return n
	case TypeFloat:
		f, _ := v.AsFloat()
		return f
	case TypeString:
		s, _ := v.AsString()
		return s
	case TypeBytes:
		b, _ := v.AsBytes()
		return b
	default:
		return fmt.Sprintf("<%s>", v.Typ())
	}
}

// execFN_X 标准函数引用占位（FN_X，opcode 224）。
// 标准函数集尚未定义，返回 ScriptError。
func execFN_X(vm *VM, _ *InstrFrame) error {
	return fmt.Errorf("%w: FN_X: standard function set not yet defined", ErrTypeMismatch)
}
