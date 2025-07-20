package kde

import (
	"math"

	"gonum.org/v1/gonum/stat"
)

type BandWidth interface {
	BandWidth([]float64) float64
}

type NormalReferenceBandWidth struct {
	kernel Kernel
}

func NewNormalReferenceBandWidth(kernel Kernel) *NormalReferenceBandWidth {
	if kernel == nil {
		kernel = NewGuassianKernel()
	}
	return &NormalReferenceBandWidth{
		kernel: kernel,
	}
}

func (bw *NormalReferenceBandWidth) BandWidth(x []float64) float64 {
	C := bw.kernel.NormalReferenceConstant()
	A := selectSigma(x)
	n := len(x)
	return C * A * math.Pow(float64(n), -0.2)
}

func selectSigma(x []float64) float64 {
	normalize := 1.349

	q75 := stat.Quantile(0.75, stat.Empirical, x, nil)
	q25 := stat.Quantile(0.25, stat.Empirical, x, nil)
	iqr := (q75 - q25) / normalize

	stdDev := stat.StdDev(x, nil)

	if iqr > 0 {
		if stdDev < iqr {
			return stdDev
		}
		return iqr
	}
	return stdDev
}
