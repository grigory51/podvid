package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type S3Config struct {
	Endpoint      string `yaml:"endpoint"`
	Region        string `yaml:"region"`
	Bucket        string `yaml:"bucket"`
	AccessKey     string `yaml:"access_key"`
	SecretKey     string `yaml:"secret_key"`
	PublicBaseURL string `yaml:"public_base_url"`
}

type AudioConfig struct {
	Bitrate string `yaml:"bitrate"`
}

type Config struct {
	S3    S3Config    `yaml:"s3"`
	Audio AudioConfig `yaml:"audio"`
}

func DefaultConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "podvid")
}

func DefaultConfigPath() string {
	return filepath.Join(DefaultConfigDir(), "config.yaml")
}

func Load(path string) (*Config, error) {
	cfg := &Config{
		Audio: AudioConfig{Bitrate: "192k"},
	}

	if path == "" {
		path = DefaultConfigPath()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			applyEnv(cfg)
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	applyEnv(cfg)
	return cfg, nil
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("PODVID_S3_ENDPOINT"); v != "" {
		cfg.S3.Endpoint = v
	}
	if v := os.Getenv("PODVID_S3_REGION"); v != "" {
		cfg.S3.Region = v
	}
	if v := os.Getenv("PODVID_S3_BUCKET"); v != "" {
		cfg.S3.Bucket = v
	}
	if v := os.Getenv("PODVID_S3_ACCESS_KEY"); v != "" {
		cfg.S3.AccessKey = v
	}
	if v := os.Getenv("PODVID_S3_SECRET_KEY"); v != "" {
		cfg.S3.SecretKey = v
	}
	if v := os.Getenv("PODVID_S3_PUBLIC_BASE_URL"); v != "" {
		cfg.S3.PublicBaseURL = v
	}
	if v := os.Getenv("PODVID_AUDIO_BITRATE"); v != "" {
		cfg.Audio.Bitrate = v
	}
}

func (cfg *Config) Save(path string) error {
	if path == "" {
		path = DefaultConfigPath()
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

func (cfg *Config) IsS3Configured() bool {
	return cfg.S3.Bucket != "" && cfg.S3.AccessKey != "" && cfg.S3.SecretKey != ""
}

func (cfg *Config) Validate() error {
	if cfg.S3.Bucket == "" {
		return fmt.Errorf("s3.bucket is required")
	}
	if cfg.S3.AccessKey == "" {
		return fmt.Errorf("s3.access_key is required")
	}
	if cfg.S3.SecretKey == "" {
		return fmt.Errorf("s3.secret_key is required")
	}
	if cfg.S3.PublicBaseURL == "" {
		return fmt.Errorf("s3.public_base_url is required")
	}
	return nil
}

// ApplyFlags overrides config values from CLI flags.
func (cfg *Config) ApplyFlags(flags map[string]string) {
	for k, v := range flags {
		if v == "" {
			continue
		}
		switch k {
		case "s3-endpoint":
			cfg.S3.Endpoint = v
		case "s3-region":
			cfg.S3.Region = v
		case "s3-bucket":
			cfg.S3.Bucket = v
		case "s3-access-key":
			cfg.S3.AccessKey = v
		case "s3-secret-key":
			cfg.S3.SecretKey = v
		case "s3-public-base-url":
			cfg.S3.PublicBaseURL = v
		case "audio-bitrate":
			cfg.Audio.Bitrate = v
		}
	}
}

func (cfg *Config) Display() string {
	s := "S3 Configuration:\n"
	s += fmt.Sprintf("  Endpoint:        %s\n", defaultStr(cfg.S3.Endpoint, "(default AWS)"))
	s += fmt.Sprintf("  Region:          %s\n", defaultStr(cfg.S3.Region, "(not set)"))
	s += fmt.Sprintf("  Bucket:          %s\n", defaultStr(cfg.S3.Bucket, "(not set)"))
	s += fmt.Sprintf("  Access Key:      %s\n", maskSecret(cfg.S3.AccessKey))
	s += fmt.Sprintf("  Secret Key:      %s\n", maskSecret(cfg.S3.SecretKey))
	s += fmt.Sprintf("  Public Base URL: %s\n", defaultStr(cfg.S3.PublicBaseURL, "(not set)"))
	s += "\nAudio Configuration:\n"
	s += fmt.Sprintf("  Bitrate:         %s\n", cfg.Audio.Bitrate)
	return s
}

func maskSecret(s string) string {
	if s == "" {
		return "(not set)"
	}
	if len(s) <= 4 {
		return "****"
	}
	return s[:4] + "****" + s[len(s)-4:]
}

func defaultStr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

