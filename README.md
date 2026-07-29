<p align="center">
  <img src="assets/logo.jpg" width="128" alt="podvid logo">
</p>

<h1 align="center">podvid</h1>

<p align="center">
  Turn any video into a podcast episode on your own S3 storage.
</p>

<p align="center">
  <a href="https://github.com/grigory51/podvid/actions/workflows/release.yml"><img src="https://github.com/grigory51/podvid/actions/workflows/release.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/grigory51/podvid/releases/latest"><img src="https://img.shields.io/github/v/release/grigory51/podvid" alt="Release"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-green" alt="MIT License"></a>
  <a href="https://claude.ai/code"><img src="https://img.shields.io/badge/Built%20with-Claude%20Code-blueviolet?logo=anthropic" alt="Built with Claude Code"></a>
</p>

---

**podvid** downloads videos from any platform supported by [yt-dlp](https://github.com/yt-dlp/yt-dlp) (YouTube, VK Video, RuTube, and [1800+ others](https://github.com/yt-dlp/yt-dlp/blob/master/supportedsites.md)), extracts the audio as MP3, uploads it to an S3-compatible storage, and maintains a valid RSS feed so you can subscribe in any podcast app.

Built for listening to video bloggers on an iPod Nano 7, but works with any podcast client.

## Features

- **TUI mode** — interactive terminal UI (Bubble Tea), just run `podvid`
- **CLI mode** — scriptable commands for automation
- **MCP mode** — expose podvid to AI agents over the Model Context Protocol (`podvid mcp`)
- **Auto-provisioning** — installs yt-dlp automatically via Python venv if not found in PATH
- **S3-compatible** — works with AWS S3, Yandex Object Storage, MinIO, Selectel, etc.
- **iTunes RSS** — generates valid RSS 2.0 with iTunes extensions (cover art, duration, per-episode thumbnails)
- **Episode metadata** — pulls title, description, thumbnail, duration and publish date from the source

## Requirements

- Go 1.21+
- ffmpeg in PATH
- Python 3 (for auto-installing yt-dlp) or yt-dlp in PATH

## Install

```bash
go install github.com/grigory51/podvid/cmd/podvid@latest
```

Or build from source:

```bash
git clone https://github.com/grigory51/podvid.git
cd podvid
go build ./cmd/podvid/
```

## Quick start

```bash
# 1. Configure S3 credentials
podvid config init

# 2. Create a podcast
podvid podcast create --name "My Podcast"

# 3. Add an episode from any video URL
podvid episode add my-podcast https://vkvideo.ru/video-1980_456246417

# 4. Subscribe to the feed URL in your podcast app
```

Or just run `podvid` for the interactive TUI.

## CLI reference

```
podvid                                        # Launch TUI
podvid config init                            # Interactive S3 setup
podvid config show                            # Show current config

podvid podcast create --name "..." [--description "..."] [--cover image.jpg]
podvid podcast list
podvid podcast edit <slug> [--name "..."] [--description "..."] [--cover image.jpg]
podvid podcast delete <slug>

podvid episode add <podcast-slug> <url>
podvid episode list <podcast-slug>
podvid episode edit <podcast-slug> <episode-id> [--title "..."] [--description "..."]
podvid episode delete <podcast-slug> <episode-id>
```

## MCP mode

`podvid mcp` starts a [Model Context Protocol](https://modelcontextprotocol.io) server over stdio,
letting an agent (e.g. Claude) perform the same operations you do in the TUI or CLI. It reads the
same config file and S3 credentials.

Register it with an MCP client. For Claude Code:

```bash
claude mcp add podvid -- podvid mcp
```

Or in a client config (`mcpServers`):

```json
{
  "mcpServers": {
    "podvid": {
      "command": "podvid",
      "args": ["mcp"]
    }
  }
}
```

An MCP client starts the server with its own environment, so the config file is not the only option —
pass the bucket and credentials through `env` and no `config.yaml` is needed at all:

```json
{
  "mcpServers": {
    "podvid": {
      "command": "podvid",
      "args": ["mcp"],
      "env": {
        "PODVID_S3_ENDPOINT": "https://storage.yandexcloud.net",
        "PODVID_S3_REGION": "ru-central1",
        "PODVID_S3_BUCKET": "my-podcasts",
        "PODVID_S3_ACCESS_KEY": "...",
        "PODVID_S3_SECRET_KEY": "...",
        "PODVID_S3_PUBLIC_BASE_URL": "https://storage.yandexcloud.net/my-podcasts"
      }
    }
  }
}
```

Exposed tools:

| Tool | Description |
| --- | --- |
| `podcast_list` | List all podcasts |
| `podcast_get` | Get one podcast by slug |
| `podcast_create` | Create a podcast |
| `podcast_edit` | Edit name/description |
| `episode_list` | List a podcast's episodes |
| `episode_add` | Download a video URL and add it as an episode |
| `episode_edit` | Edit episode title/description |
| `config_show` | Show current config (secrets masked) |

Destructive operations (deleting podcasts/episodes, removing covers) are intentionally **not**
exposed over MCP. Use the TUI or CLI for those.

## Configuration

Config file: `~/.config/podvid/config.yaml`

```yaml
s3:
  endpoint: "https://storage.yandexcloud.net"
  region: "ru-central1"
  bucket: "my-podcasts"
  access_key: "..."
  secret_key: "..."
  public_base_url: "https://storage.yandexcloud.net/my-podcasts"

audio:
  bitrate: "192k"
```

Every value can be overridden by an environment variable or a CLI flag. Environment variables win over
the config file, and the file is optional — set the variables and podvid runs without it.

| Variable | Config key |
| --- | --- |
| `PODVID_S3_ENDPOINT` | `s3.endpoint` |
| `PODVID_S3_REGION` | `s3.region` |
| `PODVID_S3_BUCKET` | `s3.bucket` |
| `PODVID_S3_ACCESS_KEY` | `s3.access_key` |
| `PODVID_S3_SECRET_KEY` | `s3.secret_key` |
| `PODVID_S3_PUBLIC_BASE_URL` | `s3.public_base_url` |
| `PODVID_AUDIO_BITRATE` | `audio.bitrate` |

```bash
PODVID_S3_BUCKET=other-bucket podvid mcp
```

## How it works

1. Downloads video via yt-dlp
2. Extracts audio track, converts to MP3 via ffmpeg
3. Uploads MP3 + episode thumbnail to S3
4. Updates the RSS feed (XML) on the same S3 bucket
5. Podcast app fetches the feed and shows new episodes

## License

MIT
