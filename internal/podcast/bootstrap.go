package podcast

import (
	"path/filepath"
	"strings"

	"github.com/grigory51/podvid/internal/config"
	"github.com/grigory51/podvid/internal/downloader"
	"github.com/grigory51/podvid/internal/storage"
)

// NewServiceFromConfig builds a Service from a validated config. It is the
// single bootstrap path shared by every frontend (TUI, CLI, MCP) so that S3
// client construction is not duplicated across them.
//
// The caller is responsible for deciding whether the config must be valid:
// callers that require S3 (CLI, MCP) should call cfg.Validate() first; the TUI
// tolerates an unconfigured state and shows the config form instead.
func NewServiceFromConfig(cfg *config.Config) (*Service, error) {
	s3Client, err := storage.NewS3Client(cfg)
	if err != nil {
		return nil, err
	}
	return NewService(s3Client, cfg), nil
}

// NewDownloader ensures ffmpeg and yt-dlp are available and returns a ready
// Downloader configured with the audio bitrate from cfg. progressFn receives
// human-readable status messages emitted while provisioning yt-dlp; pass a
// no-op (or nil) when there is no surface to display them on.
//
// This centralizes the EnsureFFmpeg + EnsureYtDlp + New sequence that all three
// frontends previously repeated.
func NewDownloader(cfg *config.Config, progressFn func(string)) (*downloader.Downloader, error) {
	if progressFn == nil {
		progressFn = func(string) {}
	}

	progressFn("Checking ffmpeg...")
	if err := downloader.EnsureFFmpeg(); err != nil {
		return nil, err
	}

	progressFn("Checking yt-dlp...")
	ytdlpPath, err := downloader.EnsureYtDlp(progressFn)
	if err != nil {
		return nil, err
	}

	return downloader.New(ytdlpPath, cfg.Audio.Bitrate), nil
}

// ContentTypeForImage returns the image MIME type for a file path based on its
// extension, defaulting to image/jpeg. Used when uploading podcast covers.
func ContentTypeForImage(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	default:
		return "image/jpeg"
	}
}
