// Copyright 2014-2022 Aerospike, Inc.
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

package atomic_test

import (
	"github.com/aerospike/aerospike-client-go/v7/internal/atomic"

	gg "github.com/onsi/ginkgo/v2"
	gm "github.com/onsi/gomega"
)

var _ = gg.Describe("TypedVal", func() {

	gg.Context("Storage must support", func() {

		gg.Context("Primitives", func() {

			gg.It("int", func() {
				var t int = 5
				var tv atomic.TypedVal[int]
				tv.Set(t)
				gm.Expect(tv.Get()).To(gm.Equal(t))
			})

			gg.It("string", func() {
				var t string = "Hello!"
				var tv atomic.TypedVal[string]
				tv.Set(t)
				gm.Expect(tv.Get()).To(gm.Equal(t))
			})

			gg.It("slice", func() {
				var t = []int{1, 2, 3}
				var tv atomic.TypedVal[[]int]
				tv.Set(t)
				gm.Expect(tv.Get()).To(gm.Equal(t))

				tv.Set(nil)
				var tt []int
				gm.Expect(tv.Get()).To(gm.Equal(tt))
			})

			gg.It("map", func() {
				var t = map[string]int{"a": 1, "b": 2, "c": 3}
				var tv atomic.TypedVal[map[string]int]
				tv.Set(t)
				gm.Expect(tv.Get()).To(gm.Equal(t))

				tv.Set(nil)
				var tt map[string]int
				gm.Expect(tv.Get()).To(gm.Equal(tt))
			})

		})

		gg.Context("Pointers", func() {

			gg.It("*int", func() {
				var t int = 5
				var tv atomic.TypedVal[*int]
				tv.Set(&t)
				gm.Expect(tv.Get()).To(gm.Equal(&t))

				tv.Set(nil)
				var tt *int
				gm.Expect(tv.Get()).To(gm.Equal(tt))
			})

			gg.It("*string", func() {
				var t string = "Hello!"
				var tv atomic.TypedVal[*string]
				tv.Set(&t)
				gm.Expect(tv.Get()).To(gm.Equal(&t))

				tv.Set(nil)
				var tt *string
				gm.Expect(tv.Get()).To(gm.Equal(tt))
			})

			gg.It("slice", func() {
				var t = []int{1, 2, 3}
				var tv atomic.TypedVal[*[]int]
				tv.Set(&t)
				gm.Expect(tv.Get()).To(gm.Equal(&t))

				t = nil
				tv.Set(&t)
				gm.Expect(tv.Get()).To(gm.Equal(&t))

				tv.Set(nil)
				var tt *[]int
				gm.Expect(tv.Get()).To(gm.Equal(tt))
			})

			gg.It("map", func() {
				var t = map[string]int{"a": 1, "b": 2, "c": 3}
				var tv atomic.TypedVal[*map[string]int]
				tv.Set(&t)
				gm.Expect(tv.Get()).To(gm.Equal(&t))

				t = nil
				tv.Set(&t)
				gm.Expect(tv.Get()).To(gm.Equal(&t))

				tv.Set(nil)
				var tt *map[string]int
				gm.Expect(tv.Get()).To(gm.Equal(tt))
			})

		})

	})

})
