package client

import (
	"math"
)

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
