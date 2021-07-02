//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package ghistogram

import (
	"strings"
	"testing"
)

func TestUnsyncedAdd(t *testing.T) {
	hists := make(Histograms)
	hists["hist1"] = NewNamedHistogram("hist1", 10, 4, 4)
	hists["hist2"] = NewNamedHistogram("hist2", 10, 4, 4)

	F1 := func(hist HistogramMutator) {
		for i := 0; i < 10000; i++ {
			hist.Add(uint64(i%100), 1)
		}
	}

	F2 := func(hist HistogramMutator) {
		for i := 0; i < 10000; i++ {
			hist.Add(uint64(i%1000), 1)
		}
	}

	hists["hist1"].CallSyncEx(F1)
	hists["hist2"].CallSyncEx(F2)

	hist1 := `hist1 (10000 Total)
[0 - 4]       4.00%    4.00% ## (400)
[4 - 16]     12.00%   16.00% ####### (1200)
[16 - 64]    48.00%   64.00% ############################## (4800)
[64 - 256]   36.00%  100.00% ###################### (3600)
`

	hist2 := `hist2 (10000 Total)
[0 - 4]         0.40%    0.40%  (40)
[4 - 16]        1.20%    1.60%  (120)
[16 - 64]       4.80%    6.40% # (480)
[64 - 256]     19.20%   25.60% ####### (1920)
[256 - 1024]   74.40%  100.00% ############################## (7440)
`

	if !strings.Contains(hists.String(), hist1) ||
		!strings.Contains(hists.String(), hist2) {
		t.Errorf("Unexpected content in histograms!")
	}
}
