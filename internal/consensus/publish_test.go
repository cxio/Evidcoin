package consensus

import "testing"

func TestPublishStateAdvance(t *testing.T) {
	cases := []struct {
		name    string
		ops     []PublishStage // 依次调用 AdvanceTo 的目标阶段
		wantErr []bool         // 对应 AdvanceTo 是否期望报错
		want    PublishStage   // 最终 stage
	}{
		{
			name:    "正常三段推进",
			ops:     []PublishStage{PublishStageBlockProof, PublishStageBlockSummary, PublishStageTxData},
			wantErr: []bool{false, false, false},
			want:    PublishStageTxData,
		},
		{
			name:    "直接跳到阶段3",
			ops:     []PublishStage{PublishStageTxData},
			wantErr: []bool{false},
			want:    PublishStageTxData,
		},
		{
			name:    "重复推进同一阶段",
			ops:     []PublishStage{PublishStageBlockProof, PublishStageBlockProof},
			wantErr: []bool{false, true},
			want:    PublishStageBlockProof,
		},
		{
			name:    "回退到更早阶段",
			ops:     []PublishStage{PublishStageBlockSummary, PublishStageBlockProof},
			wantErr: []bool{false, true},
			want:    PublishStageBlockSummary,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewPublishState()
			for i, stage := range tc.ops {
				err := s.AdvanceTo(stage)
				gotErr := err != nil
				if gotErr != tc.wantErr[i] {
					t.Errorf("op[%d] AdvanceTo(%d): gotErr=%v, wantErr=%v", i, stage, gotErr, tc.wantErr[i])
				}
			}
			if s.Stage() != tc.want {
				t.Errorf("Stage(): got %d, want %d", s.Stage(), tc.want)
			}
		})
	}
}

func TestPublishStateIsComplete(t *testing.T) {
	s := NewPublishState()
	if s.IsComplete() {
		t.Error("new state should not be complete")
	}
	_ = s.AdvanceTo(PublishStageTxData)
	if !s.IsComplete() {
		t.Error("after TxData stage should be complete")
	}
}

func TestPublishStateIdle(t *testing.T) {
	s := NewPublishState()
	if s.Stage() != PublishStageIdle {
		t.Errorf("initial stage: got %d, want %d", s.Stage(), PublishStageIdle)
	}
}
