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

import "time"

var (
	// LeastSquaresPredictor uses the ordinary least squares method of
	// estimation to predict future values
	LeastSquaresPredictor = &olsPredictor{}
)

// Predictor represents a strategy for predicting a future value based on
// historic values
type Predictor interface {
	// Predict uses the timestamp/value pairs passed in to predict the value at
	// time now+period
	Predict(period time.Duration, timestamps, values []float64) float64
}

type olsPredictor struct{}

func (p *olsPredictor) Predict(period time.Duration, timestamps, values []float64) float64 {
	// Get the timestamp for which we're going to predict the value
	then := float64(time.Now().UTC().Add(period).Unix())

	// Use least squares to find the line of best fit
	m, b := LeastSquares(timestamps, values)

	// Get the predicted future value using the line
	return then*m + b
}
