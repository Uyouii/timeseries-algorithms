package bocd

import (
	"context"
	"math"
	"time"

	"github.com/uyouii/timeseries-algorithms/model"
	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/gonum/stat/distuv"
)

// dynamic calculate update VarX
type BocdOnlineChecker struct {
	varX  float64 // known variance
	mean0 float64 // mean of the pre data

	datas           []model.TimeValue
	means           []float64
	invVariances    []float64 // 1 / Variance
	lastLogRunProbs []float64
	runLenLogProb   [][]float64
	runLenProb      [][]float64

	pMeans []float64 // prediction mean
	pVars  []float64 // prediction var

	changePoints         []*model.ChangePoint
	lastCheckTriggerTime time.Time
}

func NewBocdOnlineChecker(varx, mean0 float64) *BocdOnlineChecker {
	bocdChecker := &BocdOnlineChecker{
		varX:  varx,
		mean0: mean0,

		datas:           []model.TimeValue{},
		means:           []float64{mean0},
		invVariances:    []float64{1 / varx},
		runLenLogProb:   [][]float64{{0}},
		runLenProb:      [][]float64{{math.Inf(-1)}},
		lastLogRunProbs: []float64{0},

		pMeans: []float64{},
		pVars:  []float64{},

		changePoints:         []*model.ChangePoint{},
		lastCheckTriggerTime: time.Time{},
	}

	return bocdChecker
}

func (b *BocdOnlineChecker) LastTimeValue() (model.TimeValue, bool) {
	if len(b.datas) == 0 {
		return model.TimeValue{}, false
	}
	return b.datas[len(b.datas)-1], true
}

func (b *BocdOnlineChecker) AppendPoint(ctx context.Context, timeValue model.TimeValue) (*model.ChangePoint, bool) {
	changePoint, findChangePoint := b.appendPoint(timeValue)
	return changePoint, findChangePoint
}

func (b *BocdOnlineChecker) appendPoint(timeValue model.TimeValue) (*model.ChangePoint, bool) {
	b.datas = append(b.datas, timeValue)

	t := len(b.datas) // current time step

	// Make model predictions.
	b.pMeans = append(b.pMeans, b.predictionMean(t))
	b.pVars = append(b.pVars, b.predictionVar(t))

	// 3. Evaluate predictive probabilities.
	// logPreProbs is an array that calculates the probability density of the current point x under various run lengths
	logPreProbs := b.logOfPreProb(t, timeValue.Value)

	// 4. Calculate growth probabilities.
	// Growth probability, calculates the growth probability of the current point x under various run lengths,
	// log_growth_probs is also an array
	logGrowthProbs := b.calLogGrowthProbs(logPreProbs)

	// 5. Calculate changepoint probabilities.
	// Change point probability, calculates the probability that the current point is a change point, it's a scalar
	logChangePointProb := b.calLogChangePointProb(logPreProbs)

	// 6. Calculate evidence
	// Combine steps 4 and 5 to obtain new growth probabilities under various lengths
	logRunProbs := append([]float64{logChangePointProb}, logGrowthProbs...)
	b.lastLogRunProbs = logRunProbs

	// 7. Determine run length distribution.
	// Normalize to make the sum of these probabilities equal to 1
	normalizeLogRunProbs := NormalizeData(logRunProbs)
	// use normalizeLogRunProbs update runLenLogProb
	b.runLenLogProb = append(b.runLenLogProb, normalizeLogRunProbs)
	b.runLenProb = append(b.runLenProb, ListExp(normalizeLogRunProbs))

	// 8. update params
	b.updateGuassianParams(timeValue.Value)

	findChangePoint, chagnePoint := b.checkChangePoints(t)
	return chagnePoint, findChangePoint
}

