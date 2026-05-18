package eventlog

import (
	"log/slog"
	"os"

	domaineventlog "semantic-analysis-system/internal/domain/eventlog"
)

// implements domaineventlog.Registrator.
type SlogRegistrator struct {
	file   *os.File
	logger *slog.Logger
}

func NewSlogRegistrator(path string) (*SlogRegistrator, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	logger := slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	return &SlogRegistrator{file: f, logger: logger}, nil
}

func (r *SlogRegistrator) Register(event domaineventlog.FaultEvent) error {
	r.logger.Info("fault_event",
		slog.String("time", event.Timestamp.UTC().Format("2006-01-02T15:04:05.000Z")),
		slog.Uint64("window_id", event.WindowID),
		slog.String("label", event.Result.Label.String()),
		slog.Float64("confidence", float64(event.Result.Confidence)),
	)
	return nil
}

func (r *SlogRegistrator) Close() error {
	return r.file.Close()
}
