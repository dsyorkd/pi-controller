package server

import (
	"fmt"
	"math"
)

// safeUintToUint32 safely converts uint to uint32, checking for overflow
func safeUintToUint32(val uint) (uint32, error) {
	if val > math.MaxUint32 {
		return 0, fmt.Errorf("integer overflow: value %d exceeds uint32 max", val)
	}
	return uint32(val), nil
}

// safeIntToInt32 safely converts int to int32, checking for overflow
func safeIntToInt32(val int) (int32, error) {
	if val > math.MaxInt32 {
		return 0, fmt.Errorf("integer overflow: value %d exceeds int32 max", val)
	}
	if val < math.MinInt32 {
		return 0, fmt.Errorf("integer overflow: value %d below int32 min", val)
	}
	return int32(val), nil
}

// safeInt64ToInt32 safely converts int64 to int32, checking for overflow
func safeInt64ToInt32(val int64) (int32, error) {
	if val > math.MaxInt32 {
		return 0, fmt.Errorf("integer overflow: value %d exceeds int32 max", val)
	}
	if val < math.MinInt32 {
		return 0, fmt.Errorf("integer overflow: value %d below int32 min", val)
	}
	return int32(val), nil
}

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
