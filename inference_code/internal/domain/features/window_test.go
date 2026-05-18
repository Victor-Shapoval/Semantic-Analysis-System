package features_test

import (
	"testing"

	"semantic-analysis-system/internal/domain/features"
)

func vecWithMarker(marker float32) features.FeatureVector {
	var v features.FeatureVector
	v[0] = marker
	return v
}

func TestWindow_FirstWindowAfterSizeElements(t *testing.T) {
	w := features.NewWindow(4, 2)

	for i := 0; i < 3; i++ {
		_, ready := w.Push(vecWithMarker(float32(i)))
		if ready {
			t.Fatalf("window ready too early at push %d", i)
		}
	}

	out, ready := w.Push(vecWithMarker(3))
	if !ready {
		t.Fatal("window should be ready after size elements")
	}
	if len(out) != 4 {
		t.Fatalf("window size: got %d, want 4", len(out))
	}

	for i := 0; i < 4; i++ {
		if out[i][0] != float32(i) {
			t.Errorf("out[%d][0]: got %.0f, want %d", i, out[i][0], i)
		}
	}
}

func TestWindow_StepBetweenWindows(t *testing.T) {
	w := features.NewWindow(4, 2)

	for i := 0; i < 4; i++ {
		w.Push(vecWithMarker(float32(i)))
	}

	_, ready := w.Push(vecWithMarker(4))
	if ready {
		t.Fatal("window should not be ready after 1 step")
	}

	out, ready := w.Push(vecWithMarker(5))
	if !ready {
		t.Fatal("window should be ready after step elements")
	}

	expected := []float32{2, 3, 4, 5}
	for i, want := range expected {
		if out[i][0] != want {
			t.Errorf("out[%d][0]: got %.0f, want %.0f", i, out[i][0], want)
		}
	}
}

func TestWindow_ChronologicalOrderAfterWrap(t *testing.T) {
	w := features.NewWindow(3, 1)

	for i := 0; i < 3; i++ {
		w.Push(vecWithMarker(float32(i)))
	}

	out, ready := w.Push(vecWithMarker(3))
	if !ready {
		t.Fatal("expected window ready")
	}
	expected := []float32{1, 2, 3}
	for i, want := range expected {
		if out[i][0] != want {
			t.Errorf("out[%d][0]: got %.0f, want %.0f", i, out[i][0], want)
		}
	}
}

func TestWindow_ReturnsCopy(t *testing.T) {
	w := features.NewWindow(2, 1)

	w.Push(vecWithMarker(1))
	out1, _ := w.Push(vecWithMarker(2))

	out1[0][0] = 999

	out2, _ := w.Push(vecWithMarker(3))
	if out2[0][0] == 999 {
		t.Error("Push returns reference to internal buffer, not a copy")
	}
}

func TestWindow_RealParameters(t *testing.T) {
	w := features.NewWindow(features.WindowSize, features.WindowStep)

	readyCount := 0
	total := features.WindowSize + features.WindowStep*3

	for i := 0; i < total; i++ {
		_, ready := w.Push(vecWithMarker(float32(i)))
		if ready {
			readyCount++
		}
	}

	want := 4
	if readyCount != want {
		t.Errorf("ready count: got %d, want %d", readyCount, want)
	}
}
