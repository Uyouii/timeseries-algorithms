package bocd

import (
	"context"
	"fmt"
	"time"

	"github.com/uyouii/timeseries-algorithms/model"
	"github.com/uyouii/timeseries-algorithms/utils"
	"go.uber.org/zap"
)

// LastTriggerPointTime: all the container last check trigger point time
type BocdTriggerData struct {
	TriggeredChangePoints []*model.ChangePoint `json:"triggered_change_points"`
	LastTriggerPointTime  time.Time            `json:"last_trigger_point_time"`
}

func NewBocdTriggerData() *BocdTriggerData {
	return &BocdTriggerData{
		TriggeredChangePoints: []*model.ChangePoint{},
		LastTriggerPointTime:  time.Time{},
	}
}

// the bocd handler need cache some status and data in memory
// because our task seheduler is distributed on different containers
// so we need store some data in redis so that different containers will see the same data status
// and each container handler will handle all the time series data
// and the bocd handler will confirm that each change point will only trigger once

type BocdHandler struct {
	onlineChecker      *BocdOnlineChecker
	newChangePoints    []*model.ChangePoint // local new generate change point
	bocdTriggerData    *BocdTriggerData
	lastAppendDataTime time.Time // last append time series to handler
	timeSeriesKey      string
	varx               float64
	mean0              float64
}

func NewBocdHandler(ctx context.Context, timeSeriesKey string) (*BocdHandler, bool) {
	// get varx, mean0
	logger := utils.GetLogger(ctx)

	dailyStatisticsData, err := GetNormalStatisticData(ctx, timeSeriesKey)
	if err != nil {
		logger.Error("GetNormalStatisticData failed", zap.Error(err))
		return nil, false
	}

	varx, mean0 := dailyStatisticsData.RecentNormalVariance, dailyStatisticsData.RecentNormalMean

	return &BocdHandler{
		onlineChecker:      NewBocdOnlineChecker(varx, mean0),
		newChangePoints:    []*model.ChangePoint{},
		bocdTriggerData:    NewBocdTriggerData(),
		timeSeriesKey:      timeSeriesKey,
		varx:               varx,
		mean0:              mean0,
		lastAppendDataTime: time.Time{},
	}, true
}

// bocd algorithm need cache the history data in memory
// so need rebalance the cache data cycle cyclical so that won't occupy so many memeory
func (m *BocdHandler) rebalance(ctx context.Context, timeValue model.TimeValue) {
	logger := utils.GetLogger(ctx)

	// this means if PreSmoothMinute don't have chagne point
	// will reuse reserveMinute to regenerate the bocd checker
	const PreSmoothMinute, ReserveMinute = 360, 180 // 6h, 3h
	// if data count > 1440 minut(1 day), reset the cache
	const MaxDataSize = 1440

	lastChangePoint, ok := m.onlineChecker.LastChangePoint()
	needRebalance := (ok && timeValue.Time.Sub(lastChangePoint.TimeValue.Time) > PreSmoothMinute*time.Minute) ||
		(!ok && m.onlineChecker.DataSize() > PreSmoothMinute)
	if m.onlineChecker.DataSize() > MaxDataSize {
		needRebalance = true
	}

	if !needRebalance {
		return
	}

	datas := m.onlineChecker.Datas()
	reserveDatas := datas[len(datas)-ReserveMinute:]

	dailyStatisticsData, err := GetNormalStatisticData(ctx, m.timeSeriesKey)
	if err != nil {
		logger.Error("GetNormalStatisticData failed", zap.Error(err))
	} else {
		varx, mean0 := dailyStatisticsData.RecentNormalVariance, dailyStatisticsData.RecentNormalMean
		logger.Info("get new varx and mean0", zap.Float64("varx", varx), zap.Float64("mean0", mean0))
		m.varx, m.mean0 = varx, mean0
	}

	newOnlineChecker := NewBocdOnlineChecker(m.varx, m.mean0)
	for _, reserveTimeValue := range reserveDatas {
		newOnlineChecker.AppendPoint(ctx, reserveTimeValue)
	}
	m.onlineChecker = newOnlineChecker
	logger.Info("generage new online checker")
}

