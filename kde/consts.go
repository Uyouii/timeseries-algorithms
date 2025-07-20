package kde

const (
	// 0.977159969 ^ 30 ~= 0.5
	// KdeWeightDecayFactor = 0.977159969
	// 0.9699187011 ^ 30 ~= 0.4
	// KdeWeightDecayFactor = 0.9699187011
	KdeWeightDecayFactor = 0.95365

	ClipUpperZScore = 3.0
	ClipLowerZScore = 3.0

	KdeMinCalculateSpeed    = 5.0
	KdeMinCalculatePointCnt = 5
)

var (
	AllCalculateQuantiles = []float64{0.01, 0.02, 0.03, 0.04, 0.05, 0.06, 0.07, 0.08, 0.09, 0.1, 0.11,
		0.12, 0.9, 0.91, 0.92, 0.93, 0.94, 0.95, 0.96, 0.97, 0.98, 0.99}
)
