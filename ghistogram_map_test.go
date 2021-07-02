//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

// Package ghistogram provides a simple histogram of uint64's that
// avoids heap allocations (garbage creation) during data processing.

package ghistogram

import (
	"strings"
	"testing"
)

func initAndFetchHistograms(t *testing.T) (Histograms, string, string) {
	histograms := make(Histograms)
	histograms["test1"] = NewNamedHistogram("test1 (µs)", 10, 2, 2)
	histograms["test2"] = NewNamedHistogram("test2 (µs)", 10, 2, 2)

	histograms["test1"].Add(uint64(1), 2)
	histograms["test1"].Add(uint64(3), 4)
	histograms["test2"].Add(uint64(2), 1)
	histograms["test2"].Add(uint64(4), 3)

	test1 := `test1 (µs) (6 Total)
[0 - 2]   33.33%   33.33% ############### (2)
[2 - 4]   66.67%  100.00% ############################## (4)
`

	test2 := `test2 (µs) (4 Total)
[2 - 4]   25.00%   25.00% ########## (1)
[4 - 8]   75.00%  100.00% ############################## (3)
`

	return histograms, test1, test2
}

func TestStringHistograms(t *testing.T) {
	histograms, exp1, exp2 := initAndFetchHistograms(t)

	output := histograms.String()
	if !strings.Contains(output, exp1) || !strings.Contains(output, exp2) {
		t.Errorf("Unexpected content in String()")
	}
}

func TestAddAllHistograms(t *testing.T) {
	histograms, exp1, exp2 := initAndFetchHistograms(t)

	newhistograms := make(Histograms)
	newhistograms.AddAll(histograms)

	output := newhistograms.String()

	if !strings.Contains(output, exp1) || !strings.Contains(output, exp2) {
		t.Errorf("Unexpected content in String() after AddAll")
	}
}
