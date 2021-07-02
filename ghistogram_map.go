//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package ghistogram

import (
	"errors"
	"io"
	"strings"
)

// Histograms represents a map of histograms identified by
// unique names (string).
type Histograms map[string]*Histogram

// API that converts the contents of all histograms within
// the map into a string and returns the string to caller.
func (hmap Histograms) String() string {
	var output []string

	for _, v := range hmap {
		output = append(output, v.EmitGraph(nil, nil).String())
	}

	return strings.Join(output, "\n")
}

// Emits the ASCII graphs of all histograms held within
// the map through the provided writer.
func (hmap Histograms) Fprint(w io.Writer) (int, error) {
	wrote, err := w.Write([]byte(hmap.String()))
	return wrote, err
}

// Adds all entries/records from all histograms within the
// given map, to all histograms in the current map.
// If a histogram from the source doesn't exist in the
// destination map, it will be created first.
func (hmap Histograms) AddAll(srcmap Histograms) error {
	for k, v := range srcmap {
		if hmap[k] == nil {
			// Histogram entry not found, create a new one, based
			// on the same creation parameters
			hmap[k] = v.CloneEmpty()
		} else if (len(hmap[k].Counts) != len(v.Counts)) ||
			(len(hmap[k].Ranges) != len(v.Ranges)) {
			return errors.New("Mismatch in histogram creation parameters")
		} else {
			for i := 0; i < len(v.Ranges); i++ {
				if hmap[k].Ranges[i] != v.Ranges[i] {
					return errors.New("Mismatch in histogram creation parmeters")
				}
			}
		}
	}

	for k, v := range srcmap {
		hmap[k].AddAll(v)
	}

	return nil
}
