package statistics

type Point struct {
	X, Y float64
}

func LeastSquares(series []Point) (m float64, b float64) {
	n := float64(len(series))

	if n == 0 {
		return
	}

	var sumx, sumy, sumxx, sumxy float64

	for _, p := range series {
		sumx += p.X
		sumy += p.Y
		sumxx += p.X * p.X
		sumxy += p.X * p.Y
	}

	m = (n*sumxy - sumx*sumy) / (n*sumxx - sumx*sumx)
	b = (sumy / n) - (m * sumx / n)
	return
}
