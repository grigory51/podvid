package podcast

import (
	"context"
	"fmt"
	"strings"

	goslug "github.com/gosimple/slug"

	"github.com/grigory51/podvid/internal/config"
	"github.com/grigory51/podvid/internal/rss"
	"github.com/grigory51/podvid/internal/storage"
)

const podcastsPrefix = "podcasts/"

type PodcastInfo struct {
	Slug         string
	Name         string
	Description  string
	EpisodeCount int
	FeedURL      string
	CoverURL     string
}

type Service struct {
	s3  *storage.S3Client
	cfg *config.Config
}

func NewService(s3 *storage.S3Client, cfg *config.Config) *Service {
	return &Service{s3: s3, cfg: cfg}
}

func (s *Service) Create(ctx context.Context, name, description string) (*PodcastInfo, error) {
	slug := slugify(name)
	feedKey := feedKey(slug)

	exists, err := s.s3.Exists(ctx, feedKey)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("podcast %q already exists", slug)
	}

	feedURL := s.publicURL(feedKey)
	feed := rss.NewFeed(name, description, feedURL)

	data, err := rss.Marshal(feed)
	if err != nil {
		return nil, err
	}

	if err := s.s3.UploadFile(ctx, feedKey, data, "application/xml"); err != nil {
		return nil, err
	}

	return &PodcastInfo{
		Slug:        slug,
		Name:        name,
		Description: description,
		FeedURL:     feedURL,
	}, nil
}

func (s *Service) List(ctx context.Context) ([]PodcastInfo, error) {
	keys, err := s.s3.ListObjects(ctx, podcastsPrefix)
	if err != nil {
		return nil, err
	}

	slugs := make(map[string]bool)
	for _, key := range keys {
		parts := strings.Split(strings.TrimPrefix(key, podcastsPrefix), "/")
		if len(parts) > 0 && parts[0] != "" {
			slugs[parts[0]] = true
		}
	}

	var podcasts []PodcastInfo
	for slug := range slugs {
		feed, err := s.getFeed(ctx, slug)
		if err != nil {
			continue
		}
		podcasts = append(podcasts, PodcastInfo{
			Slug:         slug,
			Name:         feed.Channel.Title,
			Description:  feed.Channel.Description,
			EpisodeCount: len(feed.Channel.Items),
			FeedURL:      s.publicURL(feedKey(slug)),
			CoverURL:     coverURL(feed),
		})
	}
	return podcasts, nil
}

func (s *Service) Get(ctx context.Context, slug string) (*PodcastInfo, error) {
	feed, err := s.getFeed(ctx, slug)
	if err != nil {
		return nil, err
	}
	return &PodcastInfo{
		Slug:         slug,
		Name:         feed.Channel.Title,
		Description:  feed.Channel.Description,
		EpisodeCount: len(feed.Channel.Items),
		FeedURL:      s.publicURL(feedKey(slug)),
		CoverURL:     coverURL(feed),
	}, nil
}

func (s *Service) Edit(ctx context.Context, slug string, name, description *string) error {
	feed, err := s.getFeed(ctx, slug)
	if err != nil {
		return err
	}

	if name != nil {
		feed.Channel.Title = *name
	}
	if description != nil {
		feed.Channel.Description = *description
	}

	return s.saveFeed(ctx, slug, feed)
}

func (s *Service) SetCover(ctx context.Context, slug string, imageData []byte, contentType string) error {
	ext := "jpg"
	if strings.Contains(contentType, "png") {
		ext = "png"
	} else if strings.Contains(contentType, "webp") {
		ext = "webp"
	}
	coverKey := fmt.Sprintf("%s%s/cover.%s", podcastsPrefix, slug, ext)

	if err := s.s3.UploadFile(ctx, coverKey, imageData, contentType); err != nil {
		return err
	}

	feed, err := s.getFeed(ctx, slug)
	if err != nil {
		return err
	}
	feed.SetImage(s.publicURL(coverKey))
	return s.saveFeed(ctx, slug, feed)
}

func (s *Service) RemoveCover(ctx context.Context, slug string) error {
	feed, err := s.getFeed(ctx, slug)
	if err != nil {
		return err
	}

	// Delete cover files from S3 (could be jpg, png, or webp)
	for _, ext := range []string{"jpg", "png", "webp"} {
		key := fmt.Sprintf("%s%s/cover.%s", podcastsPrefix, slug, ext)
		_ = s.s3.Delete(ctx, key)
	}

	feed.ClearImage()
	return s.saveFeed(ctx, slug, feed)
}

func (s *Service) Delete(ctx context.Context, slug string) error {
	prefix := podcastsPrefix + slug + "/"
	return s.s3.DeletePrefix(ctx, prefix)
}

func (s *Service) GetFeed(ctx context.Context, slug string) (*rss.Feed, error) {
	return s.getFeed(ctx, slug)
}

func (s *Service) SaveFeed(ctx context.Context, slug string, feed *rss.Feed) error {
	return s.saveFeed(ctx, slug, feed)
}

func (s *Service) getFeed(ctx context.Context, slug string) (*rss.Feed, error) {
	data, err := s.s3.Download(ctx, feedKey(slug))
	if err != nil {
		return nil, fmt.Errorf("podcast %q not found: %w", slug, err)
	}
	return rss.Unmarshal(data)
}

func (s *Service) saveFeed(ctx context.Context, slug string, feed *rss.Feed) error {
	data, err := rss.Marshal(feed)
	if err != nil {
		return err
	}
	return s.s3.UploadFile(ctx, feedKey(slug), data, "application/xml")
}

func (s *Service) publicURL(key string) string {
	base := strings.TrimRight(s.cfg.S3.PublicBaseURL, "/")
	return base + "/" + key
}

func (s *Service) EpisodeAudioKey(slug, episodeID string) string {
	return fmt.Sprintf("%s%s/episodes/%s.mp3", podcastsPrefix, slug, episodeID)
}

func feedKey(slug string) string {
	return podcastsPrefix + slug + "/feed.xml"
}

func slugify(name string) string {
	s := goslug.Make(name)
	if s == "" {
		s = "podcast"
	}
	return s
}

func coverURL(feed *rss.Feed) string {
	if feed.Channel.Image != nil {
		return feed.Channel.Image.Href
	}
	return ""
}
