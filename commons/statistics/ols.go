// Copyright 2017 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package statistics

// Mean calculates the mean of an array of floats
func Mean(series []float64) float64 {
	var n, x, sum float64
	for _, x = range series {
		sum += x
		n++
	}
	return sum / n
}

// LeastSquares calculates the slope and y-intercept of the line of best fit
// for the series of points represented as arrays of x- and y-coordinates using
// the Ordinary Least Squares method.
func LeastSquares(xs, ys []float64) (m, b float64) {

	lx, ly := len(xs), len(ys)

	// If the arrays are not of equal length, we can't do anything
	if lx != ly {
		return
	}

	// If we don't have at least two points, we can't do anything
	if lx < 2 {
		return
	}

	var (
		x, y, meanx, meany, sumxx, sumxy float64
		i                                int
	)

	// Since we're dealing with sums of squares, we'll need to shift the data
	// to be centered around the origin to prevent potential overflow or
	// numerical instability. We'll do this by subtracting the mean of x from
	// each x value, and the mean of y from each y value. This takes an extra
	// pass in each case, but will be much more accurate, since we won't
	// compound rounding errors produced by summing squares of large floats.
	meanx = Mean(xs)
	meany = Mean(ys)

	for i, x = range xs {
		y = ys[i] - meany
		x = x - meanx
		sumxx += x * x
		sumxy += x * y
	}

	// Least squares gets a lot simpler when you subtract the means, because
	// sum of x-values and sum of y-values are both zero.
	m = sumxy / sumxx

	// Shift our y-intercept back to account for our mean subtraction
	b = meany - (m * meanx)

	return
}
