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

Override any value via environment variables (`PODVID_S3_BUCKET`, `PODVID_S3_ACCESS_KEY`, ...) or CLI flags.

## How it works

1. Downloads video via yt-dlp
2. Extracts audio track, converts to MP3 via ffmpeg
3. Uploads MP3 + episode thumbnail to S3
4. Updates the RSS feed (XML) on the same S3 bucket
5. Podcast app fetches the feed and shows new episodes

## License

MIT
