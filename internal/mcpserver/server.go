// Package mcpserver exposes podvid operations over the Model Context Protocol
// (MCP) so that an agent can manage podcasts and episodes the same way a user
// does through the TUI or CLI. It is a thin transport layer over
// podcast.Service; all business logic lives there.
//
// Destructive operations (deleting podcasts/episodes, removing covers) are
// intentionally not exposed, to prevent an agent from accidentally erasing data.
package mcpserver

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/grigory51/podvid/internal/config"
	"github.com/grigory51/podvid/internal/podcast"
)

// Version reported to MCP clients during the initialize handshake.
const Version = "0.1.0"

// deps groups the collaborators the tool handlers need. Built once in Run and
// captured by the handler closures.
type deps struct {
	svc *podcast.Service
	cfg *config.Config
}

// Run builds the MCP server, registers the tools, and serves over stdio until
// the client disconnects or the context is cancelled. The transport is stdio
// because that is how MCP clients (Claude and others) launch a server: as a
// child process speaking JSON-RPC over stdin/stdout.
func Run(ctx context.Context, cfg *config.Config) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w\nRun 'podvid config init' to set up", err)
	}

	svc, err := podcast.NewServiceFromConfig(cfg)
	if err != nil {
		return err
	}

	d := &deps{svc: svc, cfg: cfg}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "podvid",
		Title:   "podvid",
		Version: Version,
	}, nil)

	d.registerTools(server)

	return server.Run(ctx, &mcp.StdioTransport{})
}

// registerTools wires every exposed operation as an MCP tool. Input and output
// types are plain structs so the SDK can infer JSON schemas automatically.
func (d *deps) registerTools(s *mcp.Server) {
	// --- Podcasts ---

	mcp.AddTool(s, &mcp.Tool{
		Name:        "podcast_list",
		Description: "List all podcasts with their slug, name, episode count and feed URL.",
	}, d.podcastList)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "podcast_get",
		Description: "Get a single podcast by slug, including name, description, episode count, feed URL and cover URL.",
	}, d.podcastGet)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "podcast_create",
		Description: "Create a new podcast. The slug is derived from the name. Returns the created podcast including its feed URL.",
	}, d.podcastCreate)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "podcast_edit",
		Description: "Edit a podcast's name and/or description. Only the provided fields are changed.",
	}, d.podcastEdit)

	// --- Episodes ---

	mcp.AddTool(s, &mcp.Tool{
		Name:        "episode_list",
		Description: "List episodes of a podcast, including id, title, duration, publish date and audio URL.",
	}, d.episodeList)

	mcp.AddTool(s, &mcp.Tool{
		Name: "episode_add",
		Description: "Download a video from any yt-dlp supported URL (VK Video, RuTube, YouTube, etc.), " +
			"extract its audio, upload it to S3 and add it as a new episode of the given podcast. " +
			"This is a long-running operation: it downloads and transcodes the video.",
	}, d.episodeAdd)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "episode_edit",
		Description: "Edit an episode's title and/or description. Only the provided fields are changed.",
	}, d.episodeEdit)

	// --- Config ---

	mcp.AddTool(s, &mcp.Tool{
		Name:        "config_show",
		Description: "Show the current S3 and audio configuration. Secrets are masked.",
	}, d.configShow)
}

// --- Podcast tools ---

type podcastListOut struct {
	Podcasts []podcastDTO `json:"podcasts"`
}

type podcastDTO struct {
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	EpisodeCount int    `json:"episode_count"`
	FeedURL      string `json:"feed_url"`
	CoverURL     string `json:"cover_url,omitempty"`
}

func toPodcastDTO(p *podcast.PodcastInfo) podcastDTO {
	return podcastDTO{
		Slug:         p.Slug,
		Name:         p.Name,
		Description:  p.Description,
		EpisodeCount: p.EpisodeCount,
		FeedURL:      p.FeedURL,
		CoverURL:     p.CoverURL,
	}
}

func (d *deps) podcastList(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, podcastListOut, error) {
	podcasts, err := d.svc.List(ctx)
	if err != nil {
		return nil, podcastListOut{}, err
	}
	out := podcastListOut{Podcasts: make([]podcastDTO, 0, len(podcasts))}
	for i := range podcasts {
		out.Podcasts = append(out.Podcasts, toPodcastDTO(&podcasts[i]))
	}
	return nil, out, nil
}

type podcastGetIn struct {
	Slug string `json:"slug" jsonschema:"the podcast slug"`
}

func (d *deps) podcastGet(ctx context.Context, _ *mcp.CallToolRequest, in podcastGetIn) (*mcp.CallToolResult, podcastDTO, error) {
	info, err := d.svc.Get(ctx, in.Slug)
	if err != nil {
		return nil, podcastDTO{}, err
	}
	return nil, toPodcastDTO(info), nil
}

type podcastCreateIn struct {
	Name        string `json:"name" jsonschema:"the podcast name; the slug is derived from it"`
	Description string `json:"description,omitempty" jsonschema:"optional podcast description"`
}

