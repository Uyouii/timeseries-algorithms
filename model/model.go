package model

import (
	"fmt"
	"time"
)

type ChangePointType int

const (
	IncreaseChangePoint ChangePointType = 1
	DecreaseChangePoint ChangePointType = 2
)

type ChangePoint struct {
	ChangePointType ChangePointType
	TimeValue       TimeValue
}

type TimeValue struct {
	Time  time.Time
	Value float64
}

func (v *TimeValue) Less(timeValue TimeValue) bool {
	return v.Value < timeValue.Value
}

func (v *TimeValue) Before(timeValue TimeValue) bool {
	return v.Time.Before(timeValue.Time)
}

type TimeSeries struct {
	// Labels contains label key -> label value, like "instance": "localhost:9091"
	Labels map[string]string
	Values []TimeValue
}

func (s *TimeSeries) DebugString() string {
	res := fmt.Sprintf("labels: %+v, valueCount: %+v", s.Labels, len(s.Values))
	return res
}

func (s *TimeSeries) IsEmpty() bool {
	if s == nil {
		return true
	}
	return len(s.Values) == 0
}

type DailyStatisticsData struct {
	Mean                 float64 `json:"mean,omitempty"`
	RecentMean           float64 `json:"recent_mean,omitempty"`
	RecentStddev         float64 `json:"recent_stddev,omitempty"`
	RecentVariance       float64 `json:"recent_var,omitempty"`
	NormalMean           float64 `json:"normal_mean,omitempty"`        // user zscore to filter some nodes
	NormalVariance       float64 `json:"normal_var,omitempty"`         // use zscore to filter some nodes
	RecentNormalMean     float64 `json:"recent_normal_mean,omitempty"` // use zscore to filter some nodes
	RecentNormalVariance float64 `json:"recent_normal_var,omitempty"`
	UpdateTimestamp      int64   `json:"update_timestamp,omitempty"`
}

func (d *DailyStatisticsData) Valid(expiredDuration time.Duration) bool {
	if d == nil || d.UpdateTimestamp == 0 {
		return false
	}
	if d.Mean == 0 || d.RecentNormalMean == 0 || d.NormalMean == 0 {
		return false
	}
	if time.Since(time.Unix(d.UpdateTimestamp, 0)) > expiredDuration {
		return false
	}

	return true
}
