package kde

import (
	"math"
	"sort"

	"github.com/uyouii/timeseries-algorithms/common"
	"github.com/uyouii/timeseries-algorithms/model"
	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/gonum/integrate/quad"
)

// Attach the density estimate to the KDEUnivariate class.
type KDEUnivariate struct {
	Weights []float64

	//If gridsize is 0, max(len(x), 50) is used.
	gridSize int

	// An adjustment factor for the bw. Bandwidth becomes bw * adjust.
	bwAdjust float64

	// Defines the length of the grid past the lowest and highest values
	// of x so that the kernel goes to zero. The end points are
	// ``min(x) - cut * adjust * bw`` and ``max(x) + cut * adjust * bw``.
	cut float64

	// endogenous variable
	Endog []float64

	density []model.Density
	cdf     []model.Cdf
	grid    []float64
	bw      float64
	fited   bool
	kernel  *GuassianKernel
}

func NewKDEUnivariate(endog []float64, weights []float64,
	bwAdjust float64, cut float64, clip *model.Clip) (*KDEUnivariate, error) {
	sort.Float64s(endog)

	if len(endog) == 0 {
		return nil, common.ErrorInvalidValue
	}

	if len(weights) == 0 {
		weights = InitOnes(len(endog))
	} else if len(weights) != len(endog) {
		return nil, common.ErrorInvalidValue
	}

	if cut == 0 {
		cut = 3
	}

	gridSize := IntMax(len(endog), 100)

	if clip != nil {
		endog, weights = Clip(endog, weights, clip)
	}

	kde := &KDEUnivariate{
		Weights:  weights,
		gridSize: gridSize,
		bwAdjust: bwAdjust,
		cut:      cut,
		Endog:    endog,
	}

	return kde, nil
}

func (kde *KDEUnivariate) Kdensity() ([]model.Density, float64) {
	if kde.fited {
		return kde.density, kde.bw
	}

	kernel := NewGuassianKernel()
	bandWidth := NewNormalReferenceBandWidth(kernel)

	bw := bandWidth.BandWidth(kde.Endog)

	bw = bw * kde.bwAdjust
	kernel.SetH(bw)

	a := max(floats.Min(kde.Endog)-kde.cut*1.5*bw, 0)
	// a := floats.Min(kde.Endog) - kde.cut*bw
	b := floats.Max(kde.Endog) + kde.cut*bw
	grid := linspace(a, b, kde.gridSize)

	matrix := make([][]float64, len(grid))
	for i := 0; i < len(grid); i++ {
		matrix[i] = make([]float64, len(kde.Endog))
		for j := 0; j < len(kde.Endog); j++ {
			matrix[i][j] = (kde.Endog[j] - grid[i]) / bw
		}
	}

	matrix = kernel.EvaluateMatrix(matrix)

	q := floats.Sum(kde.Weights)

	dens := make([]float64, len(grid))
	for i := 0; i < len(grid); i++ {
		dens[i] = floats.Dot(matrix[i], kde.Weights) / (q * bw)
	}

	res := []model.Density{}
	for i := 0; i < len(dens); i++ {
		res = append(res, model.Density{
			X:     grid[i],
			Value: dens[i],
		})
	}

	kde.density = res
	kde.bw = bw
	kde.grid = grid
	kde.fited = true
	kde.kernel = kernel
	kde.kernel.SetWeights(kde.Weights)

	return res, bw
}

func (kde *KDEUnivariate) Cdf() ([]model.Cdf, error) {
	if !kde.fited {
		kde.Kdensity()
	}

	if len(kde.cdf) > 0 {
		return kde.cdf, nil
	}

	a, _ := 0.0, math.Inf(1)
	// a, _ := math.Inf(-1), math.Inf(1)
	newGrid := []float64{a}
	newGrid = append(newGrid, kde.grid...)
	gridsize := len(newGrid)

	f := func(x float64) float64 {
		return kde.kernel.Density(kde.Endog, x)
	}

	res := []model.Cdf{}

	var cumSum float64

	for i := 1; i < gridsize; i++ {
		integral := quad.Fixed(f, newGrid[i-1], newGrid[i], 50, nil, 0)
		cumSum += integral
		res = append(res, model.Cdf{
			X:     newGrid[i],
			Value: cumSum,
		})
	}

	kde.cdf = res
	return res, nil
}

func (kde *KDEUnivariate) Quantile(p float64) (*model.QuantileValue, error) {
	cdf, err := kde.Cdf()
	if err != nil {
		return nil, err
	}

	if len(cdf) == 0 {
		return nil, nil
	}
	if p <= cdf[0].Value {
		return &model.QuantileValue{
			Quantile: p,
			Value:    cdf[0].X,
		}, nil
	}

	if p >= cdf[len(cdf)-1].Value {
		return &model.QuantileValue{
			Quantile: p,
			Value:    cdf[len(cdf)-1].X,
		}, nil
	}

	for i := 1; i < len(cdf); i++ {
		if cdf[i].Value > p {
			lowerX, lowerP := cdf[i-1].X, cdf[i-1].Value
			upperX, upperP := cdf[i].X, cdf[i].Value
			value := lowerX + (upperX-lowerX)*(p-lowerP)/(upperP-lowerP)
			return &model.QuantileValue{
				Quantile: p,
				Value:    value,
			}, nil
		}
	}
	return &model.QuantileValue{
		Quantile: p,
		Value:    cdf[len(cdf)-1].X,
	}, nil
}
