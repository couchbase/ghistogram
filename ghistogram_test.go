package ghistogram

import (
	"bytes"
	"testing"
)

func TestNewHistogram(t *testing.T) {
	tests := []struct {
		numBins         int
		binFirst        uint64
		binGrowthFactor float64
		ent             []uint64
	}{
		{2, 123, 10.0, []uint64{0, 123}},
		{2, 123, 10.0, []uint64{0, 123}},

		// Test constant bin sizes.
		{5, 10, 0.0, []uint64{0, 10, 20, 30, 40}},

		// Test growing bin sizes.
		{5, 10, 1.5, []uint64{0, 10, 15, 23, 35}},
		{5, 10, 2.0, []uint64{0, 10, 20, 40, 80}},
		{5, 10, 10.0, []uint64{0, 10, 100, 1000, 10000}},
	}

	for testi, test := range tests {
		gh := NewHistogram(test.numBins, test.binFirst, test.binGrowthFactor)
		for i := 0; i < len(test.ent); i++ {
			gh.Add(test.ent[i], 1)
		}

		if gh._name != "Histogram" {
			t.Errorf("test #%d: Incorrect name of histogram!", testi)
		}

		if gh.Total() != uint64(len(test.ent)) {
			t.Errorf("test #%d: Incorrect total count", testi)
		}

		if len(gh._bins) != test.numBins+2 {
			t.Errorf("test #%d: Incorrect number of bins", testi)
		}
	}
}

func TestAdd(t *testing.T) {
	// Bins will look like: {0-10, 10-20, 20-40, 40-80, 80-160, 160-320, 320-inf}
	gh := NewHistogram(5, 10, 2.0)

	tests := []struct {
		val uint64
		exp []uint64
	}{
		{0, []uint64{1, 0, 0, 0, 0, 0, 0}},
		{0, []uint64{2, 0, 0, 0, 0, 0, 0}},
		{0, []uint64{3, 0, 0, 0, 0, 0, 0}},

		{2, []uint64{4, 0, 0, 0, 0, 0, 0}},
		{3, []uint64{5, 0, 0, 0, 0, 0, 0}},
		{4, []uint64{6, 0, 0, 0, 0, 0, 0}},

		{10, []uint64{6, 1, 0, 0, 0, 0, 0}},
		{11, []uint64{6, 2, 0, 0, 0, 0, 0}},
		{12, []uint64{6, 3, 0, 0, 0, 0, 0}},

		{100, []uint64{6, 3, 0, 0, 1, 0, 0}},
		{90, []uint64{6, 3, 0, 0, 2, 0, 0}},
		{80, []uint64{6, 3, 0, 0, 3, 0, 0}},

		{20, []uint64{6, 3, 1, 0, 3, 0, 0}},
		{30, []uint64{6, 3, 2, 0, 3, 0, 0}},
		{40, []uint64{6, 3, 2, 1, 3, 0, 0}},
	}

	for testi, test := range tests {
		gh.Add(test.val, 1)

		for i := 0; i < len(gh._bins); i++ {
			if gh._bins[i]._count != test.exp[i] {
				t.Errorf("test #%d, actual (%v) != exp (%v)",
					testi, gh._bins[i]._count, test.exp)
			}
		}

		if gh.Total() != uint64(testi+1) {
			t.Errorf("TotCounts wrong")
		}
	}
}

func TestAddAll(t *testing.T) {
	// Bins will look like: {0-10, 10-20, 20-40, 40-80, 80-160, 160-320, 320-inf}
	gh := NewHistogram(5, 10, 2.0)

	gh.Add(15, 2)
	gh.Add(25, 3)
	gh.Add(1000, 1)

	gh2 := NewHistogram(5, 10, 2.0)
	gh2.AddAll(gh)
	gh2.AddAll(gh)

	exp := []uint64{0, 4, 6, 0, 0, 0, 2}

	for i := 0; i < len(gh2._bins); i++ {
		if gh2._bins[i]._count != exp[i] {
			t.Errorf("AddAll mismatch, actual (%v) != exp (%v)",
				gh2._bins[i]._count, exp)
		}
	}

	if gh2.Total() != 12 {
		t.Errorf("TotCount wrong")
	}
}

func TestGraph(t *testing.T) {
	// Bins will look like: {[0 - 10], [10 - 20], [20 - 40], [40 - 80], [80 - 160],
	//                       [160 - 320], [320 - 640], [640 - 1280], [1280 - inf]
	gh := NewNamedHistogram("TestGraph", 7, 10, 2.0)

	gh.Add(5, 2)
	gh.Add(10, 20)
	gh.Add(20, 10)
	gh.Add(40, 3)
	gh.Add(160, 2)
	gh.Add(320, 1)
	gh.Add(1280, 10)

	buf := gh.EmitGraph([]byte("- "), nil)

	exp := `TestGraph (48 Total)
- [0 - 10]        4.17%    4.17% ### (2)
- [10 - 20]      41.67%   45.83% ############################## (20)
- [20 - 40]      20.83%   66.67% ############### (10)
- [40 - 80]       6.25%   72.92% #### (3)
- [160 - 320]     4.17%   77.08% ### (2)
- [320 - 640]     2.08%   79.17% # (1)
- [1280 - inf]   20.83%  100.00% ############### (10)
`

	got := buf.String()
	if got != exp {
		t.Errorf("didn't get expected graph,\ngot: %s\nexp: %s",
			got, exp)
	}
}

func BenchmarkAdd_100_10_0p0(b *testing.B) {
	benchmarkAdd(b, 100, 10, 0.0)
}

func BenchmarkAdd_100_10_1p5(b *testing.B) {
	benchmarkAdd(b, 100, 10, 1.5)
}

func BenchmarkAdd_100_10_2p0(b *testing.B) {
	benchmarkAdd(b, 100, 10, 2.0)
}

func BenchmarkAdd_1000_10_0p0(b *testing.B) {
	benchmarkAdd(b, 1000, 10, 0.0)
}

func BenchmarkAdd_1000_10_1p5(b *testing.B) {
	benchmarkAdd(b, 1000, 10, 1.5)
}

func BenchmarkAdd_1000_10_2p0(b *testing.B) {
	benchmarkAdd(b, 1000, 10, 2.0)
}

func benchmarkAdd(b *testing.B,
	numBins int,
	binFirst uint64,
	binGrowthFactor float64) {
	gh := NewHistogram(numBins, binFirst, binGrowthFactor)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		gh.Add(uint64(i), 1)
	}
}

func BenchmarkEmitGraph(b *testing.B) {
	benchmarkEmitGraph(b, 100, 10, 2.0)
}

func benchmarkEmitGraph(b *testing.B,
	numBins int,
	binFirst uint64,
	binGrowthFactor float64) {
	gh := NewHistogram(numBins, binFirst, binGrowthFactor)
	for i := 0; i < b.N/1000; i++ {
		gh.Add(uint64(i), 1)
	}

	buf := bytes.NewBuffer(make([]byte, 0, 20000))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		gh.EmitGraph(nil, buf)

		buf.Reset()
	}
}
