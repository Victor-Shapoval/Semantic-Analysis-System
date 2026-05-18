package features

type FeatureVector [FeatureCount]float32

// FeatureCount — feature-vector size (32: amplitudes + sin/cos + I2/U2 + R/X).
const FeatureCount = 32

const (
	// amplitudes in p.u.: [0..7]
	IdxAmpIa = 0
	IdxAmpIb = 1
	IdxAmpIc = 2
	IdxAmpI0 = 3
	IdxAmpUa = 4
	IdxAmpUb = 5
	IdxAmpUc = 6
	IdxAmpU0 = 7

	IdxSinIa = 8
	IdxCosIa = 9
	IdxSinIb = 10
	IdxCosIb = 11
	IdxSinIc = 12
	IdxCosIc = 13
	IdxSinI0 = 14
	IdxCosI0 = 15
	IdxSinUa = 16
	IdxCosUa = 17
	IdxSinUb = 18
	IdxCosUb = 19
	IdxSinUc = 20
	IdxCosUc = 21
	IdxSinU0 = 22
	IdxCosU0 = 23

	IdxI2 = 24
	IdxU2 = 25

	IdxRa = 26
	IdxXa = 27
	IdxRb = 28
	IdxXb = 29
	IdxRc = 30
	IdxXc = 31
)
