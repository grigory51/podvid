package downloader

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/grigory51/podvid/internal/config"
)

func FindYtDlp() (string, error) {
	// 1. Check PATH
	if path, err := exec.LookPath("yt-dlp"); err == nil {
		return path, nil
	}

	// 2. Check managed venv
	venvBin := venvYtDlpPath()
	if _, err := os.Stat(venvBin); err == nil {
		return venvBin, nil
	}

	return "", fmt.Errorf("yt-dlp not found")
}

func EnsureYtDlp(progressFn func(status string)) (string, error) {
	path, err := FindYtDlp()
	if err == nil {
		return path, nil
	}

	if progressFn != nil {
		progressFn("yt-dlp not found, installing...")
	}

	pythonPath, err := findPython()
	if err != nil {
		return "", fmt.Errorf("python not found in PATH — install python3 or yt-dlp manually: %w", err)
	}

	venvDir := venvDir()
	if err := os.MkdirAll(filepath.Dir(venvDir), 0o755); err != nil {
		return "", fmt.Errorf("creating venv directory: %w", err)
	}

	if progressFn != nil {
		progressFn("Creating Python virtual environment...")
	}

	cmd := exec.Command(pythonPath, "-m", "venv", venvDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("creating venv: %s: %w", string(out), err)
	}

	if progressFn != nil {
		progressFn("Installing yt-dlp...")
	}

	pipPath := filepath.Join(venvDir, venvBinDir(), "pip")
	cmd = exec.Command(pipPath, "install", "yt-dlp")
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("installing yt-dlp: %s: %w", string(out), err)
	}

	ytdlpPath := venvYtDlpPath()
	if _, err := os.Stat(ytdlpPath); err != nil {
		return "", fmt.Errorf("yt-dlp binary not found after install at %s", ytdlpPath)
	}

	if progressFn != nil {
		progressFn("yt-dlp installed successfully")
	}

	return ytdlpPath, nil
}

func EnsureFFmpeg() error {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("ffmpeg not found in PATH — install it (e.g. brew install ffmpeg)")
	}
	return nil
}

func findPython() (string, error) {
	for _, name := range []string{"python3", "python"} {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("python3/python not found")
}

func venvDir() string {
	return filepath.Join(config.DefaultConfigDir(), "venv")
}

func venvBinDir() string {
	if runtime.GOOS == "windows" {
		return "Scripts"
	}
	return "bin"
}

func venvYtDlpPath() string {
	name := "yt-dlp"
	if runtime.GOOS == "windows" {
		name = "yt-dlp.exe"
	}
	return filepath.Join(venvDir(), venvBinDir(), name)
}
