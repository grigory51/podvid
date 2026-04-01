package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pkg/browser"

	"github.com/grigory51/podvid/internal/podcast"
)

type episodesModel struct {
	svc      *podcast.Service
	slug     string
	name     string
	episodes []podcast.EpisodeInfo
	cursor   int
	width    int
	loading  bool
	err      error
}

type episodesLoadedMsg struct {
	name     string
	episodes []podcast.EpisodeInfo
	err      error
}

type episodeDeletedMsg struct{ err error }

func newEpisodesList(svc *podcast.Service, slug string) *episodesModel {
	return &episodesModel{svc: svc, slug: slug, loading: true}
}

func (m *episodesModel) Init() tea.Cmd {
	return m.loadEpisodes()
}

func (m *episodesModel) loadEpisodes() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var name string
		if info, err := m.svc.Get(ctx, m.slug); err == nil {
			name = info.Name
		}
		episodes, err := m.svc.ListEpisodes(ctx, m.slug)
		return episodesLoadedMsg{name: name, episodes: episodes, err: err}
	}
}

func (m *episodesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width

	case episodesLoadedMsg:
		m.loading = false
		m.name = msg.name
		m.episodes = msg.episodes
		m.err = msg.err
		if m.cursor >= len(m.episodes) {
			m.cursor = max(0, len(m.episodes)-1)
		}

	case episodeDeletedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			return m, m.loadEpisodes()
		}

	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg {
				return switchScreenMsg{screen: screenPodcasts}
			}
		case "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.episodes)-1 {
				m.cursor++
			}
		case "a":
			return m, func() tea.Msg {
				return switchScreenMsg{screen: screenAddEpisode, data: m.slug}
			}
		case "enter":
			if len(m.episodes) > 0 {
				url := m.episodes[m.cursor].AudioURL
				browser.OpenURL(url)
			}
		case "e":
			if len(m.episodes) > 0 {
				ep := m.episodes[m.cursor]
				return m, func() tea.Msg {
					return switchScreenMsg{screen: screenEpisodeForm, data: episodeFormData{slug: m.slug, episode: ep}}
				}
			}
		case "d":
			if len(m.episodes) > 0 {
				ep := m.episodes[m.cursor]
				return m, func() tea.Msg {
					err := m.svc.DeleteEpisode(context.Background(), m.slug, ep.ID)
					return episodeDeletedMsg{err: err}
				}
			}
		}
	}

	return m, nil
}

func (m *episodesModel) View() string {
		title := m.name
	if title == "" {
		title = m.slug
	}
	s := titleStyle.Render(fmt.Sprintf("%s — Episodes", title)) + "\n\n"

	if m.loading {
		s += subtitleStyle.Render("Loading episodes...") + "\n"
		return s
	}

	if m.err != nil {
		s += errorStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n\n"
	}

	if len(m.episodes) == 0 {
		s += subtitleStyle.Render("No episodes yet. Press 'a' to add one.") + "\n"
	} else {
		width := m.width
		if width == 0 {
			width = 80
		}
		// fixed meta: "  " + date(17) + "  " + duration(7) = 28
		// prefix: "> " or "  " (2) + style PaddingLeft(1 or 3) → effectively 3 total
		metaWidth := 28
		titleWidth := width - metaWidth - 3
		if titleWidth < 20 {
			titleWidth = 20
		}
		for i, ep := range m.episodes {
			date := formatPubDate(ep.PubDate)
			title := truncate(ep.Title, titleWidth)
			padding := titleWidth - runeLen(title)
			if padding < 0 {
				padding = 0
			}
			line := title + strings.Repeat(" ", padding) + fmt.Sprintf("  %s  %7s", date, ep.Duration)
			if i == m.cursor {
				s += selectedStyle.Render("> "+line) + "\n"
			} else {
				s += normalStyle.Render(line) + "\n"
			}
		}
	}

	s += "\n" + helpStyle.Render("[enter] Play  [a] Add  [e] Edit  [d] Delete  [esc] Back  [q] Quit")
	return s
}

func formatPubDate(pubDate string) string {
	t, err := time.Parse(time.RFC1123Z, pubDate)
	if err != nil {
		if len(pubDate) > 22 {
			return pubDate[:22]
		}
		return pubDate
	}
	return t.Local().Format("02 Jan 2006 15:04")
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	if n <= 3 {
		return string(runes[:n])
	}
	return string(runes[:n-3]) + "..."
}

func runeLen(s string) int {
	return len([]rune(s))
}
