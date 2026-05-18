package onnx_test

import (
	"math"
	"path/filepath"
	"runtime"
	"testing"

	"semantic-analysis-system/internal/domain/detection"
	"semantic-analysis-system/internal/domain/features"
	infraonnx "semantic-analysis-system/internal/infrastructure/onnx"
)

func modelPath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location")
	}
	// onnx/ → infrastructure/ → internal/ → semantic_analysis_system/ → models/
	root := filepath.Join(filepath.Dir(file), "..", "..", "..")
	return filepath.Join(root, "models", "model_gru.onnx")
}

func TestDetector_Inference(t *testing.T) {
	d, err := infraonnx.NewDetector(modelPath(t), 0.5)
	if err != nil {
		t.Skipf("ONNX model not available (%v) — skipping golden test", err)
	}
	defer d.Close()

	t.Run("ZeroInput", func(t *testing.T) {
		window := make([]features.FeatureVector, features.WindowSize)
		result, err := d.Detect(window)
		if err != nil {
			t.Fatalf("Detect() error: %v", err)
		}
		if result.Confidence < 0 || result.Confidence > 1 {
			t.Errorf("Confidence out of [0,1]: %.4f", result.Confidence)
		}
		if result.Label != detection.LabelNormal && result.Label != detection.LabelAnomaly {
			t.Errorf("unexpected label: %v", result.Label)
		}
		t.Logf("zero input → label=%s confidence=%.4f", result.Label, result.Confidence)
	})

	t.Run("NominalInput", func(t *testing.T) {
		window := make([]features.FeatureVector, features.WindowSize)
		for i := range window {
			v := &window[i]
			v[features.IdxAmpIa] = 0.01
			v[features.IdxAmpIb] = 0.01
			v[features.IdxAmpIc] = 0.01
			v[features.IdxAmpUa] = float32(1.0 / math.Sqrt2)
			v[features.IdxAmpUb] = float32(1.0 / math.Sqrt2)
			v[features.IdxAmpUc] = float32(1.0 / math.Sqrt2)
			v[features.IdxCosIa] = 1.0
			v[features.IdxCosIb] = 1.0
			v[features.IdxCosIc] = 1.0
			v[features.IdxCosUa] = 1.0
			v[features.IdxCosUb] = 1.0
			v[features.IdxCosUc] = 1.0
		}
		result, err := d.Detect(window)
		if err != nil {
			t.Fatalf("Detect() error: %v", err)
		}
		t.Logf("nominal input → label=%s confidence=%.4f", result.Label, result.Confidence)
	})
}
