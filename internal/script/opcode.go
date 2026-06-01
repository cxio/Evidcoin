// Package script 实现 Evidcoin 栈式脚本 VM。
// 依赖 pkg/types 和 pkg/crypto（Layer 0），不依赖 internal/blockchain、
// internal/tx、internal/utxo、internal/utco、internal/consensus 等。
package script

// Opcode 是 1 字节的指令码，标识脚本指令集中的每条指令。
// 指令集分四段：基础段 [0-169]、函数段 [170-224]、模块段 [225-250]、扩展段 [251-253]。
// opcode 254-255 为系统保留，不分配任何指令。
// 参考：docs/proposal/10.Script-System.md §4
type Opcode byte

// ─── 值指令 [0-18] ───────────────────────────────────────────────────────────
// 参考：docs/proposal/Instruction/1.Value-Instructions.md

const (
	// NIL 表达通用 nil 值，压入数据栈。
	NIL Opcode = 0
	// TRUE 表达布尔 true。
	TRUE Opcode = 1
	// FALSE 表达布尔 false。
	FALSE Opcode = 2
	// BYTE_LIT 表达单字节字面量 [0,255]（码点 <=0xff 的单引号字面量）。
	BYTE_LIT Opcode = 3
	// RUNE_LIT 表达 Unicode 码点字面量（码点 >0xff 的单引号字面量）。
	RUNE_LIT Opcode = 4
	// INT_LIT 表达 int64 字面量，后跟 ULEB128 变长编码。
	INT_LIT Opcode = 5
	// BIGINT_LIT 表达大整数字面量，附参 = slen||magnitude（DEC-0001）。
	BIGINT_LIT Opcode = 6
	// FLOAT_LIT 表达 float64 字面量，8 字节大端 IEEE 754 bit pattern（DEC-0502）。
	FLOAT_LIT Opcode = 7
	// STRING_LIT 表达字符串字面量，附参变长整数 = UTF-8 字节数。
	STRING_LIT Opcode = 8
	// VALUES_LIT 表达同类型值集合（切片），附参 = 成员序列总字节长。
	VALUES_LIT Opcode = 9
	// DATA_LIT 表达字节序列字面量，附参变长整数 = 字节数。
	DATA_LIT Opcode = 10
	// REGEXP_LIT 表达正则表达式字面量，附参 1 字节 = 内容字节数（<256）。
	REGEXP_LIT Opcode = 11
	// DATE_LIT 表达时间字面量，UNIX 毫秒有符号变长整数。
	DATE_LIT Opcode = 12
	// DICT_LIT 表达有序字典，实参 1=键序列，实参 2=值序列。
	DICT_LIT Opcode = 13
	// opcode 14-15 保留未分配。
	_opcode14 Opcode = 14
	_opcode15 Opcode = 15
	// CODE_LIT 表达已编译指令序列字面值，不执行，附参 = 序列长度。
	CODE_LIT Opcode = 16
	// SCRIPT 引入历史交易输出脚本；前期禁用（DEC-0505）。
	SCRIPT Opcode = 17
	// VALUE 通用取值；前期禁用（DEC-0505）。
	VALUE Opcode = 18
)

// ─── 截取指令 [19-23] ────────────────────────────────────────────────────────
// 参考：docs/proposal/Instruction/2.Capture-Instructions.md

const (
	// AT（@）拦截后一指令返回值放入实参区。
	AT Opcode = 19
	// SAVE（$）截取后一指令返回值存入当前局部域。
	SAVE Opcode = 20
	// LOCAL（$[n]）引用局部域成员放入实参区，附参 = 成员位置下标 [-128,127]。
	LOCAL Opcode = 21
	// LOOPVAR（$X）循环变量引用，附参 = 变量标识 [0-3]。
	LOOPVAR Opcode = 22
	// DIRECT（~）指示后一指令实参从数据栈直接提取（跳过实参区）。
	DIRECT Opcode = 23
)

