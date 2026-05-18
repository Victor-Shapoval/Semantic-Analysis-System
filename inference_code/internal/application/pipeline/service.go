package pipeline

import (
	"fmt"
	"log/slog"
	"math"
	"time"

	"semantic-analysis-system/internal/domain/detection"
	"semantic-analysis-system/internal/domain/eventlog"
	"semantic-analysis-system/internal/domain/features"
	domaingoose "semantic-analysis-system/internal/domain/goose"
	"semantic-analysis-system/internal/domain/sv"
)

// displayLines is the number of live display lines: 4 currents, a blank line, and 4 voltages.
const displayLines = 9

// displayRefreshHz is the live display refresh rate.
const displayRefreshHz = 10

// Service orchestrates the inference pipeline.
// SV frame -> DFT -> normalization -> features -> sliding window -> GRU -> EventLog + GOOSE.
type Service struct {
	dft         [8]*features.DFTExtractor // one extractor for each of the 8 channels
	ringBufs    [8][]float64              // ring buffers of instantaneous values for DFT (N)
	ringPos     int                       // current write position in the ring buffers
	ringFull    bool                      // true when the ring buffers contain at least N samples
	engineer    *features.Engineer
	window      *features.Window
	detector    detection.Detector
	registrator eventlog.Registrator
	goosePub    domaingoose.Publisher

	samplesPerPeriod int // N = sps / frequency (DFT buffer size)
	displayEvery     int // refresh the display every N frames

	windowID       uint64
	prevTrip       bool
	stNum          uint32
	debounce       int  // consecutive windows required to change state
	pendingTrip    bool // target state currently being confirmed
	consecutiveCnt int  // number of consecutive windows confirming pendingTrip
	sampleCount    uint64
	dftSampleCnt   int                // sample counter for DFTStep
	displayInit    bool               // true after the first display output
	lastPhasors    [8]features.Phasor // latest calculated phasors
	displayMode    string             // "rms" or "peak" shows the display; empty disables it

	goCbRef string
	goID    string
}

// NewService creates a pipeline orchestrator.
// sps is the sampling rate, and frequency is the nominal power-system frequency.
// debounce is the number of consecutive equal results required to change the trip state.
func NewService(
	scaler *features.Scaler,
	detector detection.Detector,
	registrator eventlog.Registrator,
	goosePub domaingoose.Publisher,
	goCbRef, goID string,
	displayMode string,
	sps, frequency int,
	debounce int,
) *Service {
	N := sps / frequency // samples per period
	dft := [8]*features.DFTExtractor{}
	for i := range dft {
		dft[i] = features.NewDFTExtractor(N, 1)
	}
	if debounce < 1 {
		debounce = 1
	}
	var ringBufs [8][]float64
	for i := range ringBufs {
		ringBufs[i] = make([]float64, N)
	}
	return &Service{
		dft:              dft,
		ringBufs:         ringBufs,
		engineer:         features.NewEngineer(scaler),
		window:           features.NewWindow(features.WindowSize, features.WindowStep),
		detector:         detector,
		registrator:      registrator,
		goosePub:         goosePub,
		goCbRef:          goCbRef,
		goID:             goID,
		stNum:            1,
		displayMode:      displayMode,
		samplesPerPeriod: N,
		displayEvery:     sps / displayRefreshHz,
		debounce:         debounce,
	}
}

