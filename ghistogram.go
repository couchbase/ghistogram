//  Copyright (c) 2015 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the
//  License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing,
//  software distributed under the License is distributed on an "AS
//  IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
//  express or implied. See the License for the specific language
//  governing permissions and limitations under the License.

// Package ghistogram provides a simple histogram of uint64's that
// avoids heap allocations (garbage creation) during data processing.
package ghistogram

import (
	"bytes"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
)

// An individual bin of the histogram structure
type HistogramBin struct {
	_count uint64
	_start uint64
	_end   uint64
}

func (hb *HistogramBin) assign(src *HistogramBin) {
	hb._count = src._count
	hb._start = src._start
	hb._end = src._end
}

// Returns the count in this bin
func (hb *HistogramBin) count() uint64 {
	return atomic.LoadUint64(&hb._count)
}

// Increment this bin by the given amount
func (hb *HistogramBin) incr(amount uint64) {
	atomic.AddUint64(&hb._count, amount)
}

// Set a specific value for this bin
func (hb *HistogramBin) set(val uint64) {
	atomic.StoreUint64(&hb._count, val)
}

// Checks if this bin can contain the value
func (hb *HistogramBin) accepts(value uint64) bool {
	return value >= hb._start &&
		(value < hb._end || value == math.MaxUint64)
}

// A bin generator that generates bin ranges of the order:
// [n*i, n*(i+1)]
type MultipleGenerator struct {
	_start   uint64
	_factor  float64
	_initial int
}

func (mg *MultipleGenerator) getBin() *HistogramBin {
	var start, end uint64
	if mg._factor == 0.0 {
		start = mg._start
		mg._start += uint64(mg._initial)
		end = mg._start
	} else {
		if mg._initial != -1 {
			start = uint64(mg._initial)
			end = uint64(float64(mg._start) * mg._factor)
			mg._initial = -1
		} else {
			start = uint64(float64(mg._start) * mg._factor)
			end = uint64(float64(start) * mg._factor)
			mg._start = start
		}
	}

	return &HistogramBin{
		_count: 0,
		_start: start,
		_end:   end,
	}
}

// A bin generator that generates bin ranges of the order:
// [n^i, n^(i+1)]
type ExponentialGenerator struct {
	_start uint64
	_power float64
}

func (eg *ExponentialGenerator) getBin() *HistogramBin {
	start := uint64(math.Pow(eg._power, float64(eg._start)))
	eg._start++
	end := uint64(math.Pow(eg._power, float64(eg._start)))
	return &HistogramBin{
		_count: 0,
		_start: start,
		_end:   end,
	}
}

// Histogram is a simple uint64 histogram implementation that avoids
// heap allocations during its processing of incoming data points.
// These array of bins is public - in case users wish to use reflection
// or JSON marhsaling.
//
// --> Motivated for tracking simple performance timings.
// --> The histogram is concurrent safe.
type Histogram struct {
	// Name assogicated with the histogram
	_name string
	// Array of histogram bins
	_bins []HistogramBin

	m sync.Mutex
}

// Populates the histogram bins using the multiple generator
func (gh *Histogram) fillMultiples(mg *MultipleGenerator) {
	for i := 0; i < len(gh._bins); i++ {
		gh._bins[i].assign(mg.getBin())
	}
	gh.completeBinArray()
}

// Populates the histogram bins using the exponential generator
func (gh *Histogram) fillExponential(eg *ExponentialGenerator) {
	for i := 0; i < len(gh._bins); i++ {
		gh._bins[i].assign(eg.getBin())
	}
	gh.completeBinArray()
}

// Adds first and last bins if necessary
func (gh *Histogram) completeBinArray() {
	// If there will not naturally be one, create a bin for the
	// smallest possible value
	start_of_first_bin := gh._bins[0]._start
	if start_of_first_bin > 0 {
		hb := HistogramBin{
			_count: 0,
			_start: 0,
			_end:   start_of_first_bin,
		}
		gh._bins = append([]HistogramBin{hb}, gh._bins...)
	}

	// Also create one reaching to the largest possible value
	end_of_last_bin := gh._bins[len(gh._bins)-1]._end
	if end_of_last_bin < math.MaxUint64 {
		hb := HistogramBin{
			_count: 0,
			_start: end_of_last_bin,
			_end:   math.MaxUint64,
		}
		gh._bins = append(gh._bins, hb)
	}

	gh.verify()
}

// This validates that we're sorted and have no gaps or overlaps. Returns
// true if tests pass, else false
func (gh *Histogram) verify() bool {
	prev := uint64(0)
	for i := 0; i < len(gh._bins); i++ {
		if gh._bins[i]._start != prev {
			return false
		}
		prev = gh._bins[i]._end
	}
	if prev != math.MaxUint64 {
		return false
	}
	return true
}

// Finds the bin containing the specified amount. Returns index of last bin
// if not found
func (gh *Histogram) findBin(amount uint64) *HistogramBin {
	if amount == math.MaxUint64 {
		return &gh._bins[len(gh._bins)-1]
	}

	index := len(gh._bins) - 1
	for i := 0; i < len(gh._bins); i++ {
		if amount < gh._bins[i]._end {
			index = i
			break
		}
	}

	if !gh._bins[index].accepts(amount) {
		return &gh._bins[len(gh._bins)-1]
	}

	return &gh._bins[index]
}