// ─── 栈操作指令 [24-34] ──────────────────────────────────────────────────────
// 参考：docs/proposal/Instruction/3.Stack-Operations.md

const (
	// NOP 无操作，读取（清空）实参区。
	NOP Opcode = 24
	// PUSH 将实参区成员顺序压入数据栈。
	PUSH Opcode = 25
	// POP 弹出栈顶项。
	POP Opcode = 26
	// POPS 弹出栈顶多项，附参 = 条目数（0=全部）。
	POPS Opcode = 27
	// TOP 引用（复制）栈顶项。
	TOP Opcode = 28
	// TOPS 引用栈顶多项，附参 = 条目数（0=全部）。
	TOPS Opcode = 29
	// PEEK 引用栈内任意位置，实参 = 位置（0=栈底，负数从栈顶）。
	PEEK Opcode = 30
	// PEEKS 引用栈内任意位置段，附参 = 条目数，实参 = 起始位置。
	PEEKS Opcode = 31
	// SHIFT 移出栈顶条目打包为切片，附参 = 条目数（0=全部）。
	SHIFT Opcode = 32
	// CLONE 克隆栈顶条目（浅复制）打包为切片，附参 = 条目数（0=全部）。
	CLONE Opcode = 33
	// VIEW 引用栈条目打包为切片，附参 = 条目数，实参 = 起始位置。
	VIEW Opcode = 34
)

// ─── 集合指令 [35-45] ────────────────────────────────────────────────────────
// 参考：docs/proposal/Instruction/4.Collection-Operations.md

const (
	// SLICE 截取子切片，附参 = 子切片大小（0=之后全部），实参 1=目标切片，2=起始下标。
	SLICE Opcode = 35
	// REVERSE 切片成员反转。
	REVERSE Opcode = 36
	// MERGE 多切片合并。
	MERGE Opcode = 37
	// EXTEND 向目标切片添加成员。
	EXTEND Opcode = 38
	// PACK 切片成员打包为字节序列。
	PACK Opcode = 39
	// SPREAD 展开切片成员到目标空间。
	SPREAD Opcode = 40
	// INDEX 取切片成员，实参 1=切片，2=下标或下标集。
	INDEX Opcode = 41
	// ITEM 取字典/对象/模块静态成员，实参 1=集合，2=键或键集。
	ITEM Opcode = 42
	// SET 设置字典键值，实参 1=目标字典，2=键或键集，3=值或值集。
	SET Opcode = 43
	// CALL 调用对象/模块方法，附参 = 传给方法的实参数量（半确定性）。
	CALL Opcode = 44
	// SIZE 返回集合大小。
	SIZE Opcode = 45
)

// ─── 交互指令 [46-50] ────────────────────────────────────────────────────────
// 参考：docs/proposal/Instruction/5.Interaction-Instructions.md

const (
	// INPUT 从 INPUT 缓存区导入数据；公共验证节点视为隐式 END（DEC-0503/0505）。
	INPUT Opcode = 46
	// OUTPUT 把实参区数据导出到 OUTPUT 缓存区（非阻塞）。
	OUTPUT Opcode = 47
	// BUFDUMP 转出 OUTPUT 缓存区到外部环境，触发外部监听，附参 = 标识值。
	BUFDUMP Opcode = 48
	// opcode 49 保留未分配。
	_opcode49 Opcode = 49
	// PRINT 打印实参区内容到控制台。
	PRINT Opcode = 50
)

// ─── 结果指令 [51-57] ────────────────────────────────────────────────────────
// 参考：docs/proposal/Instruction/6.Result-Instructions.md

const (
	// PASS 写入通关状态；true 继续，false 立即 VerifyFail（DEC-0505）。
	PASS Opcode = 51
	// CHECK 写入通关状态但不退出；后写覆盖前值（DEC-0505）。
	CHECK Opcode = 52
	// GOTO 跳转到目标输出脚本，附参 = 年度+TxID(48B)+输出序位。
	GOTO Opcode = 53
	// EMBED 嵌入目标输出脚本（共享当前域），附参同 GOTO。
	EMBED Opcode = 54
	// EXIT 终止脚本退出，检查通关状态，实参可选返回值。
	EXIT Opcode = 55
	// RETURN 从函数子块退出并返回值（仅用于 MAP/FILTER）。
	RETURN Opcode = 56
	// END 公共验证逻辑结束（DEC-0505）；以当前通关状态 PassStop。
	END Opcode = 57
)

