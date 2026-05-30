package types

import "time"

// BlockHeight is a block height. Its underlying type is fixed to uint32 to match
// the block header Height field (DEC-0001 定宽白名单); any height that feeds a
// block header hash/signature must be encoded via AppendUint32BE.
type BlockHeight uint32

// HeightYear returns the height-based year index of h, i.e. h / BlocksPerYear.
// This is the 87661-block year boundary used for year blocks; it is distinct
// from the UTC CalendarYear used by transaction short references (do not mix).
func HeightYear(h BlockHeight) uint32 {
	return uint32(h) / BlocksPerYear
}

// IsYearBoundary reports whether height h sits on a height-based year boundary
// (a multiple of BlocksPerYear). The genesis height 0 is a year boundary.
func IsYearBoundary(h BlockHeight) bool {
	return uint32(h)%BlocksPerYear == 0
}

// BlockTime returns the canonical timestamp of the block at height h, computed
// as genesis plus h block intervals.
func BlockTime(genesis time.Time, h BlockHeight) time.Time {
	return genesis.Add(time.Duration(h) * BlockInterval)
}
