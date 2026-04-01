package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/grigory51/podvid/internal/podcast"
)

type episodesModel struct {
	svc      *podcast.Service
	slug     string
	episodes []podcast.EpisodeInfo
	cursor   int
	loading  bool
	err      error
}

type episodesLoadedMsg struct {
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
		episodes, err := m.svc.ListEpisodes(context.Background(), m.slug)
		return episodesLoadedMsg{episodes: episodes, err: err}
	}
}

func (m *episodesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case episodesLoadedMsg:
		m.loading = false
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
	s := titleStyle.Render(fmt.Sprintf("%s — Episodes", m.slug)) + "\n\n"

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
		for i, ep := range m.episodes {
			line := fmt.Sprintf("%-40s %s  %s", truncate(ep.Title, 40), ep.PubDate[:16], ep.Duration)
			if i == m.cursor {
				s += selectedStyle.Render("> "+line) + "\n"
			} else {
				s += normalStyle.Render(line) + "\n"
			}
		}
	}

	s += "\n" + helpStyle.Render("[a] Add  [e] Edit  [d] Delete  [esc] Back  [q] Quit")
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
