package utils

import "math"

func DayCntBetweenTimestamp(timestamp1 int64, timestamp2 int64) int64 {
	if timestamp1 < timestamp2 {
		timestamp1, timestamp2 = timestamp2, timestamp1
	}
	res := (timestamp1 - timestamp2) / (24 * 3600)
	return res
}

func FormatFloat(f float64, round int32) float64 {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return f
	}
	return math.Round(f*1000) / 1000
}
