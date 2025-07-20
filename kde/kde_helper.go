package kde

import (
	"context"
	"fmt"
	"math"

	"github.com/uyouii/timeseries-algorithms/common"
	"github.com/uyouii/timeseries-algorithms/model"
	"github.com/uyouii/timeseries-algorithms/utils"
	"go.uber.org/zap"
	"gonum.org/v1/gonum/stat"
)

func getMinCalculateSpeed() float64 {
	return KdeMinCalculateSpeed
}

func getMinCalculatePointCnt() int {
	return KdeMinCalculatePointCnt
}

// kde algorithms need the record values,
// then calcualte the kde confidence
func CalculateKdeConfidences(ctx context.Context, timestamp int64,
	recordValues []model.RecordValue) (*model.KdeConfidence, error) {
	logger := utils.GetLogger(ctx)

	defer func() {
		if err := recover(); err != nil {
			logger.Error("CalculateKdeConfidences recover panic error!", zap.Any("err", err),
				zap.String("panic info", utils.GetPanicInfo()), zap.Any("recordValues", recordValues))
		}
	}()

	values, weights := []float64{}, []float64{}

	for _, recordValue := range recordValues {
		// if record value is zero, don't need calculate
		if recordValue.Value == 0 {
			continue
		}

		values = append(values, recordValue.Value)
		dayCntDiff := utils.DayCntBetweenTimestamp(timestamp, recordValue.Timestamp)
		weight := 1.0
		switch dayCntDiff {
		// improve the weight for the 1,7,30 day data
		case 1, 7, 30:
			weight = 1.0
		default:
			weight = math.Pow(KdeWeightDecayFactor, float64(dayCntDiff))
		}
		weights = append(weights, weight)
	}

	if len(values) < getMinCalculatePointCnt() {
		logger.Error("point too little, skip calculate", zap.Int("cnt", len(values)))
		return nil, common.ErrorInvalidValue
	}

	mean := stat.Mean(values, nil)
	stddev := stat.StdDev(values, nil)
	ZScoreUpper := mean + stddev*ClipUpperZScore
	ZScoreLower := math.Max(mean-stddev*ClipLowerZScore, 0)
	clip := &model.Clip{
		Upper: ZScoreUpper,
		Lower: ZScoreLower,
	}

	if mean < getMinCalculateSpeed() {
		logger.Error("metric speed is too low, don't need calcualte kde",
			zap.Float64("mean", mean))
		return nil, common.ErrorInvalidValue
	}

	k, err := NewKDEUnivariate(values, weights, 1.0, 4.0, clip)
	if err != nil {
		logger.Error("NewKDEUnivariate failed", zap.Error(err))
		return nil, err
	}

	calculatedQuantiles := map[string]*model.QuantileValue{}

	for _, value := range AllCalculateQuantiles {
		quantile, err := k.Quantile(value)
		if err != nil {
			logger.Error("kde Quantile failed", zap.Error(err), zap.Float64("value", value))
			continue
		}
		quantile.Value = utils.FormatFloat(quantile.Value, 3)
		calculatedQuantiles[fmt.Sprintf("%v", value)] = quantile
	}

	return &model.KdeConfidence{
		QuantileValues: calculatedQuantiles,
	}, nil
}
