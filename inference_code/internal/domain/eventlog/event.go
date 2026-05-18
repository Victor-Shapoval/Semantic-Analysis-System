package eventlog

import (
	"time"

	"semantic-analysis-system/internal/domain/detection"
	"semantic-analysis-system/internal/domain/features"
)

type FaultEvent struct {
	Timestamp time.Time `json:"timestamp"`
	WindowID  uint64    `json:"window_id"`

	Features features.FeatureVector `json:"features"`

	// Result is the model verdict
	Result detection.DetectionResult `json:"result"`
}
