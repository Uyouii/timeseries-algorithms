package kde

import (
	"math"

	"github.com/uyouii/timeseries-algorithms/model"
)

type Kernel interface {
	NormalReferenceConstant() float64
}

type GuassianKernel struct {
	l2Norm                  float64
	kernelVar               float64
	order                   int
	normalReferenceConstant float64
	h                       float64
	weights                 []float64
	domain                  *model.Clip
}

func NewGuassianKernel() *GuassianKernel {
	return &GuassianKernel{
		l2Norm:                  1.0 / (2.0 * math.Sqrt(math.Pi)),
		kernelVar:               1.0,
		order:                   2.0,
		normalReferenceConstant: 0,
		h:                       1.0,
		weights:                 nil,
	}
}

func (k *GuassianKernel) SetH(h float64) {
	k.h = h
}

func (k *GuassianKernel) SetDomain(domain *model.Clip) {
	k.domain = domain
}

func (k *GuassianKernel) SetWeights(weights []float64) {
	sum := 0.0
	for _, v := range weights {
		sum += v
	}
	kernelWeights := make([]float64, len(weights))
	if sum != 0 {
		for i := range weights {
			kernelWeights[i] = weights[i] / sum
		}
	}
	k.weights = kernelWeights
}

func (k *GuassianKernel) Shape(x float64) float64 {
	return 0.3989422804014327 * math.Exp(-x*x/2.0)
}

func (k *GuassianKernel) EvaluateMatrix(matrix [][]float64) [][]float64 {
	rows := len(matrix)
	if rows == 0 {
		return matrix
	}
	cols := len(matrix[0])
	result := make([][]float64, rows)
	for i := 0; i < rows; i++ {
		result[i] = make([]float64, cols)
		for j := 0; j < cols; j++ {
			result[i][j] = max(k.Shape(matrix[i][j]), 0)
		}
	}
	return result
}

func (k *GuassianKernel) NormalReferenceConstant() float64 {
	nu := k.order
	if k.normalReferenceConstant == 0 {
		numerator := math.Pow(math.Pi, 0.5) * math.Pow(factorial(nu), 3) * k.l2Norm
		denom := 2.0 * float64(nu) * factorial(2*nu) * math.Pow(k.Moments(nu), 2)
		C := 2 * math.Pow(numerator/denom, 1.0/float64(2*nu+1))
		k.normalReferenceConstant = C
	}
	return k.normalReferenceConstant
}

func (k *GuassianKernel) Moments(n int) float64 {
	if n == 1 {
		return 0
	}
	if n == 2 {
		return k.kernelVar
	}
	return 1.0
}

func (k *GuassianKernel) Density(xs []float64, x float64) float64 {
	n := len(xs)

	if len(xs) == 0 {
		return math.NaN()
	}

	h := k.h
	var sum float64 = 0.0

	if k.weights != nil {
		for i, xi := range xs {
			u := (xi - x) / h
			sum += k.Shape(u) * k.weights[i]
		}
		return (1 / h) * sum
	}

	for _, xi := range xs {
		u := (xi - x) / h
		sum += k.Shape(u)
	}
	return (1 / (h * float64(n))) * sum

}
