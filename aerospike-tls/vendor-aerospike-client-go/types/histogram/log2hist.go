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

package histogram

import (
	"fmt"
	"strings"
)

type Log2 struct {
	Buckets  []uint64 // slot -> count
	Min, Max uint64
	Sum      uint64
	Count    uint64
}

func NewLog2(buckets int) *Log2 {
	return &Log2{
		Buckets: make([]uint64, buckets),
	}
}

func (h *Log2) Reset() {
	for i := range h.Buckets {
		h.Buckets[i] = 0
	}

	h.Min = 0
	h.Max = 0
	h.Sum = 0
	h.Count = 0
}

func (h *Log2) String() string {
	res := new(strings.Builder)

	fmt.Fprintf(res, "[0, %v) => %d\n", 2, h.Buckets[0])
	for i := 1; i < len(h.Buckets)-1; i++ {
		v := 2 << i
		fmt.Fprintf(res, "[%v, %v) => %d\n", v, v*2, h.Buckets[i])
	}
	fmt.Fprintf(res, "[%v, inf) => %d\n", 2<<len(h.Buckets)-1, h.Buckets[len(h.Buckets)-1])

	return res.String()
}

func (h *Log2) Diff(old *Log2) *Log2 {
	diff := NewLog2(len(old.Buckets))
	for i := range h.Buckets {
		diff.Buckets[i] = h.Buckets[i] - old.Buckets[i]
	}
	return diff
}

func (h *Log2) Clone() *Log2 {
	new := Log2{
		Buckets: make([]uint64, len(h.Buckets)),
		Min:     h.Min,
		Max:     h.Max,
		Sum:     h.Sum,
		Count:   h.Count,
	}
	for i := range h.Buckets {
		new.Buckets[i] = h.Buckets[i]
	}
	return &new
}

func (h *Log2) Median() uint64 {
	var s uint64
	c := h.Count / 2
	for i, bv := range h.Buckets {
		s += bv
		if s >= c {
			return 1 << (i + 1)
		}
	}
	return h.Max
}

func (h *Log2) Add(v uint64) {
	if h.Count == 0 {
		h.Max = v
		h.Min = v
	} else {
		if v > h.Max {
			h.Max = v
		} else if v < h.Min {
			h.Min = v
		}
	}

	h.Sum += v
	h.Count++

	var slot int
	if v > 0 {
		slot = fastLog2(v)
	}

	if slot >= len(h.Buckets) {
		h.Buckets[len(h.Buckets)-1]++
	} else if slot < 0 {
		h.Buckets[0]++
	} else {
		h.Buckets[slot]++
	}
}

///////////////////////////////////////////////////////////////////

var log2tab64 = [64]int8{
	0, 58, 1, 59, 47, 53, 2, 60, 39, 48, 27, 54, 33, 42, 3, 61,
	51, 37, 40, 49, 18, 28, 20, 55, 30, 34, 11, 43, 14, 22, 4, 62,
	57, 46, 52, 38, 26, 32, 41, 50, 36, 17, 19, 29, 10, 13, 21, 56,
	45, 25, 31, 35, 16, 9, 12, 44, 24, 15, 8, 23, 7, 6, 5, 63,
}

// FastLog2 implements the FastLog2 function for uint64 values.
func fastLog2(value uint64) int {
	value |= value >> 1
	value |= value >> 2
	value |= value >> 4
	value |= value >> 8
	value |= value >> 16
	value |= value >> 32

	return int(log2tab64[(value*0x03f6eaf2cd271461)>>58])
}
