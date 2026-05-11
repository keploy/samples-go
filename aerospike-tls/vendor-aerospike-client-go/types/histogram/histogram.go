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
	"errors"
	"fmt"
	"math"
	"strings"
)

type Type byte

const (
	Linear Type = iota
	Logarithmic
)

type hvals interface {
	~int | ~uint |
		~int64 | ~int32 | ~int16 | ~int8 |
		~uint64 | ~uint32 | ~uint16 | ~uint8 |
		~float64 | ~float32
}

type Histogram[T hvals] struct {
	htype Type
	base  T

	Buckets []uint64 `json:"buckets"` // slot -> count
	Min     T        `json:"min"`
	Max     T        `json:"max"`
	Sum     float64  `json:"sum"`
	Count   uint64   `json:"count"`
}

func New[T hvals](htype Type, base T, buckets int) *Histogram[T] {
	return &Histogram[T]{
		htype:   htype,
		base:    base,
		Buckets: make([]uint64, buckets),
	}
}

func NewLinear[T hvals](base T, buckets int) *Histogram[T] {
	return &Histogram[T]{
		htype:   Linear,
		base:    base,
		Buckets: make([]uint64, buckets),
	}
}

func NewExponential[T hvals](base T, buckets int) *Histogram[T] {
	return &Histogram[T]{
		htype:   Logarithmic,
		base:    base,
		Buckets: make([]uint64, buckets),
	}
}

func (h *Histogram[T]) Reset() {
	for i := range h.Buckets {
		h.Buckets[i] = 0
	}

	h.Min = 0
	h.Max = 0
	h.Sum = 0
	h.Count = 0
}

func (h *Histogram[T]) Reshape(htype Type, base T, buckets int) {
	if h.htype == htype && h.base == base && len(h.Buckets) == buckets {
		return
	}

	h.htype = htype
	h.base = base
	h.Buckets = make([]uint64, buckets)

	h.Min = 0
	h.Max = 0
	h.Sum = 0
	h.Count = 0
}

func (h *Histogram[T]) String() string {
	res := new(strings.Builder)
	switch h.htype {
	case Linear:
		for i := 0; i < len(h.Buckets)-1; i++ {
			v := float64(h.base) * float64(i)
			fmt.Fprintf(res, "[%v, %v) => %d\n", v, v+float64(h.base), h.Buckets[i])
		}
		fmt.Fprintf(res, "[%v, inf) => %d\n", float64(h.base)*float64(len(h.Buckets)-1), h.Buckets[len(h.Buckets)-1])
	case Logarithmic:
		fmt.Fprintf(res, "[0, %v) => %d\n", float64(h.base), h.Buckets[0])
		for i := 1; i < len(h.Buckets)-1; i++ {
			v := math.Pow(float64(h.base), float64(i))
			fmt.Fprintf(res, "[%v, %v) => %d\n", v, v*float64(h.base), h.Buckets[i])
		}
		fmt.Fprintf(res, "[%v, inf) => %d\n", math.Pow(float64(h.base), float64(len(h.Buckets))-1), h.Buckets[len(h.Buckets)-1])
	}
	return res.String()
}

func (h *Histogram[T]) Clone() *Histogram[T] {
	b := make([]uint64, len(h.Buckets))
	copy(b, h.Buckets)
	return &Histogram[T]{
		htype: h.htype,
		base:  h.base,

		Buckets: b,
		Min:     h.Min,
		Max:     h.Max,
		Sum:     h.Sum,
		Count:   h.Count,
	}
}

func (h *Histogram[T]) CloneAndReset() *Histogram[T] {
	res := h.Clone()
	h.Reset()
	return res
}

func (h *Histogram[T]) Merge(other *Histogram[T]) error {
	if h.base != other.base || h.htype != other.htype || len(h.Buckets) != len(other.Buckets) {
		return errors.New("Histograms to not match")
	}

	if other.Min < h.Min || h.Min == 0 {
		h.Min = other.Min
	}

	if other.Max > h.Max {
		h.Max = other.Max
	}

	h.Sum += other.Sum
	h.Count += uint64(other.Count)

	for i := range h.Buckets {
		h.Buckets[i] += other.Buckets[i]
	}

	return nil
}

func (h *Histogram[T]) Average() float64 {
	if h.Count > 0 {
		return h.Sum / float64(h.Count)
	}
	return 0
}

func (h *Histogram[T]) Median() T {
	var s uint64 = 0
	c := h.Count / 2
	for i, bv := range h.Buckets {
		s += bv
		if s >= c {
			// found the bucket
			if h.htype == Linear {
				return T(i+1) * h.base
			}
			return T(math.Pow(float64(h.base), float64(i+1)))
		}
	}
	return h.Max
}

func (h *Histogram[T]) Add(v T) {
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

	h.Sum += float64(v)
	h.Count++

	var slot int
	if v > 0 {
		switch h.htype {
		case Linear:
			slot = int(math.Floor(float64(v / T(h.base))))
		case Logarithmic:
			slot = int(math.Floor(math.Log(float64(v)) / math.Log(float64(h.base))))
		}
	}

	if slot >= len(h.Buckets) {
		h.Buckets[len(h.Buckets)-1]++
	} else if slot < 0 {
		h.Buckets[0]++
	} else {
		h.Buckets[slot]++
	}
}
