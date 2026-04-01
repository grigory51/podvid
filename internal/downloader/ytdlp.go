package downloader

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type VideoInfo struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Duration    float64 `json:"duration"`
	Thumbnail   string  `json:"thumbnail"`
	WebpageURL  string  `json:"webpage_url"`
	Timestamp   int64   `json:"timestamp"`
}

type DownloadProgress struct {
	Stage   string  // "downloading", "converting", "done"
	Percent float64 // 0-100
	Status  string  // human-readable status
}

type Downloader struct {
	ytdlpPath string
	bitrate   string
}

func New(ytdlpPath, bitrate string) *Downloader {
	return &Downloader{ytdlpPath: ytdlpPath, bitrate: bitrate}
}

func (d *Downloader) GetVideoInfo(url string) (*VideoInfo, error) {
	cmd := exec.Command(d.ytdlpPath, "--dump-json", "--no-download", url)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("getting video info: %w", err)
	}

	var info VideoInfo
	if err := json.Unmarshal(out, &info); err != nil {
		return nil, fmt.Errorf("parsing video info: %w", err)
	}
	return &info, nil
}

func (d *Downloader) DownloadAudio(url, outputDir string, progressFn func(DownloadProgress)) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", fmt.Errorf("creating output dir: %w", err)
	}

	outputTemplate := filepath.Join(outputDir, "%(id)s.%(ext)s")

	args := []string{
		"-x",
		"--audio-format", "mp3",
		"--audio-quality", d.bitrate,
		"--newline",
		"--no-playlist",
		"-o", outputTemplate,
		url,
	}

	cmd := exec.Command(d.ytdlpPath, args...)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("creating stderr pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("creating stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("starting yt-dlp: %w", err)
	}

	var outputFile string
	progressRe := regexp.MustCompile(`\[download\]\s+([\d.]+)%`)
	destRe := regexp.MustCompile(`\[(?:ExtractAudio|Merger)\].*Destination:\s*(.+)`)
	alreadyRe := regexp.MustCompile(`\[download\]\s+(.+)\s+has already been downloaded`)

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			// stderr output, just consume
		}
	}()

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()

		if matches := progressRe.FindStringSubmatch(line); len(matches) > 1 {
			if pct, err := strconv.ParseFloat(matches[1], 64); err == nil {
				if progressFn != nil {
					stage := "downloading"
					if strings.Contains(line, "audio") {
						stage = "converting"
					}
					progressFn(DownloadProgress{
						Stage:   stage,
						Percent: pct,
						Status:  fmt.Sprintf("Downloading... %.1f%%", pct),
					})
				}
			}
		}

		if matches := destRe.FindStringSubmatch(line); len(matches) > 1 {
			outputFile = strings.TrimSpace(matches[1])
		}

		if matches := alreadyRe.FindStringSubmatch(line); len(matches) > 1 {
			outputFile = strings.TrimSpace(matches[1])
		}
	}

	if err := cmd.Wait(); err != nil {
		return "", fmt.Errorf("yt-dlp failed: %w", err)
	}

	if outputFile == "" {
		// Try to find the mp3 file in the output dir
		entries, _ := os.ReadDir(outputDir)
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".mp3") {
				outputFile = filepath.Join(outputDir, e.Name())
				break
			}
		}
	}

	if outputFile == "" {
		return "", fmt.Errorf("could not determine output file")
	}

	if progressFn != nil {
		progressFn(DownloadProgress{
			Stage:   "done",
			Percent: 100,
			Status:  "Download complete",
		})
	}

	return outputFile, nil
}

func FormatDuration(seconds float64) string {
	total := int(seconds)
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}
