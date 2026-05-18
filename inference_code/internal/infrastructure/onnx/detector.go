package onnx

import (
	"fmt"
	"os"
	"path/filepath"

	"semantic-analysis-system/internal/domain/detection"
	"semantic-analysis-system/internal/domain/features"

	ort "github.com/yalue/onnxruntime_go"
)

type Detector struct {
	session      *ort.AdvancedSession
	inputTensor  *ort.Tensor[float32]
	outputTensor *ort.Tensor[float32]
	threshold    float32
}

func NewDetector(modelPath string, threshold float64, libPath ...string) (*Detector, error) {
	// onnxruntime_go looks for "onnxruntime.so" by default;
	// on macOS the .dylib path must be specified explicitly
	if len(libPath) > 0 && libPath[0] != "" {
		ort.SetSharedLibraryPath(libPath[0])
	} else {
		for _, p := range []string{
			"/usr/local/lib/libonnxruntime.dylib",    // macOS Intel
			"/opt/homebrew/lib/libonnxruntime.dylib", // macOS Apple Silicon
			"/usr/local/lib/libonnxruntime.so",       // Linux
			"/usr/lib/libonnxruntime.so",             // Linux (system)
		} {
			if _, err := os.Stat(p); err == nil {
				ort.SetSharedLibraryPath(p)
				break
			}
		}
	}

	if err := ort.InitializeEnvironment(); err != nil {
		return nil, fmt.Errorf("onnx: init env: %w", err)
	}

	inputShape := ort.NewShape(1, int64(features.WindowSize), int64(features.FeatureCount))
	inputTensor, err := ort.NewEmptyTensor[float32](inputShape)
	if err != nil {
		return nil, fmt.Errorf("onnx: input tensor: %w", err)
	}

	outputShape := ort.NewShape(1)
	outputTensor, err := ort.NewEmptyTensor[float32](outputShape)
	if err != nil {
		inputTensor.Destroy()
		return nil, fmt.Errorf("onnx: output tensor: %w", err)
	}

	prevDir, _ := os.Getwd()
	if absModel, err := filepath.Abs(modelPath); err == nil {
		_ = os.Chdir(filepath.Dir(absModel))
		modelPath = filepath.Base(absModel)
	}

	session, err := ort.NewAdvancedSession(modelPath,
		[]string{"input"},
		[]string{"output"},
		[]ort.ArbitraryTensor{inputTensor},
		[]ort.ArbitraryTensor{outputTensor},
		nil,
	)
	if prevDir != "" {
		_ = os.Chdir(prevDir)
	}
	if err != nil {
		inputTensor.Destroy()
		outputTensor.Destroy()
		return nil, fmt.Errorf("onnx: create session: %w", err)
	}

	return &Detector{
		session:      session,
		inputTensor:  inputTensor,
		outputTensor: outputTensor,
		threshold:    float32(threshold),
	}, nil
}

// Detect runs a window of WindowSize vectors through the GRU model.
func (d *Detector) Detect(window []features.FeatureVector) (detection.DetectionResult, error) {
	if len(window) != features.WindowSize {
		return detection.DetectionResult{}, fmt.Errorf("onnx: window size %d != %d", len(window), features.WindowSize)
	}

	data := d.inputTensor.GetData()
	for step, vec := range window {
		for feat := 0; feat < features.FeatureCount; feat++ {
			data[step*features.FeatureCount+feat] = vec[feat]
		}
	}

	if err := d.session.Run(); err != nil {
		return detection.DetectionResult{}, fmt.Errorf("onnx: run: %w", err)
	}

	probs := d.outputTensor.GetData()
	pAnomaly := probs[0]

	label := detection.LabelNormal
	if pAnomaly >= d.threshold {
		label = detection.LabelAnomaly
	}

	return detection.DetectionResult{
		Label:      label,
		Confidence: pAnomaly,
	}, nil
}

func (d *Detector) Close() {
	d.session.Destroy()
	d.inputTensor.Destroy()
	d.outputTensor.Destroy()
}
