package detection

import "semantic-analysis-system/internal/domain/features"

type Detector interface {
	Detect(window []features.FeatureVector) (DetectionResult, error)
}