// ─── 流程控制 [58-66] ────────────────────────────────────────────────────────
// 参考：docs/proposal/Instruction/7.Flow-Control.md

const (
	// IF 条件分支，附参 = 子块长度。
	IF Opcode = 58
	// ELSE 否定分支，附参 = 子块长度。
	ELSE Opcode = 59
	// SWITCH 多路选择，附参 = 块长。
	SWITCH Opcode = 60
	// CASE 匹配分支，附参 = 子块长。
	CASE Opcode = 61
	// DEFAULT 默认分支，附参 = 子块长。
	DEFAULT Opcode = 62
	// EACH 有限迭代，附参 = 子块长。
	EACH Opcode = 63
	// CONTINUE 终止当前 EACH 迭代，实参可选布尔值。
	CONTINUE Opcode = 64
	// BREAK 退出 EACH/SWITCH 块，实参可选布尔值。
	BREAK Opcode = 65
	// BLOCK 创建子块（局部域范围），附参 = 子块长。
	BLOCK Opcode = 66
)

// ─── 转换指令 [67-79] ────────────────────────────────────────────────────────
// 参考：docs/proposal/Instruction/8.Conversion-Instructions.md

const (
	// BOOL 转换为布尔类型。
	BOOL Opcode = 67
	// BYTE_CONV 转换为字节类型。
	BYTE_CONV Opcode = 68
	// RUNE_CONV 转换为 Unicode 码点类型。
	RUNE_CONV Opcode = 69
	// INT_CONV 转换为 int64 类型。
	INT_CONV Opcode = 70
	// BIGINT_CONV 转换为大整数类型。
	BIGINT_CONV Opcode = 71
	// FLOAT_CONV 转换为 float64 类型（DEC-0502）。
	FLOAT_CONV Opcode = 72
	// STRING_CONV 转换为字符串，附参 = 进制或格式标识。
	STRING_CONV Opcode = 73
	// BYTES_CONV 转换为字节序列。
	BYTES_CONV Opcode = 74
	// RUNES_CONV 转换为符文序列。
	RUNES_CONV Opcode = 75
	// TIME_CONV 转换为时间类型。
	TIME_CONV Opcode = 76
	// REGEXP_CONV 从合法字符串构造正则（RE2）。
	REGEXP_CONV Opcode = 77
	// opcode 78 保留未分配。
	_opcode78 Opcode = 78
	// ANYS 切片类型转换，附参 = 转换标识。
	ANYS Opcode = 79
)

// ─── 运算指令 [80-103] ───────────────────────────────────────────────────────
// 参考：docs/proposal/Instruction/9.Arithmetic-Instructions.md

