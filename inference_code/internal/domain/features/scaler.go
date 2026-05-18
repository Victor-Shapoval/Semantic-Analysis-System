package features

import "math"

type Scaler struct {
	uNomAmp float64 // U_nom * sqrt(2)
	iNomAmp float64 // I_nom * sqrt(2)
}

func NewScaler(uNom, iNom float64) *Scaler {
	return &Scaler{
		uNomAmp: uNom * math.Sqrt2,
		iNomAmp: iNom * math.Sqrt2,
	}
}

// ScaleCurrent normalizes current amplitude to p.u.
func (s *Scaler) ScaleCurrent(amp float64) float64 {
	return amp / s.iNomAmp
}

func (s *Scaler) ScaleVoltage(amp float64) float64 {
	return amp / s.uNomAmp
}
