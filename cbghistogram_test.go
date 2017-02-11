// Copyright © 2017 Couchbase, Inc.
//
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

package ghistogram

import (
	"math/rand"
	"testing"
)

func TestNewCbHistogram(t *testing.T) {
	tests := []struct {
		name string
		bins int
	}{
		{"TestNewCbHistogram_bins_1", 1},
		{"TestNewCbHistogram_bins_10", 10},
	}

	for _, test := range tests {
		hist := NewCbHistogram(test.name, test.bins)
		if hist._name != test.name {
			t.Errorf("CbHistogram name doesn not match!")
		}

		// Account for 2 additional bins: smallest and largest
		// created by the CbHistogram
		if len(hist._bins) != test.bins+2 {
			t.Errorf("Correct number of bins not created!")
		}
	}
}

func TestCbHistogramAdd(t *testing.T) {
	tests := []struct {
		name       string
		bins       int
		samples    int
		mod_factor uint64
	}{
		{"TestAdd_10_10000_10", 10, 10000, 10},
		{"TestAdd_10_20000_5", 10, 20000, 5},
	}

	for _, test := range tests {
		hist := NewCbHistogram(test.name, test.bins)
		for i := 0; i < test.samples; i++ {
			hist.Add(uint64(rand.Uint32())%test.mod_factor, 1)
		}

		count := 0
		for i := 0; i < len(hist._bins); i++ {
			count += int(hist._bins[i]._count)
		}

		if count != test.samples {
			t.Errorf("Incorrect number of samples in the histogram!")
		}
	}
}

func TestParallelCbHistogramAdd(t *testing.T) {
	hist := NewCbHistogram("TestParallelAdd", 10)

	ch := make(chan int)

	sample_count_1 := 100000
	sample_count_2 := 200000

	go func() {
		for i := 0; i < sample_count_1; i++ {
			hist.Add(uint64(rand.Uint32())%10, 1)
		}
		ch <- 1
	}()

	go func() {
		for i := 0; i < sample_count_2; i++ {
			hist.Add(uint64(rand.Uint32())%10, 1)
		}
		ch <- 1
	}()

	<-ch
	<-ch

	count := 0
	for i := 0; i < len(hist._bins); i++ {
		count += int(hist._bins[i]._count)
	}

	if count != sample_count_1+sample_count_2 {
		t.Errorf("Incorrect number of samples in the histogram!")
	}
}

func TestCbHistogramGraph(t *testing.T) {
	hist := NewCbHistogram("TestGraph", 5)

	for i := 0; i < 10000; i++ {
		hist.Add(uint64(i)%5, 1)
	}

	buf := hist.EmitGraph()

	exp := `TestGraph (10000 Total)
[0 - 1]       2000   20.00% ###############
[1 - 2]       2000   20.00% ###############
[2 - 4]       4000   40.00% ##############################
[4 - 8]       2000   20.00% ###############
`
	got := buf.String()
	if got != exp {
		t.Errorf("Expected:\n%s\n, But got:\n%s", exp, got)
	}
}

func BenchmarkCbHistogramAdd(b *testing.B) {
	hist := NewCbHistogram("BenchmarkAdd", 100)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		hist.Add(uint64(i), 1)
	}
}

func BenchmarkCbHistogramGraph(b *testing.B) {
	hist := NewCbHistogram("BenchMarkGraph", 100)

	for i := 0; i < b.N; i++ {
		hist.Add(uint64(i), 1)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		hist.EmitGraph()
	}
}
