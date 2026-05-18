package detection

type Label uint8

const (
	LabelNormal  Label = 0
	LabelAnomaly Label = 1
)

func (l Label) String() string {
	if l == LabelAnomaly {
		return "Anomaly"
	}
	return "Normal"
}

type DetectionResult struct {
	Label      Label
	Confidence float32
}
