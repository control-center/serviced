// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build unit

package metrics

import (
	"testing"
)

var rr bool

func BenchmarkLoggingOnCallingFunctionWithLogging(b *testing.B) {
	for i := 0; i < b.N; i++ {
		testFunction(true, functionWithLogging)
	}
}

func BenchmarkLoggingOffCallingFunctionWithLogging(b *testing.B) {
	for i := 0; i < b.N; i++ {
		testFunction(false, functionWithLogging)
	}
}
func BenchmarkLoggingOnCallingFunctionWithoutLogging(b *testing.B) {
	for i := 0; i < b.N; i++ {
		testFunction(true, functionWithoutLogging)
	}
}

func BenchmarkLoggingOffCallingFunctionWithoutLogging(b *testing.B) {
	for i := 0; i < b.N; i++ {
		testFunction(false, functionWithoutLogging)
	}
}

func testFunction(loggingEnabled bool, f func(m *Metrics) bool) {
	m := NewMetrics()
	defer m.Stop(m.Start("BenchmarkLogging"))
	m.Enabled = loggingEnabled
	defer m.LogAndCleanUp(m.Start("BenchmarkLogging"))
	r := f(m) // record result to prevent compiler optimization of call to f. (See https://goo.gl/caaEOU)
	rr = r    // Assign result to package level variable to prevent compiler optimizing this function away (See https://goo.gl/caaEOU)
}

func functionWithLogging(m *Metrics) bool {
	defer m.Stop(m.Start("functionWithLogging"))
	return true
}

func functionWithoutLogging(_ *Metrics) bool {
	return true
}