const (
	// EXPR 独立运算表达式块（内部统一 float64），附参 = 表达式长度（1 字节）。
	EXPR Opcode = 80
	// MUL_OP 表达式内乘号 *（仅表达式内）。
	MUL_OP Opcode = 81
	// DIV_OP 表达式内除号 /（仅表达式内）。
	DIV_OP Opcode = 82
	// MOD_OP 表达式内模号 %（仅表达式内）。
	MOD_OP Opcode = 83
	// ADD_OP 表达式内加号 +（仅表达式内）。
	ADD_OP Opcode = 84
	// SUB_OP 表达式内减号 -（仅表达式内）。
	SUB_OP Opcode = 85
	// MUL 乘法，双实参 Int/Float。
	MUL Opcode = 86
	// DIV 除法，双实参 Int/Float。
	DIV Opcode = 87
	// ADD 加法，双实参 Int/Float；也支持 String/Bytes 连接、Dict 合并。
	ADD Opcode = 88
	// SUB 减法，双实参 Int/Float。
	SUB Opcode = 89
	// MOD 取模，双实参 Int/Float（math.Mod）。
	MOD Opcode = 90
	// POW 幂运算，双实参 Int/Float（math.Pow）。
	POW Opcode = 91
	// LMOV 左移，双实参 Int。
	LMOV Opcode = 92
	// RMOV 右移，双实参 Int。
	RMOV Opcode = 93
	// AND 位与，双实参 Int。
	AND Opcode = 94
	// ANDX 位清空（&^），双实参 Int。
	ANDX Opcode = 95
	// OR 位或，双实参 Int。
	OR Opcode = 96
	// XOR 位异或，双实参 Int。
	XOR Opcode = 97
	// NEG 取反，单实参 Int/Float。
	NEG Opcode = 98
	// NOT 逻辑非，单实参 Bool。
	NOT Opcode = 99
	// DIVMOD 整除+模，双实参 Int/Float，返回两值（自动展开）。
	DIVMOD Opcode = 100
	// REP 重复条目，附参 = 份数；0 份相当于移除栈顶。
	REP Opcode = 101
	// DEL 删除字典条目，实参 1=字典，2=键名或键名序列。
	DEL Opcode = 102
	// CLEAR 清空字典，实参 = 目标字典。
	CLEAR Opcode = 103
)

// ─── 比较指令 [104-111] ──────────────────────────────────────────────────────
// 参考：docs/proposal/Instruction/10.Comparison-Instructions.md

const (
	// EQUAL 相等比较 a==b（DEC-0502: 浮点 NaN 返回 false；+0==-0）。
	EQUAL Opcode = 104
	// NEQUAL 不等比较 a!=b。
	NEQUAL Opcode = 105
	// LT 小于比较 a<b。
	LT Opcode = 106
	// LTE 小于等于比较 a<=b。
	LTE Opcode = 107
	// GT 大于比较 a>b。
	GT Opcode = 108
	// GTE 大于等于比较 a>=b。
	GTE Opcode = 109
	// ISEFV 判断异常浮点值，附参 = 标识（0=NaN,1=+Inf,2=-Inf）（DEC-0502）。
	ISEFV Opcode = 110
	// WITHIN 半开区间比较 min<=x<max。
	WITHIN Opcode = 111
)

// ─── 逻辑指令 [112-115] ──────────────────────────────────────────────────────
// 参考：docs/proposal/Instruction/11.Logic-Instructions.md

const (
	// BOTH 逻辑 AND，双布尔实参。
	BOTH Opcode = 112
	// EITHER 逻辑 OR，双布尔实参。
	EITHER Opcode = 113
	// EVERY 逻辑 AND 集合；空集返回 true。
	EVERY Opcode = 114
	// SOME 逻辑 OR 集合，附参 = 最低为真数量；空集且 n>0 返回 false。
	SOME Opcode = 115
)

// ─── 模式指令 [116-127] ──────────────────────────────────────────────────────
// 参考：docs/proposal/Instruction/12.Pattern-Instructions.md

const (
	// MODEL 开启模式匹配子环境，附参 = 子块代码长度（最高位为取值标记，DEC-0501）。
	MODEL Opcode = 116
	// WILDCARD 单指令通配 _（仅 MODEL 内）。
	WILDCARD Opcode = 117
	// WILDCARDS 指令段通配 _[n]（仅 MODEL 内），附参 = 指令个数。
	WILDCARDS Opcode = 118
	// OPTIONAL 序列可选 ?{}（仅 MODEL 内），附参 = 序列长度。
	OPTIONAL Opcode = 119
	// MATCHMOD 通配/可选指示 ^?（仅 MODEL 内），附参 = 位标识。
	MATCHMOD Opcode = 120
	// EXTRACT 取值指令 #（仅 MODEL 内），附参 = 目标值标识。
	EXTRACT Opcode = 121
	// TYPEMATCH 类型匹配 !?（仅 MODEL 内），附参 = 类型标识（含匹配/可选高位）。
	TYPEMATCH Opcode = 122
	// INTRANGE 整数范围匹配 >{}（仅 MODEL 内），附参 = 下界+上界（各变长）。
	INTRANGE Opcode = 123
	// FLORANGE 浮点范围匹配（仅 MODEL 内），附参 = 下界(8B)+上界(8B)+误差(4B)。
	FLORANGE Opcode = 124
	// REMATCH 正则匹配查找 RE{}（仅 MODEL 内），附参 = 指令名/查找方式/正则长度。
	REMATCH Opcode = 125
	// REGEXTRACT 正则取值 &（仅 MODEL 内），附参 = 取值序位。
	REGEXTRACT Opcode = 126
	// ELLIPSIS 任意连续指令段通配 ...（仅 MODEL 内，非贪婪）。
	ELLIPSIS Opcode = 127
)

