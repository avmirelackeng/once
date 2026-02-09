package ui

import (
	"strconv"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/basecamp/once/internal/docker"
)

type resourcesFormField int

const (
	resourcesFieldCPU resourcesFormField = iota
	resourcesFieldMemory
	resourcesFieldDoneButton
	resourcesFieldCancelButton
	resourcesFieldCount
)

type SettingsFormResources struct {
	width, height int
	focused       resourcesFormField
	settings      docker.ApplicationSettings
	cpuInput      textinput.Model
	memoryInput   textinput.Model
}

func NewSettingsFormResources(settings docker.ApplicationSettings) SettingsFormResources {
	cpu := textinput.New()
	cpu.Placeholder = "e.g. 2"
	cpu.Prompt = ""
	cpu.CharLimit = 10
	if settings.Resources.CPUs != 0 {
		cpu.SetValue(strconv.Itoa(settings.Resources.CPUs))
	}
	cpu.Focus()

	memory := textinput.New()
	memory.Placeholder = "e.g. 512"
	memory.Prompt = ""
	memory.CharLimit = 10
	if settings.Resources.MemoryMB != 0 {
		memory.SetValue(strconv.Itoa(settings.Resources.MemoryMB))
	}

	return SettingsFormResources{
		focused:     resourcesFieldCPU,
		settings:    settings,
		cpuInput:    cpu,
		memoryInput: memory,
	}
}

func (m SettingsFormResources) Title() string {
	return "Resources"
}

func (m SettingsFormResources) Init() tea.Cmd {
	return nil
}

func (m SettingsFormResources) Update(msg tea.Msg) (SettingsSection, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		inputWidth := min(m.width-4, 60)
		m.cpuInput.SetWidth(inputWidth)
		m.memoryInput.SetWidth(inputWidth)

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
			return m.focusNext()
		case key.Matches(msg, key.NewBinding(key.WithKeys("shift+tab"))):
			return m.focusPrev()
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			return m.handleEnter()
		}
	}

	switch m.focused {
	case resourcesFieldCPU:
		var cmd tea.Cmd
		m.cpuInput, cmd = m.cpuInput.Update(msg)
		cmds = append(cmds, cmd)
	case resourcesFieldMemory:
		var cmd tea.Cmd
		m.memoryInput, cmd = m.memoryInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m SettingsFormResources) View() string {
	cpuLabel := Styles.Label.Render("CPU Limit")
	cpuField := Styles.Focus(Styles.Input, m.focused == resourcesFieldCPU).
		Render(m.cpuInput.View())

	memoryLabel := Styles.Label.Render("Memory Limit (MB)")
	memoryField := Styles.Focus(Styles.Input, m.focused == resourcesFieldMemory).
		Render(m.memoryInput.View())

	doneButton := Styles.Focus(Styles.ButtonPrimary, m.focused == resourcesFieldDoneButton).
		Render("Done")
	cancelButton := Styles.Focus(Styles.Button, m.focused == resourcesFieldCancelButton).
		Render("Cancel")

	buttons := lipgloss.JoinHorizontal(lipgloss.Center, doneButton, cancelButton)

	form := lipgloss.JoinVertical(lipgloss.Left,
		cpuLabel,
		cpuField,
		memoryLabel,
		memoryField,
		"",
		buttons,
	)

	return form
}

// Private

func (m SettingsFormResources) focusNext() (SettingsSection, tea.Cmd) {
	m.blurCurrent()
	m.focused = (m.focused + 1) % resourcesFieldCount
	return m.focusCurrent()
}

func (m SettingsFormResources) focusPrev() (SettingsSection, tea.Cmd) {
	m.blurCurrent()
	m.focused = (m.focused - 1 + resourcesFieldCount) % resourcesFieldCount
	return m.focusCurrent()
}

func (m *SettingsFormResources) blurCurrent() {
	switch m.focused {
	case resourcesFieldCPU:
		m.cpuInput.Blur()
	case resourcesFieldMemory:
		m.memoryInput.Blur()
	}
}

func (m SettingsFormResources) focusCurrent() (SettingsSection, tea.Cmd) {
	var cmd tea.Cmd
	switch m.focused {
	case resourcesFieldCPU:
		cmd = m.cpuInput.Focus()
	case resourcesFieldMemory:
		cmd = m.memoryInput.Focus()
	}
	return m, cmd
}

func (m SettingsFormResources) handleEnter() (SettingsSection, tea.Cmd) {
	switch m.focused {
	case resourcesFieldCPU, resourcesFieldMemory:
		return m.focusNext()
	case resourcesFieldDoneButton:
		if v, err := strconv.Atoi(m.cpuInput.Value()); err == nil {
			m.settings.Resources.CPUs = v
		} else {
			m.settings.Resources.CPUs = 0
		}

		if v, err := strconv.Atoi(m.memoryInput.Value()); err == nil {
			m.settings.Resources.MemoryMB = v
		} else {
			m.settings.Resources.MemoryMB = 0
		}

		return m, func() tea.Msg { return SettingsSectionSubmitMsg{Settings: m.settings} }
	case resourcesFieldCancelButton:
		return m, func() tea.Msg { return SettingsSectionCancelMsg{} }
	}
	return m, nil
}
