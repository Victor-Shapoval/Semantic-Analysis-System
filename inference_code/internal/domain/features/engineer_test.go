package features_test

import (
	"math"
	"testing"

	"semantic-analysis-system/internal/domain/features"
)

const tol = 1e-5

func balancedPhasors(ampI, ampU float64) [8]features.Phasor {
	return [8]features.Phasor{
		{Amplitude: ampI, Phase: 0},                // Ia
		{Amplitude: ampI, Phase: -2 * math.Pi / 3}, // Ib
		{Amplitude: ampI, Phase: 2 * math.Pi / 3},  // Ic
		{Amplitude: 0, Phase: 0},                   // I0
		{Amplitude: ampU, Phase: 0},                // Ua
		{Amplitude: ampU, Phase: -2 * math.Pi / 3}, // Ub
		{Amplitude: ampU, Phase: 2 * math.Pi / 3},  // Uc
		{Amplitude: 0, Phase: 0},                   // U0
	}
}

func TestEngineer_Amplitudes_InPU(t *testing.T) {
	const uNom, iNom = 63508.0, 166.0
	scaler := features.NewScaler(uNom, iNom)
	eng := features.NewEngineer(scaler)

	ampI := iNom * math.Sqrt2
	ampU := uNom * math.Sqrt2
	p := balancedPhasors(ampI, ampU)
	v := eng.Build(p)

	for _, tc := range []struct {
		name string
		idx  int
	}{
		{"amp_Ia", features.IdxAmpIa},
		{"amp_Ib", features.IdxAmpIb},
		{"amp_Ic", features.IdxAmpIc},
		{"amp_Ua", features.IdxAmpUa},
		{"amp_Ub", features.IdxAmpUb},
		{"amp_Uc", features.IdxAmpUc},
	} {
		if math.Abs(float64(v[tc.idx])-1.0) > tol {
			t.Errorf("%s: got %.6f, want 1.0", tc.name, v[tc.idx])
		}
	}
}

func TestEngineer_SinCos(t *testing.T) {
	scaler := features.NewScaler(63508, 166)
	eng := features.NewEngineer(scaler)

	p := balancedPhasors(100, 100)
	v := eng.Build(p)

	// Ia: phase=0 → sin=0, cos=1
	if math.Abs(float64(v[features.IdxSinIa])) > tol {
		t.Errorf("sin(Ia): got %.6f, want 0", v[features.IdxSinIa])
	}
	if math.Abs(float64(v[features.IdxCosIa])-1.0) > tol {
		t.Errorf("cos(Ia): got %.6f, want 1", v[features.IdxCosIa])
	}

	// Ib: phase=-120° → sin=-√3/2, cos=-0.5
	wantSin := math.Sin(-2 * math.Pi / 3)
	wantCos := math.Cos(-2 * math.Pi / 3)
	if math.Abs(float64(v[features.IdxSinIb])-wantSin) > tol {
		t.Errorf("sin(Ib): got %.6f, want %.6f", v[features.IdxSinIb], wantSin)
	}
	if math.Abs(float64(v[features.IdxCosIb])-wantCos) > tol {
		t.Errorf("cos(Ib): got %.6f, want %.6f", v[features.IdxCosIb], wantCos)
	}
}

func TestEngineer_NegativeSequence_BalancedSystem(t *testing.T) {
	scaler := features.NewScaler(63508, 166)
	eng := features.NewEngineer(scaler)

	p := balancedPhasors(166*math.Sqrt2, 63508*math.Sqrt2)
	v := eng.Build(p)

	if v[features.IdxI2] > float32(tol) {
		t.Errorf("I2 balanced: got %.6f, want ≈0", v[features.IdxI2])
	}
	if v[features.IdxU2] > float32(tol) {
		t.Errorf("U2 balanced: got %.6f, want ≈0", v[features.IdxU2])
	}
}

func TestEngineer_NegativeSequence_Unbalanced(t *testing.T) {
	scaler := features.NewScaler(63508, 166)
	eng := features.NewEngineer(scaler)

	p := balancedPhasors(166*math.Sqrt2, 63508*math.Sqrt2)
	p[0].Amplitude *= 2 // Ia is doubled
	v := eng.Build(p)

	if v[features.IdxI2] < float32(tol) {
		t.Errorf("I2 unbalanced: got %.6f, want > 0", v[features.IdxI2])
	}
}

func TestEngineer_Impedance_PureResistive(t *testing.T) {
	scaler := features.NewScaler(63508, 166)
	eng := features.NewEngineer(scaler)

	// Ia and Ua have the same phase → Z = |Ua|/|Ia|, X=0
	ampI := 166 * math.Sqrt2
	ampU := 63508 * math.Sqrt2
	p := balancedPhasors(ampI, ampU)
	v := eng.Build(p)

	// Ra = Re(Ua_pu / Ia_pu) = (ampU/ampU_nom) / (ampI/ampI_nom)
	wantRa := float32((ampU / (63508 * math.Sqrt2)) / (ampI / (166 * math.Sqrt2))) // = 1.0
	if math.Abs(float64(v[features.IdxRa]-wantRa)) > tol {
		t.Errorf("Ra: got %.6f, want %.6f", v[features.IdxRa], wantRa)
	}
	if math.Abs(float64(v[features.IdxXa])) > tol {
		t.Errorf("Xa pure resistive: got %.6f, want ≈0", v[features.IdxXa])
	}
}

//
// func TestEngineer_ImpedanceDerivatives_Stable(t *testing.T) { ... }
// func TestEngineer_Derivatives_FirstCallZero(t *testing.T) { ... }
// func TestEngineer_Derivatives_SecondCall(t *testing.T) { ... }
