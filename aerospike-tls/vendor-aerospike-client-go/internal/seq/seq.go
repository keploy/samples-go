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

package seq

import (
	"errors"
	"sync"
)

var Break = errors.New("Break")

func Do[T any](seq []T, f func(T) error) {
	for i := range seq {
		if err := f(seq[i]); err == Break {
			break
		}
	}
}

func ParDo[T any](seq []T, f func(T)) {
	if len(seq) == 0 {
		return
	}

	wg := new(sync.WaitGroup)
	wg.Add(len(seq))
	for i := range seq {
		go func(t T) {
			defer wg.Done()
			f(t)
		}(seq[i])
	}
	wg.Wait()
}

func Any[T any](seq []T, f func(T) bool) bool {
	for i := range seq {
		if f(seq[i]) {
			return true
		}
	}
	return false
}

func All[T any](seq []T, f func(T) bool) bool {
	if len(seq) == 0 {
		return false
	}

	for i := range seq {
		if !f(seq[i]) {
			return false
		}
	}
	return true
}

func Clone[T any](seq []T) []T {
	if seq == nil {
		return nil
	}

	if len(seq) == 0 {
		return []T{}
	}

	res := make([]T, len(seq))
	copy(res, seq)
	return res
}
