package podcast

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/grigory51/podvid/internal/downloader"
	"github.com/grigory51/podvid/internal/rss"
)

type EpisodeInfo struct {
	ID          string
	Title       string
	Description string
	Duration    string
	PubDate     string
	AudioURL    string
	FileSize    int64
}

type AddEpisodeResult struct {
	Episode  *EpisodeInfo
	FeedURL  string
}

func (s *Service) AddEpisode(ctx context.Context, slug, videoURL string, dl *downloader.Downloader, progressFn func(string)) (*AddEpisodeResult, error) {
	if progressFn != nil {
		progressFn("Getting video info...")
	}

	info, err := dl.GetVideoInfo(videoURL)
	if err != nil {
		return nil, fmt.Errorf("getting video info: %w", err)
	}

	if progressFn != nil {
		progressFn(fmt.Sprintf("Downloading: %s", info.Title))
	}

	tmpDir, err := os.MkdirTemp("", "podvid-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	audioPath, err := dl.DownloadAudio(videoURL, tmpDir, func(p downloader.DownloadProgress) {
		if progressFn != nil {
			progressFn(p.Status)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("downloading audio: %w", err)
	}

	audioData, err := os.ReadFile(audioPath)
	if err != nil {
		return nil, fmt.Errorf("reading audio file: %w", err)
	}

	episodeID := strings.TrimSuffix(filepath.Base(audioPath), filepath.Ext(audioPath))
	audioKey := s.EpisodeAudioKey(slug, episodeID)

	if progressFn != nil {
		progressFn("Uploading to S3...")
	}

	if err := s.s3.UploadFile(ctx, audioKey, audioData, "audio/mpeg"); err != nil {
		return nil, fmt.Errorf("uploading audio: %w", err)
	}

	audioURL := s.publicURL(audioKey)
	duration := downloader.FormatDuration(info.Duration)

	pubDate := time.Now()
	if info.Timestamp > 0 {
		pubDate = time.Unix(info.Timestamp, 0)
	}

	item := rss.NewItem(info.Title, audioURL, int64(len(audioData)), duration, pubDate)

	var descParts []string
	if info.Description != "" {
		descParts = append(descParts, info.Description)
	}
	if info.WebpageURL != "" {
		descParts = append(descParts, fmt.Sprintf("Original: %s", info.WebpageURL))
	}
	if len(descParts) > 0 {
		item.Description = strings.Join(descParts, "\n\n")
	}

	if info.Thumbnail != "" {
		if progressFn != nil {
			progressFn("Uploading episode thumbnail...")
		}
		if thumbURL, err := s.uploadThumbnail(ctx, slug, episodeID, info.Thumbnail); err == nil {
			item.Image = &rss.ItunesImg{Href: thumbURL}
		}
	}

	feed, err := s.getFeed(ctx, slug)
	if err != nil {
		return nil, err
	}
	feed.AddItem(item)

	if err := s.saveFeed(ctx, slug, feed); err != nil {
		return nil, fmt.Errorf("updating feed: %w", err)
	}

	if progressFn != nil {
		progressFn("Episode added successfully")
	}

	return &AddEpisodeResult{
		Episode: &EpisodeInfo{
			ID:          episodeID,
			Title:       info.Title,
			Description: info.Description,
			Duration:    duration,
			PubDate:     item.PubDate,
			AudioURL:    audioURL,
			FileSize:    int64(len(audioData)),
		},
		FeedURL: s.publicURL(feedKey(slug)),
	}, nil
}

func (s *Service) ListEpisodes(ctx context.Context, slug string) ([]EpisodeInfo, error) {
	feed, err := s.getFeed(ctx, slug)
	if err != nil {
		return nil, err
	}

	var episodes []EpisodeInfo
	for _, item := range feed.Channel.Items {
		id := extractEpisodeID(item.Enclosure.URL)
		episodes = append(episodes, EpisodeInfo{
			ID:          id,
			Title:       item.Title,
			Description: item.Description,
			Duration:    item.Duration,
			PubDate:     item.PubDate,
			AudioURL:    item.Enclosure.URL,
			FileSize:    item.Enclosure.Length,
		})
	}
	return episodes, nil
}

func (s *Service) EditEpisode(ctx context.Context, slug, episodeID string, title, description *string) error {
	feed, err := s.getFeed(ctx, slug)
	if err != nil {
		return err
	}

	for _, item := range feed.Channel.Items {
		id := extractEpisodeID(item.Enclosure.URL)
		if id == episodeID {
			if title != nil {
				item.Title = *title
			}
			if description != nil {
				item.Description = *description
			}
			return s.saveFeed(ctx, slug, feed)
		}
	}

	return fmt.Errorf("episode %q not found in podcast %q", episodeID, slug)
}

func (s *Service) DeleteEpisode(ctx context.Context, slug, episodeID string) error {
	feed, err := s.getFeed(ctx, slug)
	if err != nil {
		return err
	}

	audioKey := s.EpisodeAudioKey(slug, episodeID)

	found := false
	for _, item := range feed.Channel.Items {
		id := extractEpisodeID(item.Enclosure.URL)
		if id == episodeID {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("episode %q not found in podcast %q", episodeID, slug)
	}

	if err := s.s3.Delete(ctx, audioKey); err != nil {
		return fmt.Errorf("deleting audio file: %w", err)
	}

	// Remove from feed by matching URL
	for i, item := range feed.Channel.Items {
		id := extractEpisodeID(item.Enclosure.URL)
		if id == episodeID {
			feed.Channel.Items = append(feed.Channel.Items[:i], feed.Channel.Items[i+1:]...)
			break
		}
	}

	return s.saveFeed(ctx, slug, feed)
}

func extractEpisodeID(audioURL string) string {
	base := filepath.Base(audioURL)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func (s *Service) uploadThumbnail(ctx context.Context, slug, episodeID, thumbnailURL string) (string, error) {
	resp, err := http.Get(thumbnailURL)
	if err != nil {
		return "", fmt.Errorf("downloading thumbnail: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("thumbnail returned %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading thumbnail: %w", err)
	}

	contentType := resp.Header.Get("Content-Type")
	ext := "jpg"
	if strings.Contains(contentType, "png") {
		ext = "png"
	} else if strings.Contains(contentType, "webp") {
		ext = "webp"
	}

	key := fmt.Sprintf("%s%s/episodes/%s.%s", podcastsPrefix, slug, episodeID, ext)
	if err := s.s3.UploadFile(ctx, key, data, contentType); err != nil {
		return "", fmt.Errorf("uploading thumbnail: %w", err)
	}

	return s.publicURL(key), nil
}
