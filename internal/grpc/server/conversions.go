package server

import (
	"math"
)

// mustSafeUintToUint32 safely converts uint to uint32 or returns 0 on overflow
// Use this when you want to gracefully handle overflow without error propagation
func mustSafeUintToUint32(val uint) uint32 {
	if val > math.MaxUint32 {
		return 0
	}
	return uint32(val)
}

// mustSafeIntToInt32 safely converts int to int32 or returns 0 on overflow
// Use this when you want to gracefully handle overflow without error propagation
func mustSafeIntToInt32(val int) int32 {
	if val > math.MaxInt32 {
		return math.MaxInt32
	}
	if val < math.MinInt32 {
		return math.MinInt32
	}
	return int32(val)
}

// mustSafeInt64ToInt32 safely converts int64 to int32 or returns max/min on overflow
func mustSafeInt64ToInt32(val int64) int32 {
	if val > math.MaxInt32 {
		return math.MaxInt32
	}
	if val < math.MinInt32 {
		return math.MinInt32
	}
	return int32(val)
}
