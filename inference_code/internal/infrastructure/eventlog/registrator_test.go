package eventlog_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"semantic-analysis-system/internal/domain/detection"
	domaineventlog "semantic-analysis-system/internal/domain/eventlog"
	"semantic-analysis-system/internal/domain/features"
	infraeventlog "semantic-analysis-system/internal/infrastructure/eventlog"
)

func TestSlogRegistrator_Register_WritesLog(t *testing.T) {
	// temporary file
	f, err := os.CreateTemp("", "eventlog_test_*.log")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	path := f.Name()
	f.Close()
	defer os.Remove(path)

	reg, err := infraeventlog.NewSlogRegistrator(path)
	if err != nil {
		t.Fatalf("NewSlogRegistrator: %v", err)
	}
	defer reg.Close()

	var vec features.FeatureVector
	vec[0] = 0.5 // amp_Ia
	vec[1] = 0.4 // amp_Ib

	event := domaineventlog.FaultEvent{
		Timestamp: time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC),
		WindowID:  7,
		Features:  vec,
		Result: detection.DetectionResult{
			Label:      detection.LabelAnomaly,
			Confidence: 0.93,
		},
	}

	if err := reg.Register(event); err != nil {
		t.Fatalf("Register: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}

	content := string(data)

	checks := []string{"fault_event", "Anomaly", "window_id=7"}
	for _, want := range checks {
		if !strings.Contains(content, want) {
			t.Errorf("log missing %q in output:\n%s", want, content)
		}
	}
}
