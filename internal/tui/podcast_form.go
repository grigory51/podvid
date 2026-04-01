package tui

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/grigory51/podvid/internal/podcast"
)

const (
	fieldName = iota
	fieldDesc
	fieldCover
	fieldCount
)

type podcastFormModel struct {
	svc      *podcast.Service
	editSlug string // empty = create mode

	nameInput    textinput.Model
	descInput    textinput.Model
	coverInput   textinput.Model
	focused      int
	currentCover string // URL текущей обложки (только в режиме редактирования)
	removeCover  bool   // пользователь хочет удалить обложку

	done bool
	err  error
}

type podcastSavedMsg struct {
	err error
}

func newPodcastForm(svc *podcast.Service, editSlug string) *podcastFormModel {
	nameInput := textinput.New()
	nameInput.Placeholder = "Podcast name"
	nameInput.Focus()
	nameInput.CharLimit = 100
	nameInput.Width = 50

	descInput := textinput.New()
	descInput.Placeholder = "Description (optional)"
	descInput.CharLimit = 500
	descInput.Width = 50

	coverInput := textinput.New()
	coverInput.Placeholder = "/path/to/cover.jpg (optional, leave empty to keep current)"
	coverInput.CharLimit = 300
	coverInput.Width = 50

	m := &podcastFormModel{
		svc:        svc,
		editSlug:   editSlug,
		nameInput:  nameInput,
		descInput:  descInput,
		coverInput: coverInput,
	}

	if editSlug != "" && svc != nil {
		info, err := svc.Get(context.Background(), editSlug)
		if err == nil {
			m.nameInput.SetValue(info.Name)
			m.descInput.SetValue(info.Description)
			m.currentCover = info.CoverURL
		}
	}

	return m
}

func (m *podcastFormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *podcastFormModel) focusField(field int) {
	m.focused = field
	m.nameInput.Blur()
	m.descInput.Blur()
	m.coverInput.Blur()
	switch field {
	case fieldName:
		m.nameInput.Focus()
	case fieldDesc:
		m.descInput.Focus()
	case fieldCover:
		m.coverInput.Focus()
	}
}

func (m *podcastFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case podcastSavedMsg:
		m.done = true
		m.err = msg.err
		if msg.err == nil {
			return m, func() tea.Msg {
				return switchScreenMsg{screen: screenPodcasts}
			}
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg {
				return switchScreenMsg{screen: screenPodcasts}
			}
		case "ctrl+d":
			if m.focused == fieldCover && m.currentCover != "" {
				m.removeCover = !m.removeCover
				m.coverInput.SetValue("")
				return m, nil
			}
		case "tab":
			m.focusField((m.focused + 1) % fieldCount)
			return m, nil
		case "shift+tab":
			m.focusField((m.focused - 1 + fieldCount) % fieldCount)
			return m, nil
		case "enter":
			if m.focused < fieldCover {
				m.focusField(m.focused + 1)
				return m, nil
			}
			return m, m.save()
		}
	}

	var cmd tea.Cmd
	switch m.focused {
	case fieldName:
		m.nameInput, cmd = m.nameInput.Update(msg)
	case fieldDesc:
		m.descInput, cmd = m.descInput.Update(msg)
	case fieldCover:
		if !m.removeCover {
			m.coverInput, cmd = m.coverInput.Update(msg)
		}
	}
	return m, cmd
}

func (m *podcastFormModel) save() tea.Cmd {
	return func() tea.Msg {
		name := m.nameInput.Value()
		desc := m.descInput.Value()
		coverPath := strings.TrimSpace(m.coverInput.Value())

		if name == "" {
			return podcastSavedMsg{err: fmt.Errorf("name is required")}
		}

		ctx := context.Background()
		slug := m.editSlug

		if slug == "" {
			info, err := m.svc.Create(ctx, name, desc)
			if err != nil {
				return podcastSavedMsg{err: err}
			}
			slug = info.Slug
		} else {
			if err := m.svc.Edit(ctx, slug, &name, &desc); err != nil {
				return podcastSavedMsg{err: err}
			}
		}

		if m.removeCover {
			if err := m.svc.RemoveCover(ctx, slug); err != nil {
				return podcastSavedMsg{err: fmt.Errorf("removing cover: %w", err)}
			}
		} else if coverPath != "" {
			data, err := os.ReadFile(coverPath)
			if err != nil {
				return podcastSavedMsg{err: fmt.Errorf("reading cover: %w", err)}
			}
			ct := "image/jpeg"
			lower := strings.ToLower(coverPath)
			if strings.HasSuffix(lower, ".png") {
				ct = "image/png"
			} else if strings.HasSuffix(lower, ".webp") {
				ct = "image/webp"
			}
			if err := m.svc.SetCover(ctx, slug, data, ct); err != nil {
				return podcastSavedMsg{err: fmt.Errorf("uploading cover: %w", err)}
			}
		}
		// пустой coverPath + !removeCover = не трогаем обложку

		return podcastSavedMsg{err: nil}
	}
}

func (m *podcastFormModel) View() string {
	action := "Create"
	if m.editSlug != "" {
		action = "Edit"
	}
	s := titleStyle.Render(fmt.Sprintf("%s Podcast", action)) + "\n\n"

	if m.err != nil {
		s += errorStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n\n"
	}

	s += inputLabelStyle.Render("Name:") + "\n"
	s += "  " + m.nameInput.View() + "\n\n"
	s += inputLabelStyle.Render("Description:") + "\n"
	s += "  " + m.descInput.View() + "\n\n"

	s += inputLabelStyle.Render("Cover image:") + "\n"
	if m.removeCover {
		s += "  " + errorStyle.Render("Will be removed") + "\n"
	} else {
		s += "  " + m.coverInput.View() + "\n"
	}
	if m.currentCover != "" && !m.removeCover {
		s += subtitleStyle.Render(fmt.Sprintf("  Current: %s", m.currentCover)) + "\n"
	}

	help := "[tab] Next field  [enter] Save  [esc] Cancel"
	if m.focused == fieldCover && m.currentCover != "" {
		if m.removeCover {
			help = "[ctrl+d] Undo remove  [tab] Next field  [enter] Save  [esc] Cancel"
		} else {
			help = "[ctrl+d] Remove cover  [tab] Next field  [enter] Save  [esc] Cancel"
		}
	}
	s += "\n" + helpStyle.Render(help)
	return s
}
