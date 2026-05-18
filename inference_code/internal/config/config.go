package config

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Interface string    `yaml:"interface"`
	AppID     uint16    `yaml:"app_id"`
	SV        SVCfg     `yaml:"sv"`
	Scaler    ScalerCfg `yaml:"scaler"`
	Model     ModelCfg  `yaml:"model"`
	GOOSE     GOOSECfg  `yaml:"goose"`
	Log       LogCfg    `yaml:"log"`
}

type SVCfg struct {
	SrcMAC    string `yaml:"src_mac"`
	DstMAC    string `yaml:"dst_mac"`
	SPS       int    `yaml:"sps"`
	Frequency int    `yaml:"frequency"`
}

type ScalerCfg struct {
	UNom float64 `yaml:"u_nom"`
	INom float64 `yaml:"i_nom"`
}

type ModelCfg struct {
	Path      string  `yaml:"path"`
	Threshold float64 `yaml:"threshold"`
	Debounce  int     `yaml:"debounce"`
}

type GOOSECfg struct {
	Interface  string `yaml:"interface"`
	AppID      uint16 `yaml:"app_id"`
	GoID       string `yaml:"go_id"`
	GoCbRef    string `yaml:"go_cb_ref"`
	DstMAC     string `yaml:"dst_mac"`
	InvertTrip bool   `yaml:"invert_trip"`
}

type LogCfg struct {
	Path        string `yaml:"path"`
	Level       string `yaml:"level"`
	DisplayMode string `yaml:"display_mode"`
}

func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, err
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) validate() error {
	if c.Interface == "" {
		return fmt.Errorf("config: interface is empty")
	}
	if c.Scaler.UNom <= 0 {
		return fmt.Errorf("config: scaler.u_nom must be > 0")
	}
	if c.Scaler.INom <= 0 {
		return fmt.Errorf("config: scaler.i_nom must be > 0")
	}
	if c.Model.Path == "" {
		return fmt.Errorf("config: model.path is empty")
	}
	if c.Log.Path == "" {
		return fmt.Errorf("config: log.path is empty")
	}
	if c.Model.Threshold < 0 || c.Model.Threshold > 1 {
		c.Model.Threshold = 0.5
	}
	if c.Log.Level == "" {
		c.Log.Level = "ERROR"
	}
	if c.Model.Debounce <= 0 {
		c.Model.Debounce = 3
	}
	if c.SV.SPS == 0 {
		c.SV.SPS = 4000
	}
	if c.SV.Frequency == 0 {
		c.SV.Frequency = 50
	}
	if c.SV.SPS%c.SV.Frequency != 0 {
		return fmt.Errorf("config: sv.sps (%d) must be divisible by sv.frequency (%d)", c.SV.SPS, c.SV.Frequency)
	}
	for _, m := range []struct{ name, val string }{
		{"sv.src_mac", c.SV.SrcMAC},
		{"sv.dst_mac", c.SV.DstMAC},
		{"goose.dst_mac", c.GOOSE.DstMAC},
	} {
		if m.val != "" {
			if _, err := net.ParseMAC(m.val); err != nil {
				return fmt.Errorf("config: %s: invalid MAC %q: %w", m.name, m.val, err)
			}
		}
	}
	return nil
}

func (c *Config) SamplesPerPeriod() int {
	return c.SV.SPS / c.SV.Frequency
}

func (c *Config) SlogLevel() slog.Level {
	switch strings.ToUpper(c.Log.Level) {
	case "INFO":
		return slog.LevelInfo
	case "WARN", "WARNING":
		return slog.LevelWarn
	default:
		return slog.LevelError
	}
}

func (c *Config) DisplayMode() string {
	switch strings.ToLower(c.Log.DisplayMode) {
	case "rms":
		return "rms"
	case "peak":
		return "peak"
	default:
		return ""
	}
}