func (b *BocdOnlineChecker) checkChangePoints(t int) (bool, *model.ChangePoint) {
	// calculate chagne points
	if len(b.runLenProb[t]) == 0 {
		return false, nil
	}

	observeMinutes := int(getObserveDuration().Minutes())
	threshold := getChangePointThreshold()

	for j := 0; j < len(b.runLenProb[t]) && j <= observeMinutes; j++ {
		if b.runLenProb[t][j] >= threshold {
			changePointLoc := int64(t - j)
			if changePointLoc == 0 {
				break
			}
			changePointTimeValue := b.datas[changePointLoc]

			changePoint := &model.ChangePoint{
				TimeValue: changePointTimeValue,
			}

			lastPoint := b.datas[changePointLoc-1]
			if changePointTimeValue.Value > lastPoint.Value {
				changePoint.ChangePointType = model.IncreaseChangePoint
			} else {
				changePoint.ChangePointType = model.DecreaseChangePoint
			}

			if len(b.changePoints) == 0 {
				b.changePoints = append(b.changePoints, changePoint)
				return true, changePoint
			}

			// if time point not equal last change point
			lastChangePoint := b.changePoints[len(b.changePoints)-1]
			if !lastChangePoint.TimeValue.Time.Equal(changePointTimeValue.Time) {
				return true, changePoint
			}

			break
		}
	}
	return false, nil
}

func (b *BocdOnlineChecker) updateGuassianParams(x float64) {
	newInvVariances := make([]float64, len(b.invVariances))
	for i := range b.invVariances {
		newInvVariances[i] = b.invVariances[i] + 1/b.varX
	}
	b.invVariances = append([]float64{1 / b.varX}, newInvVariances...)

	for i := range b.means {
		b.means[i] = (b.means[i]*b.invVariances[i] + x/b.varX) / newInvVariances[i]
	}
	b.means = append([]float64{b.mean0}, b.means...)
}

func (b *BocdOnlineChecker) logh() float64 {
	return math.Log(hazard())
}

func (b *BocdOnlineChecker) log1mh() float64 {
	return math.Log(1 - hazard())
}

func (b *BocdOnlineChecker) calLogChangePointProb(logPreProbs []float64) float64 {
	data := make([]float64, len(logPreProbs))

	for i := range logPreProbs {
		data[i] = logPreProbs[i] + b.lastLogRunProbs[i] + b.logh()
	}

	return LogSumExp(data)
}

func (b *BocdOnlineChecker) calLogGrowthProbs(logPreProbs []float64) []float64 {
	logGrowthProbs := make([]float64, len(logPreProbs))

	for i := range logPreProbs {
		logGrowthProbs[i] = logPreProbs[i] + b.lastLogRunProbs[i] + b.log1mh()
	}

	return logGrowthProbs
}

func (b *BocdOnlineChecker) logOfPreProb(t int, x float64) []float64 {
	// compute predictive probabilities
	// the posterior predictive for each run length hypothesis.
	logProbs := make([]float64, t)

	variances := b.calVariances()

	for i := 0; i < t; i++ {
		normalDist := distuv.Normal{
			Mu:    b.means[i],
			Sigma: math.Sqrt(variances[i]),
		}
		logProbs[i] = normalDist.LogProb(x)
	}

	return logProbs
}

func (b *BocdOnlineChecker) calVariances() []float64 {
	res := make([]float64, len(b.invVariances))
	for i := range b.invVariances {
		res[i] = 1/b.invVariances[i] + b.varX
	}
	return res
}

func (b *BocdOnlineChecker) predictionMean(t int) float64 {
	meanProbsValue := ListMul(ListExp(b.runLenLogProb[t-1]), b.means)
	return floats.Sum(meanProbsValue)
}

func (b *BocdOnlineChecker) predictionVar(t int) float64 {
	varProbsValue := ListMul(ListExp(b.runLenLogProb[t-1]), b.calVariances())
	return floats.Sum(varProbsValue)
}

func (b *BocdOnlineChecker) GetPredictionMeans() []float64 {
	return b.pMeans
}

func (b *BocdOnlineChecker) GetPredictionVariances() []float64 {
	return b.pVars
}

func (b *BocdOnlineChecker) Datas() []model.TimeValue {
	return b.datas
}

func (b *BocdOnlineChecker) DataSize() int {
	return len(b.datas)
}

func (b *BocdOnlineChecker) GetChangePoints() []*model.ChangePoint {
	return b.changePoints
}

func (b *BocdOnlineChecker) LastChangePoint() (*model.ChangePoint, bool) {
	if len(b.changePoints) > 0 {
		return b.changePoints[len(b.changePoints)-1], true
	}
	return nil, false
}
