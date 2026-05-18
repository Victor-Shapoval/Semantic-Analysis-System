package features

// DFTStep is the sliding DFT step in samples.
// One phasor spans 80 samples, or 20 ms, for one 50 Hz period.
// The step is 40 samples, a half-period or 10 ms.
// This matches STEP_DFT=40 from the Python pipeline.
const DFTStep = 40

// WindowSize is the sliding-window size in phasors.
// 10 phasors × 20 ms = 200 ms.
// This matches WINDOW_SIZE=10 from the Python pipeline.
const WindowSize = 10

// WindowStep is the sliding-window step in phasors.
// 5 phasors × 20 ms = 100 ms between windows.
// This matches STEP=5 from the Python pipeline.
const WindowStep = 5

// Window accumulates consecutive FeatureVector values and signals when the window is filled.
// It implements a sliding window with WindowStep stride using a ring buffer for O(1) insertion.
type Window struct {
	buf   []FeatureVector // fixed-size ring buffer
	size  int
	step  int
	pos   int // write position in the ring
	count int // total number of added elements
	since int // number of new elements since the last trigger
}

// NewWindow creates a Window with the given size and step.
func NewWindow(size, step int) *Window {
	return &Window{
		buf:  make([]FeatureVector, size),
		size: size,
		step: step,
	}
}

// Push adds a vector to the buffer.
// It returns a WindowSize slice and true when the window is ready for inference,
// otherwise (nil, false).
func (w *Window) Push(v FeatureVector) ([]FeatureVector, bool) {
	w.buf[w.pos] = v
	w.pos = (w.pos + 1) % w.size
	w.count++
	w.since++

	if w.count >= w.size && w.since >= w.step {
		w.since = 0
		// Collect data from the ring in chronological order.
		out := make([]FeatureVector, w.size)
		for i := 0; i < w.size; i++ {
			out[i] = w.buf[(w.pos+i)%w.size]
		}
		return out, true
	}
	return nil, false
}