func (m *BocdHandler) appendPoint(ctx context.Context, timeValue model.TimeValue) (*model.ChangePoint, bool) {
	logger := utils.GetLogger(ctx)
	logger.Info("Begin Append Point", zap.Any("timeValue", timeValue))

	// 1. check whether need reblance
	m.rebalance(ctx, timeValue)

	// 2. append new point
	changePoint, found := m.onlineChecker.AppendPoint(ctx, timeValue)
	if found {
		logger.Info("find new change point", zap.Any("changePoint", changePoint))
		m.newChangePoints = append(m.newChangePoints, changePoint)
	}

	return changePoint, found
}

func (m *BocdHandler) AppendTimeSeriesData(ctx context.Context, timeSeries *model.TimeSeries) {
	logger := utils.GetLogger(ctx)
	// init can first append 2 hours data

	lastAppendDataTime := m.lastAppendDataTime
	if lastAppendDataTime.IsZero() {
		lastAppendDataTime = time.Now().Add(-2 * time.Hour)
	}

	foundCount := 0
	for _, timeValue := range timeSeries.Values {
		if !timeValue.Time.After(lastAppendDataTime) {
			continue
		}
		_, found := m.appendPoint(ctx, timeValue)
		if found {
			foundCount++
		}
	}

	if len(timeSeries.Values) > 0 {
		m.lastAppendDataTime = timeSeries.Values[len(timeSeries.Values)-1].Time
	}

	logger.Info(fmt.Sprintf("found %v change points", foundCount))
}

func (m *BocdHandler) PopNeedTriggerChangePoints(ctx context.Context) []*model.ChangePoint {
	logger := utils.GetLogger(ctx)

	// 1. if no change point need trigger, just return
	if len(m.newChangePoints) == 0 {
		logger.Info("no new change point")
		return nil
	}

	logger.Info(fmt.Sprintf("%v local chagne point need be checked", len(m.newChangePoints)))

	// 2. get trigger data from redis

	beginCheckTime := m.bocdTriggerData.LastTriggerPointTime
	// if a change point appear too long ago, don't check it
	minTracebackTime := time.Now().Add(-1 * getMaxTracebackDuration())
	if beginCheckTime.IsZero() || minTracebackTime.After(beginCheckTime) {
		beginCheckTime = minTracebackTime
	}

	// 3. check chagne points, remove the change point already triggered
	index := 0
	for ; index < len(m.newChangePoints); index++ {
		changePoint := m.newChangePoints[index]
		if changePoint.TimeValue.Time.After(beginCheckTime) {
			break
		}
	}
	m.newChangePoints = m.newChangePoints[index:]

	if len(m.newChangePoints) == 0 {
		logger.Info("no new local change point need trigger")
		return nil
	}

	// 4. get need trigger chagne point
	nowCheckTriggerTime := time.Now().Add(-1 * getObserveDuration())

	index = 0
	for ; index < len(m.newChangePoints); index++ {
		changePoint := m.newChangePoints[index]
		if changePoint.TimeValue.Time.After(nowCheckTriggerTime) {
			break
		}
	}

	needTriggerChangePoints := m.newChangePoints[:index]
	m.newChangePoints = m.newChangePoints[index:]

	// 5. reset redis trigger point data
	m.bocdTriggerData.TriggeredChangePoints = append(
		m.bocdTriggerData.TriggeredChangePoints, needTriggerChangePoints...)
	// remove some old chagne points, to prevent redis cache too big
	m.bocdTriggerData.TriggeredChangePoints = RemoveOldChangePoints(m.bocdTriggerData.TriggeredChangePoints)
	m.bocdTriggerData.LastTriggerPointTime = nowCheckTriggerTime

	if m.skipCheckChangePoints(ctx, m.newChangePoints, m.bocdTriggerData.TriggeredChangePoints) {
		logger.Info("too many change points in recent times, skip cpd check")
		return nil
	}

	logger.Info(fmt.Sprintf("%v new local change point need trigger", len(needTriggerChangePoints)))

	return needTriggerChangePoints
}

func (m *BocdHandler) skipCheckChangePoints(ctx context.Context,
	triggeredChangePonits, newChangePoints []*model.ChangePoint) bool {
	logger := utils.GetLogger(ctx)

	recentDuration := time.Minute * 30

	skipCheckChangePointCount := 10

	count := GetChangePointCountInRecentTime(recentDuration, triggeredChangePonits) +
		GetChangePointCountInRecentTime(recentDuration, newChangePoints)
	if count > skipCheckChangePointCount {
		logger.Info("too many chagne point in recent times", zap.Any("rcentTime", recentDuration),
			zap.Int("limitCount", skipCheckChangePointCount), zap.Int("changePointCnt", count))
		return true
	}
	return false
}
