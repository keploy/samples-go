// Copyright 2014-2024 Aerospike, Inc.
//
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

package histogram_test

import (
	"testing"

	"github.com/aerospike/aerospike-client-go/v7/types/histogram"
)

var (
	_median    int
	_medianu64 uint64
)

func Benchmark_Histogram_Linear_Add(b *testing.B) {
	h := histogram.New[int](histogram.Linear, 5, 10)
	for i := 0; i < b.N; i++ {
		h.Add(i)
	}
}

func Benchmark_Histogram_Linear_Median(b *testing.B) {
	h := histogram.New[int](histogram.Linear, 50, 101)
	for i := 0; i < 10000; i++ {
		h.Add(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_median = h.Median()
	}
}

func Benchmark_Histogram_Log_Add(b *testing.B) {
	h := histogram.NewExponential[int](2, 10)
	for i := 0; i < b.N; i++ {
		h.Add(i)
	}
}

func Benchmark_Histogram_Log_Median(b *testing.B) {
	h := histogram.NewExponential[int](2, 32)
	for i := 0; i < 100000; i++ {
		h.Add(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_median = h.Median()
	}
}

func Benchmark_Histogram_Log2_Add(b *testing.B) {
	h := histogram.NewLog2(10)
	for i := 0; i < b.N; i++ {
		h.Add(uint64(i))
	}
}

func Benchmark_Histogram_Log2_Median(b *testing.B) {
	h := histogram.NewLog2(32)
	for i := 0; i < 100000; i++ {
		h.Add(uint64(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_medianu64 = h.Median()
	}
}
