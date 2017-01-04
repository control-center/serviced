package statistics

type Point interface {
	X() float64
	Y() float64
}

func LeastSquares(series []Point) (m float64, b float64) {
	n := float64(len(series))

	if n == 0 {
		return
	}

	var sumx, sumy, sumxx, sumxy float64

	for _, p := range series {
		x, y := p.X(), p.Y()
		sumx += x
		sumy += y
		sumxx += x * x
		sumxy += x * y
	}

	m = (n*sumxy - sumx*sumy) / (n*sumxx - sumx*sumx)
	b = (sumy / n) - (m * sumx / n)
	return
}