// Process processes one SVFrame from a 9-2LE stream.
func (s *Service) Process(frame *sv.SVFrame) {
	// Get instantaneous values from all 8 channels in physical units.
	raw := [8]float64{
		frame.CurrentA(), frame.CurrentB(), frame.CurrentC(), frame.CurrentN(),
		frame.VoltageA(), frame.VoltageB(), frame.VoltageC(), frame.VoltageN(),
	}

	// Sliding DFT input: one ring buffer per channel.
	N := s.samplesPerPeriod
	for i, val := range raw {
		s.ringBufs[i][s.ringPos] = val
	}
	s.ringPos++
	s.dftSampleCnt++
	if s.ringPos >= N {
		s.ringPos = 0
		s.ringFull = true
	}

	// Extract phasors only after the buffers are filled and only on DFTStep boundaries.
	// DFTStep is a half-period step, which is 10 ms at 50 Hz and 4000 sps.
	if s.ringFull && s.dftSampleCnt%features.DFTStep == 0 {
		for i := range raw {
			// Collect ring-buffer samples in chronological order.
			ordered := make([]float64, N)
			for j := 0; j < N; j++ {
				ordered[j] = s.ringBufs[i][(s.ringPos+j)%N]
			}
			s.lastPhasors[i] = s.dft[i].Extract(ordered)
		}
	}

	// Refresh the live display at about 10 Hz when display_mode is enabled.
	s.sampleCount++
	if s.displayMode != "" && s.sampleCount%uint64(s.displayEvery) == 0 {
		s.printLiveValues()
	}

	// Skip until the DFT buffer is warm and the next DFTStep boundary arrives.
	if !s.ringFull || s.dftSampleCnt%features.DFTStep != 0 {
		return
	}

	// build the vector 46 features
	vec := s.engineer.Build(s.lastPhasors)

	// Push into the sliding window and run inference when the window is ready.
	window, ready := s.window.Push(vec)
	if !ready {
		return
	}

	s.windowID++

	result, err := s.detector.Detect(window)
	if err != nil {
		slog.Error("detection failed", "error", err)
		return
	}

	// Register only fault events.
	if result.Label == detection.LabelAnomaly {
		event := eventlog.FaultEvent{
			Timestamp: time.Now(),
			WindowID:  s.windowID,
			Features:  vec,
			Result:    result,
		}
		if err := s.registrator.Register(event); err != nil {
			slog.Error("registrator failed", "error", err)
		}
	}

	// Publish GOOSE only after a stable state change.
	// Require s.debounce consecutive windows with the same result.
	trip := result.Label == detection.LabelAnomaly
	if trip == s.pendingTrip {
		s.consecutiveCnt++
	} else {
		s.pendingTrip = trip
		s.consecutiveCnt = 1
	}
	if s.consecutiveCnt >= s.debounce && s.pendingTrip != s.prevTrip {
		s.prevTrip = s.pendingTrip
		s.stNum++
		msg := domaingoose.Message{
			GoCbRef: s.goCbRef,
			GoID:    s.goID,
			StNum:   s.stNum,
			SqNum:   1,
			Trip:    s.prevTrip,
		}
		if err := s.goosePub.Publish(msg); err != nil {
			slog.Error("goose publish failed", "error", err)
		}
	}

	slog.Info("window classified",
		"window_id", s.windowID,
		"label", result.Label.String(),
		"confidence", result.Confidence,
	)
}

// printLiveValues prints phasor amplitudes or RMS values and angles without scrolling.
// Repeated calls rewrite the same lines with ANSI escape codes.
func (s *Service) printLiveValues() {
	if s.displayInit {
		// ESC[nF moves the cursor up n lines and to the start of the line.
		fmt.Printf("\033[%dF", displayLines)
	}

	scale := 1.0
	if s.displayMode == "rms" {
		scale = 1.0 / math.Sqrt2
	}

	p := s.lastPhasors
	fmt.Printf("Current A: %10.3f A   \u2220 %7.2f\u00b0\n", p[0].Amplitude*scale, p[0].Phase*180/math.Pi)
	fmt.Printf("Current B: %10.3f A   \u2220 %7.2f\u00b0\n", p[1].Amplitude*scale, p[1].Phase*180/math.Pi)
	fmt.Printf("Current C: %10.3f A   \u2220 %7.2f\u00b0\n", p[2].Amplitude*scale, p[2].Phase*180/math.Pi)
	fmt.Printf("Current N: %10.3f A   \u2220 %7.2f\u00b0\n", p[3].Amplitude*scale, p[3].Phase*180/math.Pi)
	fmt.Print("\n")
	fmt.Printf("Voltage A: %10.3f V   \u2220 %7.2f\u00b0\n", p[4].Amplitude*scale, p[4].Phase*180/math.Pi)
	fmt.Printf("Voltage B: %10.3f V   \u2220 %7.2f\u00b0\n", p[5].Amplitude*scale, p[5].Phase*180/math.Pi)
	fmt.Printf("Voltage C: %10.3f V   \u2220 %7.2f\u00b0\n", p[6].Amplitude*scale, p[6].Phase*180/math.Pi)
	fmt.Printf("Voltage N: %10.3f V   \u2220 %7.2f\u00b0\n", p[7].Amplitude*scale, p[7].Phase*180/math.Pi)

	s.displayInit = true
}
