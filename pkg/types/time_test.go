package types

import (
	"testing"
	"time"
)

func TestConstants(t *testing.T) {
	if BlockInterval != 6*time.Minute {
		t.Fatalf("BlockInterval = %v, want 6m", BlockInterval)
	}
	if BlocksPerYear != 87661 {
		t.Fatalf("BlocksPerYear = %d, want 87661", BlocksPerYear)
	}
}

func TestHeightYear(t *testing.T) {
	tests := []struct {
		h    BlockHeight
		want uint32
	}{
		{0, 0},
		{87660, 0},
		{87661, 1},
		{87661 * 2, 2},
	}
	for _, tt := range tests {
		if got := HeightYear(tt.h); got != tt.want {
			t.Errorf("HeightYear(%d) = %d, want %d", tt.h, got, tt.want)
		}
	}
}

func TestIsYearBoundary(t *testing.T) {
	tests := []struct {
		h    BlockHeight
		want bool
	}{
		{0, true},
		{1, false},
		{BlocksPerYear, true},
		{BlocksPerYear + 1, false},
		{BlocksPerYear * 2, true},
	}
	for _, tt := range tests {
		if got := IsYearBoundary(tt.h); got != tt.want {
			t.Errorf("IsYearBoundary(%d) = %v, want %v", tt.h, got, tt.want)
		}
	}
}

func TestBlockTime(t *testing.T) {
	genesis := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if got := BlockTime(genesis, 0); !got.Equal(genesis) {
		t.Errorf("BlockTime(genesis, 0) = %v, want %v", got, genesis)
	}
	want1 := genesis.Add(6 * time.Minute)
	if got := BlockTime(genesis, 1); !got.Equal(want1) {
		t.Errorf("BlockTime(genesis, 1) = %v, want %v", got, want1)
	}
}
