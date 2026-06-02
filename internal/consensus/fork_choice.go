package consensus

import (
	"bytes"

	"github.com/cxio/evidcoin/pkg/types"
)

// 分叉链段竞争（第 12 章 §4，DEC-0303）。
//
// 比较流程：
//   1. 新分叉长度必须 <= ForkAcceptLimit(20)，否则拒绝接收。
//   2. 最多比较 ForkEvalLength(31) 个高度。
//   3. 每高度经 SelectCandidate 归一化后取有效候选块的 MintHash 计分。
//   4. 任一链先达 ForkDecisiveThreshold(16) 分即胜出。
//   5. 31 高度仍平局进入 RandomX 裁决（见 decision.go）。

// ForkSide 标识分叉哪侧胜出。
type ForkSide int8

const (
	// ForkSideNone 表示仍处平局，需 RandomX 裁决。
	ForkSideNone ForkSide = 0
	// ForkSideA 表示 A 链胜出。
	ForkSideA ForkSide = 1
	// ForkSideB 表示 B 链胜出。
	ForkSideB ForkSide = -1
)

// ForkSegment 描述分叉点之后一条链某高度的候选块集合。
// 每个元素对应分叉后的一个区块高度；同一高度可能存在多个竞争块（冗余出块）。
type ForkSegment [][]ForkBlock

// CompareForkSegments 比较两条分叉链段并返回胜负结论（DEC-0303，第 12 章 §4）。
//
// segA、segB 各包含从分叉点之后若干高度的候选块集合，按高度升序排列。
// 比较步骤：
//  1. 若 len(segA) 或 len(segB) 超过 ForkAcceptLimit(20)，对应链视为可接收上限内
//     仅前 ForkAcceptLimit 个高度参与比较（调用方已保证长度约束，此处只做截断保护）。
//  2. 取 min(len(segA), len(segB), ForkEvalLength) 作为有效比较窗口。
//  3. 每高度调用 SelectCandidate 归一化，取 MintHash；若某高度无有效候选块则该高度双方都不得分。
//  4. 任一链先达 ForkDecisiveThreshold(16) 分即胜出。
//  5. 比较窗口耗尽仍未决 → ForkSideNone（需 RandomX 裁决）。
func CompareForkSegments(segA, segB ForkSegment) ForkSide {
	maxLen := types.ForkEvalLength // 31
	n := len(segA)
	if len(segB) < n {
		n = len(segB)
	}
	if n > maxLen {
		n = maxLen
	}

	scoreA, scoreB := 0, 0
	threshold := types.ForkDecisiveThreshold // 16

	for i := range n {
		candidateA, okA := SelectCandidate(segA[i])
		candidateB, okB := SelectCandidate(segB[i])

		if !okA || !okB {
			// 一方或双方无有效候选，该高度不计分
			continue
		}

		cmp := bytes.Compare(candidateA.MintHash[:], candidateB.MintHash[:])
		switch {
		case cmp < 0:
			scoreA++
		case cmp > 0:
			scoreB++
			// cmp == 0：MintHash 完全相等，双方均不得分（DEC-0303）
		}

		if scoreA >= threshold {
			return ForkSideA
		}
		if scoreB >= threshold {
			return ForkSideB
		}
	}

	// 比较完成仍平局
	if scoreA > scoreB {
		return ForkSideA
	}
	if scoreB > scoreA {
		return ForkSideB
	}
	return ForkSideNone
}

// ValidateForkLength 检查新发现分叉的长度是否在接收上限内（<= ForkAcceptLimit = 20）。
// 超出上限返回 ErrForkTooLong；在限内返回 nil（第 12 章 §4，DEC-0303）。
func ValidateForkLength(forkLen int) error {
	if forkLen > types.ForkAcceptLimit {
		return ErrForkTooLong
	}
	return nil
}
