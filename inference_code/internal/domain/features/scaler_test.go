package features_test

import (
	"math"
	"testing"

	"semantic-analysis-system/internal/domain/features"
)

func TestScaler_Current_RoundTrip(t *testing.T) {
	const uNom = 132790.0
	const iNom = 248.0

	s := features.NewScaler(uNom, iNom)

	wantPU := 1.0 / (iNom * math.Sqrt2)
	gotPU := s.ScaleCurrent(1.0)

	if math.Abs(gotPU-wantPU) > 1e-12 {
		t.Errorf("ScaleCurrent(1.0): got %.10f, want %.10f", gotPU, wantPU)
	}
}

func TestScaler_Voltage_RoundTrip(t *testing.T) {
	const uNom = 132790.0
	const iNom = 248.0

	s := features.NewScaler(uNom, iNom)

	wantPU := 1.0 / math.Sqrt2
	gotPU := s.ScaleVoltage(uNom)

	if math.Abs(gotPU-wantPU) > 1e-12 {
		t.Errorf("ScaleVoltage(U_nom): got %.10f, want %.10f", gotPU, wantPU)
	}
}

func TestScaler_NominalAmplitude_IsOne(t *testing.T) {
	const uNom = 132790.0
	const iNom = 248.0

	s := features.NewScaler(uNom, iNom)

	gotPU := s.ScaleCurrent(iNom * math.Sqrt2)
	if math.Abs(gotPU-1.0) > 1e-12 {
		t.Errorf("ScaleCurrent(I_nom·√2) should be 1.0, got %.10f", gotPU)
	}

	gotPU = s.ScaleVoltage(uNom * math.Sqrt2)
	if math.Abs(gotPU-1.0) > 1e-12 {
		t.Errorf("ScaleVoltage(U_nom·√2) should be 1.0, got %.10f", gotPU)
	}
}
