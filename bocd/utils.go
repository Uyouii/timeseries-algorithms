package bocd

import (
	"context"
	"math"
	"time"

	"github.com/uyouii/timeseries-algorithms/model"
	"github.com/uyouii/timeseries-algorithms/utils"
	"go.uber.org/zap"
)

func LogSumExp(data []float64) float64 {
	max := math.Inf(-1)
	for _, v := range data {
		max = math.Max(max, v)
	}
	res := 0.0
	for i := range data {
		res += math.Exp(data[i] - max)
	}
	return math.Log(res) + max
}

func NormalizeData(data []float64) []float64 {
	logSum := LogSumExp(data)
	res := make([]float64, len(data))
	for i := range data {
		res[i] = data[i] - logSum
	}
	return res
}

func ListExp(data []float64) []float64 {
	res := make([]float64, len(data))
	for i, v := range data {
		res[i] = math.Exp(v)
	}
	return res
}

func getObserveDuration() time.Duration {
	return 5 * time.Minute
}

func getChangePointThreshold() float64 {
	return 0.75
}

func hazard() float64 {
	return 2 / 1000.0
}

func IntMin(i1, i2 int) int {
	if i1 < i2 {
		return i1
	}
	return i2
}

func ListMul(l1, l2 []float64) []float64 {
	listLen := IntMin(len(l1), len(l2))

	res := make([]float64, listLen)
	for i := 0; i < listLen; i++ {
		res[i] = l1[i] * l2[i]
	}
	return res
}

func RemoveOldChangePoints(changePoints []*model.ChangePoint) []*model.ChangePoint {
	res := []*model.ChangePoint{}

	var MaxDuration = 24 * time.Hour
	startTime := time.Now().Add(-1 * MaxDuration)

	for _, changePoint := range changePoints {
		if changePoint.TimeValue.Time.Before(startTime) {
			continue
		}
		res = append(res, changePoint)
	}
	return res
}

func getMaxTracebackDuration() time.Duration {
	return 15 * time.Minute
}

func GetChangePointCountInRecentTime(duration time.Duration, changePoints []*model.ChangePoint) int {
	startTime := time.Now().Add(-1 * duration)
	res := 0
	for _, changePoint := range changePoints {
		if changePoint.TimeValue.Time.After(startTime) {
			res += 1
		}
	}
	return res
}

// TODO: need a calcualte data daily statistics data process
// Important: this is a function to get timeSeries data daily statistics data
func GetNormalStatisticData(ctx context.Context, timeSeriesKey string) (*model.DailyStatisticsData, error) {
	logger := utils.GetLogger(ctx)

	// speedHelper := speed_helper.GetSpeedHelper()
	// dailyStatisticsData, err := speedHelper.GetDailyStatisticsData(ctx, service, checkType, labels)
	// if err != nil {
	// 	logger.Error("GetDailyStatisticsData failed", zap.Error(err))
	// 	return nil, err
	// }

	res := &model.DailyStatisticsData{}

	logger.Info("GetDailyStatisticsData success", zap.Any("dailyStatisticsData", res))
	return res, nil
}
