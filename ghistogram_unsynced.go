//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package ghistogram

// HistogramMutator represents the subset of Histogram methods related
// to mutation operations.
type HistogramMutator interface {
	Add(dataPoint uint64, count uint64)
}

// histogramMutator implements the HistogramMutator interface for a
// given Histogram.
type histogramMutator struct {
	*Histogram // An anonymous field of type Histogram
}

// Add increases the count in the histogram bin for the given dataPoint.
func (h *histogramMutator) Add(dataPoint uint64, count uint64) {
	h.addUNLOCKED(dataPoint, count)
}