// ─── 环境指令 [128-137] ──────────────────────────────────────────────────────
// 参考：docs/proposal/Instruction/13.Environment-Instructions.md

const (
	// ENV 取环境变量（系统/交易/校验域），附参 = 目标标识。
	ENV Opcode = 128
	// IN 取输入项数据（校验域），附参 = 目标标识。
	IN Opcode = 129
	// OUT 取输出项数据（交易域），附参 1=输出序位，2=目标标识。
	OUT Opcode = 130
	// INOUT 取输入项来源输出集兄弟条目；前期禁用（DEC-0505）。
	INOUT Opcode = 131
	// opcode 132 保留未分配。
	_opcode132 Opcode = 132
	// XFROM 获取源交易信息（仅 GOTO/EMBED 目标脚本中可用），附参 = 目标标识。
	XFROM Opcode = 133
	// SIGNED 签名序位核实，附参 = 目标序位。
	SIGNED Opcode = 134
	// VAR 全局变量取值，附参 = 变量位置 [0-255]。
	VAR Opcode = 135
	// SETVAR 全局变量赋值，附参 = 变量位置 [0-255]。
	SETVAR Opcode = 136
	// SOURCE 脚本源码提取，附参 = 标识值。
	SOURCE Opcode = 137
)

// ─── 工具指令 [138-163] ──────────────────────────────────────────────────────
// 参考：docs/proposal/Instruction/14.Tool-Instructions.md

const (
	// EVAL 执行脚本指令序列；前期禁用（DEC-0505）。
	EVAL Opcode = 138
	// COPY 浅复制切片。
	COPY Opcode = 139
	// DCOPY 深复制切片。
	DCOPY Opcode = 140
	// KEYVAL 字典键值切取，附参 = 切取类型（0=键值,1=键,2=值）。
	KEYVAL Opcode = 141
	// MATCH 正则匹配，附参 = 匹配方式。
	MATCH Opcode = 142
	// SUBSTR 截取 UTF-8 字符串，附参 = 字符数。
	SUBSTR Opcode = 143
	// REPLACE 字符串替换，附参 = 替换次数（0=全部）。
	REPLACE Opcode = 144
	// RANDOM 确定性随机数（ChaCha8），实参 1=种子(32B)，2=上限。
	RANDOM Opcode = 145
	// SLRAND 确定性切片乱序（ChaCha8），实参 1=种子(32B)，2=切片。
	SLRAND Opcode = 146
	// CMPFLO 浮点比较（带误差），附参 = 类型（0=相等,-1=<=,1=>=）。
	CMPFLO Opcode = 147
	// opcode 148-151 保留未分配。
	_opcode148 Opcode = 148
	_opcode149 Opcode = 149
	_opcode150 Opcode = 150
	_opcode151 Opcode = 151
	// RANGE 创建数值序列，附参 = 序列长度（1 字节）。
	RANGE Opcode = 152
	// MAP 映射迭代，附参 = 子块长。
	MAP Opcode = 153
	// FILTER 成员过滤，附参 = 子块长。
	FILTER Opcode = 154
	// SHELL 执行 Shell（公共节点忽略消费实参，非禁用），附参 = 内容长度。
	SHELL Opcode = 155
	// opcode 156-163 量子安全保留区，不分配。
	_opcode156 Opcode = 156
	_opcode157 Opcode = 157
	_opcode158 Opcode = 158
	_opcode159 Opcode = 159
	_opcode160 Opcode = 160
	_opcode161 Opcode = 161
	_opcode162 Opcode = 162
	_opcode163 Opcode = 163
)

