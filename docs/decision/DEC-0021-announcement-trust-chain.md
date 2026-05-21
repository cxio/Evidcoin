# DEC-0021: Announcement Trust Chain Removed（全网通告信任链已移除）

Status: Deprecated

## Context

旧草案曾定义全网通告为文本消息 Proof 输出，并引入客户端公告根公钥集合。最近 conception 修订已经取消全网通告设计，当前 conception 中不再保留该协议边界。

## Decision

- 不再定义全网通告信任链。
- 不再定义公告根公钥集合、链上授权通告、通告标题格式或公告撤销规则。
- 客户端发布、版本提示、节点公告或运营消息属于应用/发布流程，不进入当前协议 Decision。

## Rationale

Decision 不能保留已被 conception 取消的协议功能。继续保留公告根会伪造信任根，并影响创世规范边界。

## Consequences

旧 `Announcement:<Level>:` 格式、公告根开放问题和链上授权 Proof 格式全部废弃。如未来恢复通告设计，必须先在 conception 中重新提出。

## Conception references

- `docs/conception/2.共识-端点约定.md`

## Open questions

无。
