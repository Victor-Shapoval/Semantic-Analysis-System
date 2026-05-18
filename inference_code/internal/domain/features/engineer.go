package features

import (
	"math"
	"math/cmplx"
)

const eps = 1e-6 // division-by-zero guard

// Engineer builds a 40-feature vector from phasors of one time step.
// It implements the full pipeline from 2_features_create.ipynb:
// amplitude normalization to p.u., sin/cos angle encoding,
// negative-sequence current and voltage I2/U2,
// and current, voltage, and impedance derivatives relative to the previous step.
// Absolute R and X values are not included in the vector; only their derivatives are included.
type Engineer struct {
	scaler *Scaler
	prev   *FeatureVector // previous vector for current and voltage derivatives
	prevRX [6]float32     // R_a, X_a, R_b, X_b, R_c, X_c from the previous step for dR/dX
}

// NewEngineer creates an Engineer with the given scaler.
func NewEngineer(scaler *Scaler) *Engineer {
	return &Engineer{scaler: scaler}
}

// phasors8 — phasors for 8 channels: [Ia, Ib, Ic, I0, Ua, Ub, Uc, U0].
type phasors8 [8]Phasor

// Build assembles a FeatureVector from 8 phasors of the current step.
func (e *Engineer) Build(p phasors8) FeatureVector {
	var v FeatureVector

	// amplitudes in p.u.
	v[IdxAmpIa] = float32(e.scaler.ScaleCurrent(p[0].Amplitude))
	v[IdxAmpIb] = float32(e.scaler.ScaleCurrent(p[1].Amplitude))
	v[IdxAmpIc] = float32(e.scaler.ScaleCurrent(p[2].Amplitude))
	v[IdxAmpI0] = float32(e.scaler.ScaleCurrent(p[3].Amplitude))
	v[IdxAmpUa] = float32(e.scaler.ScaleVoltage(p[4].Amplitude))
	v[IdxAmpUb] = float32(e.scaler.ScaleVoltage(p[5].Amplitude))
	v[IdxAmpUc] = float32(e.scaler.ScaleVoltage(p[6].Amplitude))
	v[IdxAmpU0] = float32(e.scaler.ScaleVoltage(p[7].Amplitude))

	// Angle sin/cos values alternate in pairs: sin_Ia, cos_Ia, sin_Ib, cos_Ib, ...
	sinIdx := [8]int{IdxSinIa, IdxSinIb, IdxSinIc, IdxSinI0, IdxSinUa, IdxSinUb, IdxSinUc, IdxSinU0}
	cosIdx := [8]int{IdxCosIa, IdxCosIb, IdxCosIc, IdxCosI0, IdxCosUa, IdxCosUb, IdxCosUc, IdxCosU0}
	for i := 0; i < 8; i++ {
		v[sinIdx[i]] = float32(math.Sin(p[i].Phase))
		v[cosIdx[i]] = float32(math.Cos(p[i].Phase))
	}

	// Negative-sequence current I2 = (1/3)|Ia + a^2*Ib + a*Ic|.
	// a = exp(j·2π/3)
	a := cmplx.Exp(complex(0, 2*math.Pi/3))
	a2 := a * a

	Ia := complex(float64(v[IdxAmpIa]), 0) * cmplx.Exp(complex(0, p[0].Phase))
	Ib := complex(float64(v[IdxAmpIb]), 0) * cmplx.Exp(complex(0, p[1].Phase))
	Ic := complex(float64(v[IdxAmpIc]), 0) * cmplx.Exp(complex(0, p[2].Phase))
	v[IdxI2] = float32(cmplx.Abs(Ia+a2*Ib+a*Ic) / 3.0)

	Ua := complex(float64(v[IdxAmpUa]), 0) * cmplx.Exp(complex(0, p[4].Phase))
	Ub := complex(float64(v[IdxAmpUb]), 0) * cmplx.Exp(complex(0, p[5].Phase))
	Uc := complex(float64(v[IdxAmpUc]), 0) * cmplx.Exp(complex(0, p[6].Phase))
	v[IdxU2] = float32(cmplx.Abs(Ua+a2*Ub+a*Uc) / 3.0)

	// Impedance Z = U/I gives R=Re(Z), X=Im(Z) in p.u. for phases a, b, c.
	phaseIU := [3]struct{ I, U complex128 }{{Ia, Ua}, {Ib, Ub}, {Ic, Uc}}
	idxR := [3]int{IdxRa, IdxRb, IdxRc}
	idxX := [3]int{IdxXa, IdxXb, IdxXc}
	for i, ph := range phaseIU {
		denom := ph.I
		if cmplx.Abs(denom) < eps {
			denom = complex(eps, 0)
		}
		Z := ph.U / denom
		v[idxR[i]] = float32(real(Z))
		v[idxX[i]] = float32(imag(Z))
	}

	return v
}