// ─── 系统指令 [164-169] ──────────────────────────────────────────────────────
// 参考：docs/proposal/Instruction/15.System-Instructions.md

const (
	// SYS_TIME 取全局时间；禁用于公共验证路径，仅 END 后可用（DEC-0505）。
	SYS_TIME Opcode = 164
	// SYS_AWARD 兑奖验算；仅 Coinbase 交易输出脚本可用（DEC-0503）。
	SYS_AWARD Opcode = 165
	// SYS_CHKPASS 系统验证（地址核实+签名验证），也是通关指令（DEC-0505）。
	SYS_CHKPASS Opcode = 166
	// opcode 167-168 保留未分配。
	_opcode167 Opcode = 167
	_opcode168 Opcode = 168
	// SYS_NULL 源码零点标识；可突破解锁段限制（特例）。
	SYS_NULL Opcode = 169
)

// ─── 函数段 [170-224] ────────────────────────────────────────────────────────
// 参考：docs/proposal/Instruction/16.Function-Instructions.md

const (
	// FN_BASE58 Base58 编解码。
	FN_BASE58 Opcode = 170
	// FN_BASE32 Base32 编解码（RFC4648 无填充）。
	FN_BASE32 Opcode = 171
	// FN_BASE64 Base64 编解码（URL 友好无填充）。
	FN_BASE64 Opcode = 172
	// FN_ADDRESS 公钥哈希↔账户地址编解码（Base58）。
	FN_ADDRESS Opcode = 173
	// FN_PUBHASH 从公钥创建单签公钥哈希。
	FN_PUBHASH Opcode = 174
	// FN_MPUBHASH 创建多签复合公钥哈希。
	FN_MPUBHASH Opcode = 175
	// FN_CHECKSIG 单签验证（定制验证路径）。
	FN_CHECKSIG Opcode = 176
	// FN_MCHECKSIG 多签验证（定制验证路径）。
	FN_MCHECKSIG Opcode = 177
	// FN_HASH224 224 位哈希，附参 = 算法标识（0=BLAKE3,1=BLAKE2,2=SHA2,3=SHA3）。
	FN_HASH224 Opcode = 178
	// FN_HASH256 256 位哈希，附参 = 算法标识。
	FN_HASH256 Opcode = 179
	// FN_HASH384 384 位哈希，附参 = 算法标识。
	FN_HASH384 Opcode = 180
	// FN_HASH512 512 位哈希，附参 = 算法标识。
	FN_HASH512 Opcode = 181
	// opcode 182-222 保留未分配（41 个）。
	_opcode182 Opcode = 182
	// FN_PRINTF 格式化打印（fmt.Printf）。
	FN_PRINTF Opcode = 223
	// FN_X 标准函数引用，附参 = 目标索引。
	FN_X Opcode = 224
)

// ─── 模块段 [225-250] ────────────────────────────────────────────────────────
// 参考：docs/proposal/Instruction/17.Module-Instructions.md

const (
	// MO_MATH 数学运算模块对象。
	MO_MATH Opcode = 225
	// MO_FMT 数据格式化模块（Go fmt 局部封装）。
	MO_FMT Opcode = 226
	// opcode 227-249 保留未分配（23 个）。
	_opcode227 Opcode = 227
	// MO_XX 标准模块引用，附参 = 目标索引（1 字节）。
	MO_XX Opcode = 250
)

// ─── 扩展段 [251-253] ────────────────────────────────────────────────────────
// 参考：docs/proposal/Instruction/18.Extension-Instructions.md

