# DEC-0021: Announcement Trust Chain（全网通告信任链）

Status: Proposed

## Context

conception 定义全网通告为文本消息 Proof 输出，签名者由官方 App 内置或授权链上的授权公钥决定。旧方案使用创世特定签名公钥作为初始公告根，缺少 conception 依据。

## Decision

建议仅保留以下边界：

- 客户端可内置一组公告根公钥或公告根公钥哈希，但具体初始集合必须由发布规范给出。
- 链上授权通告必须引用上一有效公告授权，并由当前有效公告密钥签名。
- 通告标题必须以前缀 `Announcement:<Level>:` 开始。
- 全网通告不得承载自动升级、脚本执行或权限扩展语义。

## Rationale

Decision 不能指定 conception 未给出的创世公告密钥。将其降为 Proposed 可保留实现方向，同时避免伪造信任根。

## Consequences

正式客户端发布前必须有人工作出公告根配置和轮换格式。

## Conception references

- `docs/conception/2.共识-端点约定.md`

## Open questions

- 初始公告根公钥集合由哪个发布工件承载。
- 链上授权 Proof 的 payload 精确格式。
- 公告撤销和过期规则。
