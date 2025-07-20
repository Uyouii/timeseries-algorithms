package model

import "fmt"

type Clip struct {
	Lower float64
	Upper float64
}

type Density struct {
	X     float64
	Value float64
}

type Cdf struct {
	X     float64
	Value float64
}

type QuantileValue struct {
	Value    float64 `json:"v,omitempty"`
	Quantile float64 `json:"q,omitempty"`
}

type ConfidenceInterval struct {
	Lower *QuantileValue `json:"l,omitempty"`
	Upper *QuantileValue `json:"u,omitempty"`
}

type KdeConfidence struct {
	QuantileValues map[string]*QuantileValue `json:"quantiles,omitempty"`
}

func (c *KdeConfidence) GetQuantileValue(value float64) (*QuantileValue, bool) {
	if c == nil || c.QuantileValues == nil {
		return nil, false
	}
	valueStr := fmt.Sprintf("%v", value)
	quantile, ok := c.QuantileValues[valueStr]
	return quantile, ok
}

type RecordValue struct {
	Timestamp int64   `json:"t,omitempty"`
	Value     float64 `json:"v,omitempty"`
}
