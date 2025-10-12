package client

import (
	"fmt"
	"math"
)

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

// mustSafeIntToInt32 safely converts int to int32 or clamps to max/min on overflow
func mustSafeIntToInt32(val int) int32 {
	if val > math.MaxInt32 {
		return math.MaxInt32
	}
	if val < math.MinInt32 {
		return math.MinInt32
	}
	return int32(val)
}
