package pipeline_test

import (
	"testing"

	"semantic-analysis-system/internal/application/pipeline"
	"semantic-analysis-system/internal/domain/detection"
	"semantic-analysis-system/internal/domain/eventlog"
	"semantic-analysis-system/internal/domain/features"
	domaingoose "semantic-analysis-system/internal/domain/goose"
	"semantic-analysis-system/internal/domain/sv"
)

// --- mocks ---

type stubDetector struct {
	label detection.Label
}

func (d *stubDetector) Detect(_ []features.FeatureVector) (detection.DetectionResult, error) {
	return detection.DetectionResult{Label: d.label, Confidence: 0.9}, nil
}

type spyRegistrator struct {
	events []eventlog.FaultEvent
}

func (r *spyRegistrator) Register(e eventlog.FaultEvent) error {
	r.events = append(r.events, e)
	return nil
}

type spyPublisher struct {
	messages []domaingoose.Message
}

func (p *spyPublisher) Publish(msg domaingoose.Message) error {
	p.messages = append(p.messages, msg)
	return nil
}

// --- helpers ---

func newTestService(det *stubDetector, reg *spyRegistrator, pub *spyPublisher, debounce int) *pipeline.Service {
	scaler := features.NewScaler(63508, 166)
	return pipeline.NewService(
		scaler, det, reg, pub,
		"test/cbRef", "testGoID",
		"",   // display off
		4000, // sps
		50,   // frequency
		debounce,
	)
}

func makeSVFrame() *sv.SVFrame {
	return &sv.SVFrame{}
}

// samplesPerPeriod=80, WindowSize=80, WindowStep=20
func feedFrames(svc *pipeline.Service, n int) {
	for i := 0; i < n; i++ {
		svc.Process(makeSVFrame())
	}
}

func TestDebounce_NoPublishBeforeThreshold(t *testing.T) {
	det := &stubDetector{label: detection.LabelAnomaly}
	reg := &spyRegistrator{}
	pub := &spyPublisher{}
	svc := newTestService(det, reg, pub, 3)

	feedFrames(svc, 160)

	// 1 window < debounce=3 → GOOSE must not be sent
	if len(pub.messages) != 0 {
		t.Errorf("expected 0 GOOSE messages before debounce, got %d", len(pub.messages))
	}
}

func TestDebounce_PublishAfterThreshold(t *testing.T) {
	det := &stubDetector{label: detection.LabelAnomaly}
	reg := &spyRegistrator{}
	pub := &spyPublisher{}
	svc := newTestService(det, reg, pub, 3)

	feedFrames(svc, 200)

	if len(pub.messages) != 1 {
		t.Fatalf("expected 1 GOOSE message after debounce, got %d", len(pub.messages))
	}
	if !pub.messages[0].Trip {
		t.Error("expected Trip=true")
	}
}

func TestDebounce_ResetOnLabelChange(t *testing.T) {
	det := &stubDetector{label: detection.LabelAnomaly}
	reg := &spyRegistrator{}
	pub := &spyPublisher{}
	svc := newTestService(det, reg, pub, 3)

	feedFrames(svc, 180)

	det.label = detection.LabelNormal
	feedFrames(svc, 20)

	if len(pub.messages) != 0 {
		t.Errorf("expected 0 GOOSE messages after label change, got %d", len(pub.messages))
	}
}

func TestDebounce_TripAndRecover(t *testing.T) {
	det := &stubDetector{label: detection.LabelAnomaly}
	reg := &spyRegistrator{}
	pub := &spyPublisher{}
	svc := newTestService(det, reg, pub, 2)

	feedFrames(svc, 180)

	if len(pub.messages) != 1 || !pub.messages[0].Trip {
		t.Fatalf("expected Trip=true after 2 anomaly windows, got %d messages", len(pub.messages))
	}

	det.label = detection.LabelNormal
	feedFrames(svc, 40)

	if len(pub.messages) != 2 {
		t.Fatalf("expected 2 GOOSE messages (trip + recover), got %d", len(pub.messages))
	}
	if pub.messages[1].Trip {
		t.Error("expected Trip=false on recovery")
	}
}

func TestRegistrator_OnlyOnAnomaly(t *testing.T) {
	det := &stubDetector{label: detection.LabelNormal}
	reg := &spyRegistrator{}
	pub := &spyPublisher{}
	svc := newTestService(det, reg, pub, 1)

	// 3 Normal windows
	feedFrames(svc, 200)

	if len(reg.events) != 0 {
		t.Errorf("expected 0 fault events for Normal, got %d", len(reg.events))
	}

	// switch to Anomaly, 1 window
	det.label = detection.LabelAnomaly
	feedFrames(svc, 20)

	if len(reg.events) != 1 {
		t.Errorf("expected 1 fault event for Anomaly, got %d", len(reg.events))
	}
}
