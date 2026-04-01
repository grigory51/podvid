package tui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/grigory51/podvid/internal/config"
	"github.com/grigory51/podvid/internal/downloader"
	"github.com/grigory51/podvid/internal/podcast"
)

type addEpisodeModel struct {
	svc  *podcast.Service
	cfg  *config.Config
	slug string

	urlInput   textinput.Model
	status     string
	working    bool
	done       bool
	err        error
	statusChan chan string
}

type addEpisodeStatusMsg struct{ status string }
type addEpisodeDoneMsg struct {
	result *podcast.AddEpisodeResult
	err    error
}

func newAddEpisode(svc *podcast.Service, cfg *config.Config, slug string) *addEpisodeModel {
	ti := textinput.New()
	ti.Placeholder = "Any URL supported by yt-dlp"
	ti.Focus()
	ti.CharLimit = 500
	ti.Width = 60

	return &addEpisodeModel{
		svc:      svc,
		cfg:      cfg,
		slug:     slug,
		urlInput: ti,
	}
}

func (m *addEpisodeModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *addEpisodeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case addEpisodeStatusMsg:
		m.status = msg.status
		return m, m.waitForStatus()

	case addEpisodeDoneMsg:
		m.done = true
		m.working = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.status = fmt.Sprintf("Added: %s", msg.result.Episode.Title)
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.working {
				return m, nil
			}
			return m, func() tea.Msg {
				return switchScreenMsg{screen: screenEpisodes, data: m.slug}
			}
		case "enter":
			if m.working {
				return m, nil
			}
			if m.done {
				return m, func() tea.Msg {
					return switchScreenMsg{screen: screenEpisodes, data: m.slug}
				}
			}
			url := m.urlInput.Value()
			if url == "" {
				return m, nil
			}
			m.status = "Starting..."
			m.working = true
			m.urlInput.Blur()
			m.statusChan = make(chan string, 32)
			return m, tea.Batch(m.startDownload(url), m.waitForStatus())
		}
	}

	if !m.working && !m.done {
		var cmd tea.Cmd
		m.urlInput, cmd = m.urlInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *addEpisodeModel) waitForStatus() tea.Cmd {
	ch := m.statusChan
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		status, ok := <-ch
		if !ok {
			return nil
		}
		return addEpisodeStatusMsg{status: status}
	}
}

func (m *addEpisodeModel) startDownload(url string) tea.Cmd {
	ch := m.statusChan
	return func() tea.Msg {
		defer close(ch)

		progress := func(s string) {
			ch <- s
		}

		progress("Checking ffmpeg...")
		if err := downloader.EnsureFFmpeg(); err != nil {
			return addEpisodeDoneMsg{err: err}
		}

		progress("Checking yt-dlp...")
		ytdlpPath, err := downloader.EnsureYtDlp(progress)
		if err != nil {
			return addEpisodeDoneMsg{err: err}
		}

		dl := downloader.New(ytdlpPath, m.cfg.Audio.Bitrate)

		result, err := m.svc.AddEpisode(context.Background(), m.slug, url, dl, progress)
		return addEpisodeDoneMsg{result: result, err: err}
	}
}

func (m *addEpisodeModel) View() string {
	s := titleStyle.Render(fmt.Sprintf("Add episode to %q", m.slug)) + "\n\n"

	if m.done {
		if m.err != nil {
			s += errorStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n"
		} else {
			s += successStyle.Render(m.status) + "\n"
		}
		s += "\n" + helpStyle.Render("[enter] Back  [esc] Back")
		return s
	}

	if m.working {
		s += progressStyle.Render(m.status) + "\n"
		return s
	}

	s += inputLabelStyle.Render("Video URL:") + "\n"
	s += "  " + m.urlInput.View() + "\n"
	s += "\n" + helpStyle.Render("[enter] Download  [esc] Cancel")
	return s
}
