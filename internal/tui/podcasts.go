package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/grigory51/podvid/internal/podcast"
)

type podcastsModel struct {
	svc      *podcast.Service
	podcasts []podcast.PodcastInfo
	cursor   int
	loading  bool
	err      error
}

type podcastsLoadedMsg struct {
	podcasts []podcast.PodcastInfo
	err      error
}

type podcastDeletedMsg struct{ err error }

func newPodcastsList(svc *podcast.Service) *podcastsModel {
	return &podcastsModel{svc: svc, loading: true}
}

func (m *podcastsModel) Init() tea.Cmd {
	if m.svc == nil {
		return nil
	}
	return m.loadPodcasts()
}

func (m *podcastsModel) loadPodcasts() tea.Cmd {
	return func() tea.Msg {
		podcasts, err := m.svc.List(context.Background())
		return podcastsLoadedMsg{podcasts: podcasts, err: err}
	}
}

func (m *podcastsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case podcastsLoadedMsg:
		m.loading = false
		m.podcasts = msg.podcasts
		m.err = msg.err
		if m.cursor >= len(m.podcasts) {
			m.cursor = max(0, len(m.podcasts)-1)
		}

	case podcastDeletedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			return m, m.loadPodcasts()
		}

	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.podcasts)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.podcasts) > 0 {
				slug := m.podcasts[m.cursor].Slug
				return m, func() tea.Msg {
					return switchScreenMsg{screen: screenEpisodes, data: slug}
				}
			}
		case "n":
			return m, func() tea.Msg {
				return switchScreenMsg{screen: screenPodcastForm, data: ""}
			}
		case "e":
			if len(m.podcasts) > 0 {
				slug := m.podcasts[m.cursor].Slug
				return m, func() tea.Msg {
					return switchScreenMsg{screen: screenPodcastForm, data: slug}
				}
			}
		case "d":
			if len(m.podcasts) > 0 {
				slug := m.podcasts[m.cursor].Slug
				return m, func() tea.Msg {
					err := m.svc.Delete(context.Background(), slug)
					return podcastDeletedMsg{err: err}
				}
			}
		case "s":
			return m, func() tea.Msg {
				return switchScreenMsg{screen: screenConfigForm}
			}
		}
	}

	return m, nil
}

func (m *podcastsModel) View() string {
	s := titleStyle.Render("podvid — Video to Podcast") + "\n\n"

	if m.svc == nil {
		s += errorStyle.Render("S3 not configured. Press 's' to configure.") + "\n"
		s += helpStyle.Render("[s] Settings  [q] Quit")
		return s
	}

	if m.loading {
		s += subtitleStyle.Render("Loading podcasts...") + "\n"
		return s
	}

	if m.err != nil {
		s += errorStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n\n"
	}

	if len(m.podcasts) == 0 {
		s += subtitleStyle.Render("No podcasts yet. Press 'n' to create one.") + "\n"
	} else {
		s += subtitleStyle.Render("Podcasts:") + "\n\n"
		for i, p := range m.podcasts {
			epWord := episodeWord(p.EpisodeCount)
			line := fmt.Sprintf("%-30s %d %s", p.Name, p.EpisodeCount, epWord)
			if i == m.cursor {
				s += selectedStyle.Render("> "+line) + "\n"
			} else {
				s += normalStyle.Render(line) + "\n"
			}
		}
	}

	s += "\n" + helpStyle.Render("[n] New  [d] Delete  [e] Edit  [enter] Episodes  [s] Settings  [q] Quit")
	return s
}

func episodeWord(n int) string {
	if n%10 == 1 && n%100 != 11 {
		return "episode"
	}
	return "episodes"
}
