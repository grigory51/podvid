package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/grigory51/podvid/internal/config"
)

type configFormModel struct {
	cfg     *config.Config
	cfgPath string

	inputs  []textinput.Model
	labels  []string
	focused int

	done bool
	err  error
}

type configSavedMsg struct{ err error }

func newConfigForm(cfg *config.Config, cfgPath string) *configFormModel {
	labels := []string{
		"S3 Endpoint (empty for AWS)",
		"S3 Region",
		"S3 Bucket",
		"S3 Access Key",
		"S3 Secret Key",
		"Public Base URL",
		"Audio Bitrate",
	}

	values := []string{
		cfg.S3.Endpoint,
		cfg.S3.Region,
		cfg.S3.Bucket,
		cfg.S3.AccessKey,
		cfg.S3.SecretKey,
		cfg.S3.PublicBaseURL,
		cfg.Audio.Bitrate,
	}

	var inputs []textinput.Model
	for i, label := range labels {
		ti := textinput.New()
		ti.Placeholder = label
		ti.CharLimit = 200
		ti.Width = 50
		if i < len(values) && values[i] != "" {
			ti.SetValue(values[i])
		}
		if i == 0 {
			ti.Focus()
		}
		inputs = append(inputs, ti)
	}

	return &configFormModel{
		cfg:     cfg,
		cfgPath: cfgPath,
		inputs:  inputs,
		labels:  labels,
	}
}

func (m *configFormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *configFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case configSavedMsg:
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
		case "tab", "down":
			m.inputs[m.focused].Blur()
			m.focused = (m.focused + 1) % len(m.inputs)
			m.inputs[m.focused].Focus()
			return m, nil
		case "shift+tab", "up":
			m.inputs[m.focused].Blur()
			m.focused = (m.focused - 1 + len(m.inputs)) % len(m.inputs)
			m.inputs[m.focused].Focus()
			return m, nil
		case "enter":
			if m.focused == len(m.inputs)-1 {
				return m, m.save()
			}
			m.inputs[m.focused].Blur()
			m.focused++
			m.inputs[m.focused].Focus()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
	return m, cmd
}

func (m *configFormModel) save() tea.Cmd {
	return func() tea.Msg {
		m.cfg.S3.Endpoint = m.inputs[0].Value()
		m.cfg.S3.Region = m.inputs[1].Value()
		m.cfg.S3.Bucket = m.inputs[2].Value()
		m.cfg.S3.AccessKey = m.inputs[3].Value()
		m.cfg.S3.SecretKey = m.inputs[4].Value()
		m.cfg.S3.PublicBaseURL = m.inputs[5].Value()
		m.cfg.Audio.Bitrate = m.inputs[6].Value()

		if m.cfg.Audio.Bitrate == "" {
			m.cfg.Audio.Bitrate = "192k"
		}

		path := m.cfgPath
		if path == "" {
			path = config.DefaultConfigPath()
		}

		err := m.cfg.Save(path)
		return configSavedMsg{err: err}
	}
}

func (m *configFormModel) View() string {
	s := titleStyle.Render("S3 Configuration") + "\n\n"

	if m.err != nil {
		s += errorStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n\n"
	}

	for i, input := range m.inputs {
		s += inputLabelStyle.Render(m.labels[i]+":") + "\n"
		s += "  " + input.View() + "\n\n"
	}

	s += helpStyle.Render("[tab/down] Next  [shift+tab/up] Prev  [enter] Save  [esc] Cancel")
	return s
}
