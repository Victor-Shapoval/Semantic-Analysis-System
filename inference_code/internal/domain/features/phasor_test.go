package features_test

import (
	"math"
	"testing"

	"semantic-analysis-system/internal/domain/features"
)

// TestDFTExtractor_Sinusoid checks that DFT correctly determines
func TestDFTExtractor_Sinusoid(t *testing.T) {
	const (
		N         = 80
		wantAmp   = 1.0         // amplitude (peak value)
		wantPhase = math.Pi / 4 // phase 45°
		tol       = 1e-9
	)

	dft := features.NewDFTExtractor(N, 1)

	// generate exactly one period: x(n) = A·cos(2π·n/N + φ)
	samples := make([]float64, N)
	for n := 0; n < N; n++ {
		samples[n] = wantAmp * math.Cos(2*math.Pi*float64(n)/float64(N)+wantPhase)
	}

	phasor := dft.Extract(samples)

	if math.Abs(phasor.Amplitude-wantAmp) > tol {
		t.Errorf("Amplitude: got %.10f, want %.10f (tol %.0e)", phasor.Amplitude, wantAmp, tol)
	}
	if math.Abs(phasor.Phase-wantPhase) > tol {
		t.Errorf("Phase: got %.10f rad, want %.10f rad (tol %.0e)", phasor.Phase, wantPhase, tol)
	}
}

func TestDFTExtractor_WrongSize(t *testing.T) {
	dft := features.NewDFTExtractor(80, 1)
	phasor := dft.Extract([]float64{1.0, 2.0}) // size is not 80

	if phasor.Amplitude != 0 || phasor.Phase != 0 {
		t.Errorf("expected zero phasor for wrong size input, got amp=%.4f phase=%.4f",
			phasor.Amplitude, phasor.Phase)
	}
}

func TestDFTExtractor_ZeroSignal(t *testing.T) {
	dft := features.NewDFTExtractor(80, 1)
	phasor := dft.Extract(make([]float64, 80))

	if phasor.Amplitude > 1e-15 {
		t.Errorf("expected zero amplitude for zero signal, got %.6e", phasor.Amplitude)
	}
}
