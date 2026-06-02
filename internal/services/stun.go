package services

// STUN 是 STUN/stun2p 公共服务的接口边界
// (第 15 章 §1，DEC-0603).
//
// STUN 提供 NAT 穿透探测与打洞辅助，以支持节点之间的 UDP 直连 P2P 通信。
// 实现位于外部仓库（github.com/cxio/stun2p）；该接口定义边界契约。
// STUN 不参与区块链共识、区块验证、交易执行、PoH 或脚本验证。
// 服务不可用不会影响区块合法性。
type STUN interface {
	// Probe 为本地节点发起 NAT 穿透探测。
	// 返回观测到的外部地址字符串；探测失败时返回错误。
	Probe() (externalAddr string, err error)

	// Assist 请求打洞辅助以连接远端节点。
	// peerAddr 为目标节点地址，格式为 host:port。
	// 请求失败时返回错误。
	Assist(peerAddr string) error
}

// BaseNetwork 是基础 P2P 网络（节点发现层）的接口边界
// (第 15 章 §1，DEC-0603).
//
// 基础网络提供轻量级 P2P 节点发现与通用广播能力，
// 所有服务与应用子网都依赖该基础能力。
// 实现位于外部仓库（github.com/cxio/p2p）；该接口定义边界契约。
// 基础网络不参与区块/交易/PoH/脚本验证。
// 服务不可用不会影响区块合法性。
type BaseNetwork interface {
	// Broadcast 向本地子网中所有已连接节点广播数据。
	// 无法发起广播时返回错误。
	Broadcast(data []byte) error

	// ConnectedPeers 返回当前已连接节点数量。
	ConnectedPeers() int
}
