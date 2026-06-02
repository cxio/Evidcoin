package validation

import (
	"github.com/cxio/evidcoin/internal/tx"
	"github.com/cxio/evidcoin/pkg/types"
)

// Manager 是校验组管理层的接口（第 13 章 §1.1）。
// 管理层负责交易校验分发、冗余控制、合法交易汇总、业绩记录与区块交互。
// 组间通讯线格式属 C-10 外包，本接口仅定义协议约束。
type Manager interface {
	// SubmitTask 接收来自守卫者的准合法交易并存入待验证池。
	SubmitTask(task Task) error
	// AssignTasks 向指定校验员派发任务（含冗余度 >= 2 的适当冗余分配）。
	AssignTasks(validatorID string) ([]Task, error)
	// ReceiveResult 接收校验员完整校验结果（要求无条件反馈）。
	ReceiveResult(result TaskResult) error
	// GuardianDeliveries 返回与守卫者关联的递送记录（供组间反馈触发重复核使用）。
	GuardianDeliveries(guardianID string) []types.TxID
}

// Guardian 是校验组守卫者的接口（第 13 章 §1）。
// 守卫者接收外部传入交易、执行首领校验、向管理层提交准合法交易。
type Guardian interface {
	// ReceiveExternal 接收外部传入交易（来自其它校验组守卫者或直接提交）。
	// 随机抉择：约 50% 概率先执行首领校验再转发，其余直接转发（共约性优化）。
	ReceiveExternal(txData []byte) error
	// ForwardToManager 将通过首领校验的准合法交易提交给管理层。
	ForwardToManager(task Task) error
	// NotifyIllegal 通知来源校验员其转发的交易被本组判定为非法（组间反馈）。
	NotifyIllegal(txID types.TxID, sourceValidatorID string) error
}

// Validator 是校验组校验员的接口（第 13 章 §1）。
// 校验员向管理层请求任务、执行完整校验并无条件反馈结果。
type Validator interface {
	// RequestTasks 向管理层请求校验任务。
	RequestTasks() ([]Task, error)
	// SubmitResult 向管理层提交完整校验结果（无条件，包含合法/非法/拒绝/错误）。
	SubmitResult(result TaskResult) error
	// ForwardValid 向其它校验组守卫者传送本校验员已验证合法的交易。
	ForwardValid(txID types.TxID, txData []byte) error
}

// UTXOCache 是组内 UTXO 缓存器接口（第 13 章 §1.2）。
// 缓存当前 UTXO 集，提供未花费币金输出查询；完成状态指纹计算后通知管理层拉取。
// UTXO 缓存器与 UTCO 缓存器逻辑类似但独立运行。
type UTXOCache interface {
	// IsUnspent 查询指定 OutPoint 是否未花费（币金语义）。
	IsUnspent(ref tx.OutPoint) bool
	// Root 返回当前 UTXO 集状态指纹（TreeHash，DEC-0201）。
	Root() types.TreeHash
	// NotifyReady 通知管理层当前 UTXO 指纹已就绪，可拉取用于区块打包。
	NotifyReady(root types.TreeHash)
}

// UTCOCache 是组内 UTCO 缓存器接口（第 13 章 §1.2）。
// 结构与 UTXOCache 同形但独立运行，负责凭信集的未转出条目查询与指纹计算。
type UTCOCache interface {
	// IsUnspent 查询指定 OutPoint 是否未转出（凭信语义）。
	IsUnspent(ref tx.OutPoint) bool
	// Root 返回当前 UTCO 集状态指纹（TreeHash，DEC-0201）。
	Root() types.TreeHash
	// NotifyReady 通知管理层当前 UTCO 指纹已就绪，可拉取用于区块打包。
	NotifyReady(root types.TreeHash)
}

// ScriptRefCache 是外部脚本引用缓存器接口（第 13 章 §1.2）。
// 缓存 GOTO/EMBED/SCRIPT 指令引用的外部脚本；需及时清理与实时更新。
// 可与 UTXO/UTCO 缓存器同机运行。
type ScriptRefCache interface {
	// Get 按脚本 TxID 检索外部脚本字节；缓存未命中返回 nil, false。
	Get(scriptID types.TxID) ([]byte, bool)
	// Put 注册外部脚本；若已存在相同 TxID 则覆盖（实时更新）。
	Put(scriptID types.TxID, script []byte)
	// Evict 从缓存中移除指定 TxID 的脚本（及时清理过期/无用引用）。
	Evict(scriptID types.TxID)
}