// NewHistogram creates a new, ready to use Histogram.  The numBins
// must be >= 1.  The binFirst is the width of the first bin.  The
// binGrowthFactor must be > 1.0 or 0.0.
//
// Uses multiple generator to prepare the bins.
//
// A special case of binGrowthFactor of 0.0 means that the allocated
// bins will have constant, now growing size or "width"
func NewHistogram(
	numBins int,
	binFirst uint64,
	binGrowthFactor float64) *Histogram {

	mg := &MultipleGenerator{
		_start:   binFirst,
		_factor:  binGrowthFactor,
		_initial: int(binFirst),
	}

	gh := &Histogram{
		_name: "Histogram",
		_bins: make([]HistogramBin, numBins),
	}

	gh.fillMultiples(mg)

	return gh
}

// NewExpHistogram creates a new, ready to use Histogram. The numBins
// must be >= 1.
//
// Uses exponential generator to prepare the bins.
//
// If the growthFactor were less than or equal to 1.0, a default
// of 2.0 will be applied.
func NewExpHistogram(
	name string,
	numBins int,
	growthFactor float64) *Histogram {

	if growthFactor <= 1.0 {
		growthFactor = 2.0
	}

	eg := &ExponentialGenerator{
		_start: 0,
		_power: growthFactor,
	}

	gh := &Histogram{
		_name: name,
		_bins: make([]HistogramBin, numBins),
	}

	gh.fillExponential(eg)

	return gh
}

// Add a value to this histogram
func (gh *Histogram) Add(amount uint64, count uint64) {
	gh.m.Lock()
	gh.findBin(amount).incr(count)
	gh.m.Unlock()
}

// Set all bins to zero
func (gh *Histogram) Reset() {
	gh.m.Lock()
	for i := 0; i < len(gh._bins); i++ {
		gh._bins[i].set(0)
	}
	gh.m.Unlock()
}

// Gets the total number of samples counted
func (gh *Histogram) Total() uint64 {
	gh.m.Lock()
	var count uint64
	for i := 0; i < len(gh._bins); i++ {
		count += gh._bins[i]._count
	}
	gh.m.Unlock()
	return count
}

// AddAll adds all the Counts from the src histogram into this
// histogram.  The src and this histogram must have the same
// exact creation parameters.
func (gh *Histogram) AddAll(src *Histogram) {
	if len(gh._bins) != len(src._bins) {
		fmt.Errorf("Error: Bin-count mismatch: %d != %d",
			len(gh._bins), len(src._bins))
		return
	}

	src.m.Lock()
	gh.m.Lock()

	for i := 0; i < len(src._bins); i++ {
		if gh._bins[i]._start == src._bins[i]._start &&
			gh._bins[i]._end == src._bins[i]._end {
			gh._bins[i]._count += src._bins[i]._count
		}
	}
	copy(src._bins, gh._bins)

	gh.m.Unlock()
	src.m.Unlock()
}

// Graph emits an ascii graph to the optional out buffer, allocating a
// out buffer if none was supplied.  Returns the out buffer.  Each
// line emitted may have an optional prefix.
//
// For example:
//       0+  10=2 10.00% ********
//      10+  10=1 10.00% ****
//      20+  10=3 10.00% ************
func (gh *Histogram) EmitGraph(prefix []byte,
	out *bytes.Buffer) *bytes.Buffer {
	if out == nil {
		out = bytes.NewBuffer(make([]byte, 0, 80*len(gh._bins)))
	}

	barLen := float64(len(bar))

	var totalCount uint64
	var maxCount uint64
	var ranges []string
	var longestRange int

	gh.m.Lock()

	for i := 0; i < len(gh._bins); i++ {
		totalCount += gh._bins[i]._count
		if maxCount < gh._bins[i]._count {
			maxCount = gh._bins[i]._count
		}

		var temp string
		if gh._bins[i]._end != math.MaxUint64 {
			temp = fmt.Sprintf("%v - %v", gh._bins[i]._start, gh._bins[i]._end)
		} else {
			temp = fmt.Sprintf("%v - inf", gh._bins[i]._start)
		}
		ranges = append(ranges, temp)
		if gh._bins[i]._count > 0 && longestRange < len(temp) {
			longestRange = len(temp)
		}
	}

	fmt.Fprintf(out, "%s (%v Total)\n", gh._name, totalCount)
	for i := 0; i < len(gh._bins); i++ {
		binCount := gh._bins[i]._count
		if binCount == 0 {
			continue
		}

		var padding string
		for j := 0; j < (longestRange - len(ranges[i])); j++ {
			padding = padding + " "
		}

		if prefix != nil {
			out.Write(prefix)
		}

		fmt.Fprintf(out, "[%s] %s%10v %7.2f%%",
			ranges[i], padding, binCount, 100.0*(float64(binCount)/float64(totalCount)))

		out.Write([]byte(" "))
		barWant := int(math.Floor(barLen * (float64(binCount) / float64(maxCount))))
		out.Write(bar[0:barWant])

		out.Write([]byte("\n"))
	}

	gh.m.Unlock()

	return out
}

var bar = []byte("##############################")

// CallSync invokes the callback func while the histogram is locked.
func (gh *Histogram) CallSync(f func()) {
	gh.m.Lock()
	f()
	gh.m.Unlock()
}