const (
	// EXT_MO 扩展模块包，附参 = 目标索引（2 字节），64K 空间。
	EXT_MO Opcode = 251
	// opcode 252 保留未分配。
	_opcode252 Opcode = 252
	// EXT_PRIV 私有扩展包；公共路径触达即 ScriptError（DEC-0505）。
	EXT_PRIV Opcode = 253
)

// ─── 系统保留 [254-255] ──────────────────────────────────────────────────────
// opcode 254/255 永久保留，不分配任何指令。

// 指令段范围常量
const (
	// BasicSegStart 基础段起始码位。
	BasicSegStart Opcode = 0
	// BasicSegEnd 基础段终止码位（含）。
	BasicSegEnd Opcode = 169
	// FunctionSegStart 函数段起始码位。
	FunctionSegStart Opcode = 170
	// FunctionSegEnd 函数段终止码位（含）。
	FunctionSegEnd Opcode = 224
	// ModuleSegStart 模块段起始码位。
	ModuleSegStart Opcode = 225
	// ModuleSegEnd 模块段终止码位（含）。
	ModuleSegEnd Opcode = 250
	// ExtensionSegStart 扩展段起始码位。
	ExtensionSegStart Opcode = 251
	// ExtensionSegEnd 扩展段终止码位（含）。
	ExtensionSegEnd Opcode = 253
)

// IsBasic 返回 opcode 是否属于基础段 [0-169]。
func (o Opcode) IsBasic() bool {
	return o >= BasicSegStart && o <= BasicSegEnd
}

// IsFunction 返回 opcode 是否属于函数段 [170-224]。
func (o Opcode) IsFunction() bool {
	return o >= FunctionSegStart && o <= FunctionSegEnd
}

// IsModule 返回 opcode 是否属于模块段 [225-250]。
func (o Opcode) IsModule() bool {
	return o >= ModuleSegStart && o <= ModuleSegEnd
}

// IsExtension 返回 opcode 是否属于扩展段 [251-253]。
func (o Opcode) IsExtension() bool {
	return o >= ExtensionSegStart && o <= ExtensionSegEnd
}

// IsReserved 返回 opcode 是否为系统保留（254-255），不得用于任何指令。
func (o Opcode) IsReserved() bool {
	return o >= 254
}

// IsAssigned 返回 opcode 是否已分配（在指令集中有定义）。
// 注意：IsAssigned 仅检查段位，具体 opcode 是否注册需查询注册表。
func (o Opcode) IsAssigned() bool {
	return !o.IsReserved()
}

// UnlockSegMaxOpcode 是解锁脚本中允许的最大 opcode（[0-50]），
// SYS_NULL(169) 是唯一例外，可突破此限制。
const UnlockSegMaxOpcode Opcode = 50

// IsAllowedInUnlock 返回 opcode 是否允许在解锁脚本中使用。
// 允许范围为 [0-50]，以及 SYS_NULL(169) 特例。
// SCRIPT(17)/VALUE(18) 虽在 [0-50] 内，但已被禁用，不得出现在主网有效解锁脚本中。
// 参考：DEC-0505、docs/proposal/Instruction/AGENTS.md
func (o Opcode) IsAllowedInUnlock() bool {
	if o.IsDisabled() {
		return false
	}
	return o <= UnlockSegMaxOpcode || o == SYS_NULL
}

// 前期禁用指令清单（主网公共验证路径触达即 ScriptError）。
// 参考：DEC-0505、docs/proposal/Instruction/AGENTS.md
var disabledOpcodes = map[Opcode]struct{}{
	SCRIPT: {},
	VALUE:  {},
	EVAL:   {},
	INOUT:  {},
}

// IsDisabled 返回 opcode 是否为前期禁用指令。
// 禁用指令在公共验证实际执行路径触达时产生 ScriptError，
// 仅因静态出现在脚本中不会产生错误。
func (o Opcode) IsDisabled() bool {
	_, ok := disabledOpcodes[o]
	return ok
}
