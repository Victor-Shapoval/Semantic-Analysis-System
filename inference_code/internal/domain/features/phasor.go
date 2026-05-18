package features

import (
	"math"
	"math/cmplx"
)

type Phasor struct {
	Amplitude float64
	Phase     float64
}

type DFTExtractor struct {
	N int
	k int
	w complex128
}

// for 9-2LE 4000 sps, 50 Hz: N=80, k=1.
func NewDFTExtractor(N, k int) *DFTExtractor {
	angle := -2 * math.Pi * float64(k) / float64(N)
	return &DFTExtractor{
		N: N,
		k: k,
		w: cmplx.Exp(complex(0, angle)),
	}
}

// returns amplitude (peak value) and phase angle.
func (d *DFTExtractor) Extract(samples []float64) Phasor {
	if len(samples) != d.N {
		// fallback: zero phasor
		return Phasor{}
	}

	var sum complex128
	wk := complex(1, 0)
	for _, s := range samples {
		sum += complex(s, 0) * wk
		wk *= d.w
	}

	// normalize: divide by N/2 to obtain amplitude (not RMS)
	coeff := sum * complex(2.0/float64(d.N), 0)
	return Phasor{
		Amplitude: cmplx.Abs(coeff),
		Phase:     cmplx.Phase(coeff),
	}
}
