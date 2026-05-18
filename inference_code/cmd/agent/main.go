package main

import (
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"semantic-analysis-system/internal/application/pipeline"
	"semantic-analysis-system/internal/config"
	"semantic-analysis-system/internal/domain/features"
	"semantic-analysis-system/internal/domain/sv"
	"semantic-analysis-system/internal/infrastructure/capture"
	infraeventlog "semantic-analysis-system/internal/infrastructure/eventlog"
	infragoose "semantic-analysis-system/internal/infrastructure/goose"
	infraonnx "semantic-analysis-system/internal/infrastructure/onnx"
)

func main() {
	cfgPath := flag.String("config", "configs/config.yaml", "path to config")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// all logs are written to a file; the console is reserved for the live display
	logFile, err := os.OpenFile(cfg.Log.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		slog.Error("failed to open log file", "error", err)
		os.Exit(1)
	}
	defer logFile.Close()

	log := slog.New(slog.NewTextHandler(logFile, &slog.HandlerOptions{Level: cfg.SlogLevel()}))
	slog.SetDefault(log)

	// --- infrastructure ---
	detector, err := infraonnx.NewDetector(cfg.Model.Path, cfg.Model.Threshold)
	if err != nil {
		slog.Error("failed to load model", "error", err)
		os.Exit(1)
	}
	defer detector.Close()

	registrator, err := infraeventlog.NewSlogRegistrator(cfg.Log.Path)
	if err != nil {
		slog.Error("failed to open event log", "error", err)
		os.Exit(1)
	}
	defer registrator.Close()

	goosePub, err := infragoose.NewRawPublisher(
		cfg.GOOSE.Interface,
		cfg.GOOSE.DstMAC,
		cfg.GOOSE.GoCbRef,
		cfg.GOOSE.GoID,
		cfg.GOOSE.AppID,
		cfg.GOOSE.InvertTrip,
	)
	if err != nil {
		slog.Error("failed to create GOOSE publisher", "error", err)
		os.Exit(1)
	}
	defer goosePub.Close()

	scaler := features.NewScaler(cfg.Scaler.UNom, cfg.Scaler.INom)

	svc := pipeline.NewService(
		scaler,
		detector,
		registrator,
		goosePub,
		cfg.GOOSE.GoCbRef,
		cfg.GOOSE.GoID,
		cfg.DisplayMode(),
		cfg.SV.SPS,
		cfg.SV.Frequency,
		cfg.Model.Debounce,
	)

	// --- SV capture ---
	capturer := capture.NewCapturer(cfg.Interface, cfg.AppID, cfg.SV.SrcMAC, cfg.SV.DstMAC)

	frames := make(chan *sv.SVFrame, 256)
	errs := make(chan error, 16)
	done := make(chan struct{})
	capDone := make(chan struct{}) // capture goroutine completion signal

	go func() {
		defer close(capDone)
		if err := capturer.Run(done, frames, errs); err != nil {
			slog.Error("capturer stopped", "error", err)
		}
		close(errs)
	}()

	go func() {
		for err := range errs {
			slog.Warn("capture error", "error", err)
		}
	}()

	slog.Info("semantic analysis agent started",
		"interface", cfg.Interface,
		"app_id", cfg.AppID,
		"model", cfg.Model.Path,
		"threshold", cfg.Model.Threshold,
	)

	// --- main loop ---
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case frame := <-frames:
			svc.Process(frame)
		case <-sig:
			slog.Info("shutting down")
			close(done)
			<-capDone // wait for the capture goroutine to finish before closing resources
			return
		}
	}
}