func (d *deps) podcastCreate(ctx context.Context, _ *mcp.CallToolRequest, in podcastCreateIn) (*mcp.CallToolResult, podcastDTO, error) {
	if in.Name == "" {
		return nil, podcastDTO{}, fmt.Errorf("name is required")
	}
	info, err := d.svc.Create(ctx, in.Name, in.Description)
	if err != nil {
		return nil, podcastDTO{}, err
	}
	return nil, toPodcastDTO(info), nil
}

type podcastEditIn struct {
	Slug        string  `json:"slug" jsonschema:"the podcast slug"`
	Name        *string `json:"name,omitempty" jsonschema:"new name; omit to keep current"`
	Description *string `json:"description,omitempty" jsonschema:"new description; omit to keep current"`
}

func (d *deps) podcastEdit(ctx context.Context, _ *mcp.CallToolRequest, in podcastEditIn) (*mcp.CallToolResult, podcastDTO, error) {
	if err := d.svc.Edit(ctx, in.Slug, in.Name, in.Description); err != nil {
		return nil, podcastDTO{}, err
	}
	info, err := d.svc.Get(ctx, in.Slug)
	if err != nil {
		return nil, podcastDTO{}, err
	}
	return nil, toPodcastDTO(info), nil
}

// --- Episode tools ---

type episodeDTO struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Duration    string `json:"duration"`
	PubDate     string `json:"pub_date"`
	AudioURL    string `json:"audio_url"`
	FileSize    int64  `json:"file_size"`
}

func toEpisodeDTO(e *podcast.EpisodeInfo) episodeDTO {
	return episodeDTO{
		ID:          e.ID,
		Title:       e.Title,
		Description: e.Description,
		Duration:    e.Duration,
		PubDate:     e.PubDate,
		AudioURL:    e.AudioURL,
		FileSize:    e.FileSize,
	}
}

type episodeListIn struct {
	Slug string `json:"slug" jsonschema:"the podcast slug"`
}

type episodeListOut struct {
	Episodes []episodeDTO `json:"episodes"`
}

func (d *deps) episodeList(ctx context.Context, _ *mcp.CallToolRequest, in episodeListIn) (*mcp.CallToolResult, episodeListOut, error) {
	episodes, err := d.svc.ListEpisodes(ctx, in.Slug)
	if err != nil {
		return nil, episodeListOut{}, err
	}
	out := episodeListOut{Episodes: make([]episodeDTO, 0, len(episodes))}
	for i := range episodes {
		out.Episodes = append(out.Episodes, toEpisodeDTO(&episodes[i]))
	}
	return nil, out, nil
}

type episodeAddIn struct {
	Slug     string `json:"slug" jsonschema:"the target podcast slug"`
	VideoURL string `json:"video_url" jsonschema:"the video URL to download (any yt-dlp supported source)"`
}

type episodeAddOut struct {
	Episode episodeDTO `json:"episode"`
	FeedURL string     `json:"feed_url"`
}

func (d *deps) episodeAdd(ctx context.Context, _ *mcp.CallToolRequest, in episodeAddIn) (*mcp.CallToolResult, episodeAddOut, error) {
	// Progress messages are discarded: there is no interactive surface to show
	// them on, and the agent only needs the final result.
	dl, err := podcast.NewDownloader(d.cfg, nil)
	if err != nil {
		return nil, episodeAddOut{}, err
	}

	result, err := d.svc.AddEpisode(ctx, in.Slug, in.VideoURL, dl, func(string) {})
	if err != nil {
		return nil, episodeAddOut{}, err
	}

	return nil, episodeAddOut{
		Episode: toEpisodeDTO(result.Episode),
		FeedURL: result.FeedURL,
	}, nil
}

type episodeEditIn struct {
	Slug        string  `json:"slug" jsonschema:"the podcast slug"`
	EpisodeID   string  `json:"episode_id" jsonschema:"the episode id"`
	Title       *string `json:"title,omitempty" jsonschema:"new title; omit to keep current"`
	Description *string `json:"description,omitempty" jsonschema:"new description; omit to keep current"`
}

func (d *deps) episodeEdit(ctx context.Context, _ *mcp.CallToolRequest, in episodeEditIn) (*mcp.CallToolResult, episodeDTO, error) {
	if err := d.svc.EditEpisode(ctx, in.Slug, in.EpisodeID, in.Title, in.Description); err != nil {
		return nil, episodeDTO{}, err
	}

	episodes, err := d.svc.ListEpisodes(ctx, in.Slug)
	if err != nil {
		return nil, episodeDTO{}, err
	}
	for i := range episodes {
		if episodes[i].ID == in.EpisodeID {
			return nil, toEpisodeDTO(&episodes[i]), nil
		}
	}
	return nil, episodeDTO{}, fmt.Errorf("episode %q not found in podcast %q after edit", in.EpisodeID, in.Slug)
}

// --- Config tool ---

type configShowOut struct {
	Display string `json:"display" jsonschema:"human-readable configuration summary with secrets masked"`
}

func (d *deps) configShow(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, configShowOut, error) {
	return nil, configShowOut{Display: d.cfg.Display()}, nil
}
