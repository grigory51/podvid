package tui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/grigory51/podvid/internal/podcast"
)

const (
	epFieldTitle = iota
	epFieldDesc
	epFieldCount
)

type episodeFormData struct {
	slug    string
	episode podcast.EpisodeInfo
}

type episodeFormModel struct {
	svc     *podcast.Service
	slug    string
	episode podcast.EpisodeInfo

	titleInput textinput.Model
	descInput  textarea.Model
	focused    int

	done bool
	err  error
}

type episodeSavedMsg struct{ err error }

func newEpisodeForm(svc *podcast.Service, slug string, ep podcast.EpisodeInfo) *episodeFormModel {
	ti := textinput.New()
	ti.Placeholder = "Episode title"
	ti.Focus()
	ti.CharLimit = 200
	ti.Width = 60
	ti.SetValue(ep.Title)

	ta := textarea.New()
	ta.Placeholder = "Episode description (optional)"
	ta.SetWidth(60)
	ta.SetHeight(6)
	ta.SetValue(ep.Description)

	return &episodeFormModel{
		svc:        svc,
		slug:       slug,
		episode:    ep,
		titleInput: ti,
		descInput:  ta,
	}
}

func (m *episodeFormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *episodeFormModel) focusField(field int) {
	m.focused = field
	m.titleInput.Blur()
	m.descInput.Blur()
	switch field {
	case epFieldTitle:
		m.titleInput.Focus()
	case epFieldDesc:
		m.descInput.Focus()
	}
}

func (m *episodeFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case episodeSavedMsg:
		m.done = true
		m.err = msg.err
		if msg.err == nil {
			return m, func() tea.Msg {
				return switchScreenMsg{screen: screenEpisodes, data: m.slug}
			}
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg {
				return switchScreenMsg{screen: screenEpisodes, data: m.slug}
			}
		case "tab":
			m.focusField((m.focused + 1) % epFieldCount)
			return m, nil
		case "shift+tab":
			m.focusField((m.focused - 1 + epFieldCount) % epFieldCount)
			return m, nil
		case "ctrl+s":
			return m, m.save()
		case "enter":
			if m.focused == epFieldTitle {
				m.focusField(epFieldDesc)
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	switch m.focused {
	case epFieldTitle:
		m.titleInput, cmd = m.titleInput.Update(msg)
	case epFieldDesc:
		m.descInput, cmd = m.descInput.Update(msg)
	}
	return m, cmd
}

func (m *episodeFormModel) save() tea.Cmd {
	return func() tea.Msg {
		title := m.titleInput.Value()
		desc := m.descInput.Value()

		if title == "" {
			return episodeSavedMsg{err: fmt.Errorf("title is required")}
		}

		err := m.svc.EditEpisode(context.Background(), m.slug, m.episode.ID, &title, &desc)
		return episodeSavedMsg{err: err}
	}
}

func (m *episodeFormModel) View() string {
	s := titleStyle.Render(fmt.Sprintf("Edit Episode — %s", m.slug)) + "\n\n"

	if m.err != nil {
		s += errorStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n\n"
	}

	s += inputLabelStyle.Render("Title:") + "\n"
	s += "  " + m.titleInput.View() + "\n\n"
	s += inputLabelStyle.Render("Description:") + "\n"
	s += "  " + m.descInput.View() + "\n"

	s += "\n" + helpStyle.Render("[tab] Next field  [ctrl+s] Save  [esc] Cancel")
	return s
}
