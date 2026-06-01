package script

// ArgModel 描述指令的运行时实参数量模型。
// 参考：docs/proposal/10.Script-System.md §2（DEC-0501）
type ArgModel int8

const (
	// ArgNone 无需实参，不读取实参区。
	ArgNone ArgModel = 0
	// ArgVariadic 不定数量（-1）；只读实参区，空则空实参，不访问数据栈。
	ArgVariadic ArgModel = -1
	// ArgSemiFixed 半确定性（-2）；专用于 CALL，数量 = 2 + 附参值，运行时确定。
	ArgSemiFixed ArgModel = -2
)

// FixedArgs 返回指定固定实参数量的 ArgModel（n > 0）。
// 固定数量：实参区为空则取数据栈 n 个；实参区非空则数量必须恰为 n。
func FixedArgs(n int) ArgModel {
	if n <= 0 {
		panic("script: FixedArgs requires n > 0")
	}
	return ArgModel(n)
}

// IsFixed 返回 ArgModel 是否为固定数量（n > 0）。
func (m ArgModel) IsFixed() bool { return m > 0 }

// IsVariadic 返回 ArgModel 是否为不定数量（-1）。
func (m ArgModel) IsVariadic() bool { return m == ArgVariadic }

// IsSemiFixed 返回 ArgModel 是否为半确定性（-2，仅 CALL 使用）。
func (m ArgModel) IsSemiFixed() bool { return m == ArgSemiFixed }

// CostTier 描述指令成本等级，C-6 裁决前以占位枚举隔离，不可固化为具体数值。
// 参考：DEC-0504，待决问题 C-6。
type CostTier uint8

const (
	// CostTierFree 极低成本指令（如 NOP、NIL、TRUE 等字面量）。
	CostTierFree CostTier = 0
	// CostTierLow 低成本指令（如 PUSH、POP、栈操作类）。
	CostTierLow CostTier = 1
	// CostTierMedium 中等成本指令（如算术、比较、逻辑类）。
	CostTierMedium CostTier = 2
	// CostTierHigh 较高成本指令（如哈希函数类）。
	CostTierHigh CostTier = 3
	// CostTierExternal 外部引用成本（GOTO/EMBED/SCRIPT/INOUT，先计数再解析）。
	CostTierExternal CostTier = 4
	// CostTierUnknown 成本等级未确定（C-6 开放，待后续裁决）。
	CostTierUnknown CostTier = 255
)

// Availability 描述指令的可用性标记（组合位标志）。
type Availability uint8

const (
	// AvailUnlock 指令可在解锁脚本（opcode [0-50]+ SYS_NULL 特例）中使用。
	AvailUnlock Availability = 1 << 0
	// AvailPublic 指令可在公共验证路径中执行。
	AvailPublic Availability = 1 << 1
	// AvailPrivate 指令可在私有验证路径中执行。
	AvailPrivate Availability = 1 << 2
	// AvailDisabled 指令当前被禁用（主网公共路径触达即 ScriptError）。
	AvailDisabled Availability = 1 << 3
)

// Metadata 是每条指令的完整元数据描述。
// 所有导出指令必须在 registry.go 中注册完整元数据。
// 参考：docs/proposal/Instruction/0.Base-Constraints.md §3（对 Plan 的约束第 3 点）
type Metadata struct {
	// Opcode 指令码（1 字节）。
	Opcode Opcode
	// Mnemonic 助记符（英文）。
	Mnemonic string
	// Category 指令类别（1-18）。
	Category uint8
	// ArgCount 运行时实参数量模型。
	ArgCount ArgModel
	// ReturnCount 返回值数量；-1 表示不定（自动展开）。
	ReturnCount int
	// AttrParamSizes 附参字节数序列（按声明顺序，0=变长 ULEB128，>0=固定宽度大端）。
	// 多个附参按此顺序连续编码，不插分隔符（DEC-0501）。
	AttrParamSizes []int
	// AssocDataParam 关联数据由第几个附参（从 0 起）给出其长度；-1 表示无关联数据。
	AssocDataParam int
	// Availability 可用性位标志组合。
	Availability Availability
	// Deterministic 指令是否确定性（公共验证路径要求确定性）。
	Deterministic bool
	// CostTier 成本等级（C-6 裁决前以 CostTierUnknown 占位）。
	CostTier CostTier
	// Description 简短说明（中文或英文）。
	Description string
	// ErrorScenarios 已知错误场景列表（供测试参考）。
	ErrorScenarios []string
}

// HasAssocData 返回指令是否有关联数据。
func (m *Metadata) HasAssocData() bool { return m.AssocDataParam >= 0 }

// IsUnlockSafe 返回指令是否可安全用于解锁脚本。
func (m *Metadata) IsUnlockSafe() bool { return m.Availability&AvailUnlock != 0 }

// IsPublic 返回指令是否可在公共验证路径中执行。
func (m *Metadata) IsPublic() bool { return m.Availability&AvailPublic != 0 }

// IsPrivate 返回指令是否可在私有路径中执行。
func (m *Metadata) IsPrivate() bool { return m.Availability&AvailPrivate != 0 }

// IsDisabled 返回指令是否被禁用（主网前期禁用清单）。
func (m *Metadata) IsDisabled() bool { return m.Availability&AvailDisabled != 0 }
