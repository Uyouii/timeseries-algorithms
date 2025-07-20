package kde

import "github.com/uyouii/timeseries-algorithms/model"

func factorial(n int) float64 {
	result := 1.0
	for i := 2; i <= n; i++ {
		result *= float64(i)
	}
	return result
}

func linspace(start, stop float64, num int) []float64 {
	if num < 2 {
		return []float64{start}
	}
	step := (stop - start) / float64(num-1)
	grid := make([]float64, num)
	for i := 0; i < num; i++ {
		grid[i] = start + float64(i)*step
	}
	return grid
}

func Clip(x []float64, weights []float64, clip *model.Clip) ([]float64, []float64) {
	if len(x) != len(weights) || clip == nil {
		// do nothing
		return x, weights
	}

	resX, resWeight := []float64{}, []float64{}
	n := len(x)
	for i := 0; i < n; i++ {
		if x[i] > 0 && x[i] >= clip.Lower && x[i] <= clip.Upper {
			resX = append(resX, x[i])
			resWeight = append(resWeight, weights[i])
		}
	}
	return resX, resWeight
}

func InitOnes(n int) []float64 {
	res := make([]float64, 0, n)
	for i := 0; i < n; i++ {
		res = append(res, 1)
	}
	return res
}

func IntMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}
