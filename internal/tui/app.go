package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/grigory51/podvid/internal/config"
	"github.com/grigory51/podvid/internal/podcast"
	"github.com/grigory51/podvid/internal/storage"
)

type screen int

const (
	screenPodcasts screen = iota
	screenEpisodes
	screenAddEpisode
	screenPodcastForm
	screenEpisodeForm
	screenConfigForm
)

type appModel struct {
	cfg     *config.Config
	cfgPath string
	svc     *podcast.Service
	s3      *storage.S3Client

	current screen
	screens map[screen]tea.Model

	width  int
	height int
	err    error
}

type switchScreenMsg struct {
	screen screen
	data   interface{}
}

type errMsg struct{ err error }

func Run(cfg *config.Config, cfgPath string) error {
	m := newAppModel(cfg, cfgPath)

	if !cfg.IsS3Configured() {
		m.current = screenConfigForm
		m.screens[screenConfigForm] = newConfigForm(cfg, cfgPath)
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func newAppModel(cfg *config.Config, cfgPath string) appModel {
	m := appModel{
		cfg:     cfg,
		cfgPath: cfgPath,
		current: screenPodcasts,
		screens: make(map[screen]tea.Model),
	}

	if cfg.IsS3Configured() {
		m.initService()
	}

	m.screens[screenPodcasts] = newPodcastsList(m.svc)
	return m
}

func (m *appModel) initService() {
	s3Client, err := storage.NewS3Client(m.cfg)
	if err != nil {
		m.err = err
		return
	}
	m.s3 = s3Client
	m.svc = podcast.NewService(s3Client, m.cfg)
}

func (m appModel) Init() tea.Cmd {
	if s, ok := m.screens[m.current]; ok {
		return s.Init()
	}
	return nil
}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case switchScreenMsg:
		return m.handleScreenSwitch(msg)
	case errMsg:
		m.err = msg.err
	}

	if s, ok := m.screens[m.current]; ok {
		updated, cmd := s.Update(msg)
		m.screens[m.current] = updated
		return m, cmd
	}
	return m, nil
}

func (m appModel) View() string {
	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v\n\nPress q to quit.", m.err))
	}
	if s, ok := m.screens[m.current]; ok {
		return s.View()
	}
	return ""
}

func (m appModel) handleScreenSwitch(msg switchScreenMsg) (tea.Model, tea.Cmd) {
	m.current = msg.screen

	switch msg.screen {
	case screenPodcasts:
		m.screens[screenPodcasts] = newPodcastsList(m.svc)
		return m, m.screens[screenPodcasts].Init()

	case screenEpisodes:
		if slug, ok := msg.data.(string); ok {
			m.screens[screenEpisodes] = newEpisodesList(m.svc, slug)
			return m, m.screens[screenEpisodes].Init()
		}

	case screenAddEpisode:
		if slug, ok := msg.data.(string); ok {
			m.screens[screenAddEpisode] = newAddEpisode(m.svc, m.cfg, slug)
			return m, m.screens[screenAddEpisode].Init()
		}

	case screenPodcastForm:
		var slug string
		if s, ok := msg.data.(string); ok {
			slug = s
		}
		m.screens[screenPodcastForm] = newPodcastForm(m.svc, slug)
		return m, m.screens[screenPodcastForm].Init()

	case screenEpisodeForm:
		if data, ok := msg.data.(episodeFormData); ok {
			m.screens[screenEpisodeForm] = newEpisodeForm(m.svc, data.slug, data.episode)
			return m, m.screens[screenEpisodeForm].Init()
		}

	case screenConfigForm:
		m.screens[screenConfigForm] = newConfigForm(m.cfg, m.cfgPath)
		return m, m.screens[screenConfigForm].Init()
	}

	return m, nil
}
