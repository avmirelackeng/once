package ui

import (
	"context"
	"fmt"
	"image/color"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/basecamp/amar/internal/docker"
)

type dashboardKeyMap struct {
	Upgrade key.Binding
	PrevApp key.Binding
	NextApp key.Binding
	Quit    key.Binding
}

func (k dashboardKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.PrevApp, k.NextApp, k.Upgrade, k.Quit}
}

func (k dashboardKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.PrevApp, k.NextApp, k.Upgrade, k.Quit}}
}

var dashboardKeys = dashboardKeyMap{
	Upgrade: key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "upgrade")),
	PrevApp: key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "prev app")),
	NextApp: key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "next app")),
	Quit:    key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "quit")),
}

type Dashboard struct {
	app           *docker.Application
	width, height int
	upgrading     bool
	progress      ProgressBusy
	help          help.Model
}

type dashboardTickMsg struct{}

type upgradeFinishedMsg struct {
	err error
}

func NewDashboard(app *docker.Application) Dashboard {
	return Dashboard{
		app:  app,
		help: help.New(),
	}
}

func (m Dashboard) Init() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return dashboardTickMsg{} })
}

func (m Dashboard) Update(msg tea.Msg) (Component, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.progress = NewProgressBusy(m.width, lipgloss.Color("#6272a4"))
		m.help.SetWidth(m.width)
		if m.upgrading {
			cmds = append(cmds, m.progress.Init())
		}
	case tea.KeyMsg:
		if key.Matches(msg, dashboardKeys.Upgrade) && !m.upgrading {
			m.upgrading = true
			m.progress = NewProgressBusy(m.width, lipgloss.Color("#6272a4"))
			return m, tea.Batch(m.progress.Init(), m.runUpgrade())
		}
	case upgradeFinishedMsg:
		m.upgrading = false
	case dashboardTickMsg:
		cmds = append(cmds, tea.Tick(time.Second, func(time.Time) tea.Msg { return dashboardTickMsg{} }))
	case progressBusyTickMsg:
		if m.upgrading {
			var cmd tea.Cmd
			m.progress, cmd = m.progress.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m Dashboard) View() string {
	title := Styles.Title.Width(m.width).Align(lipgloss.Center).Render(m.app.Settings.Name)

	var status string
	var statusColor color.Color
	if m.upgrading {
		status = "upgrading..."
		statusColor = lipgloss.Color("#f1fa8c")
	} else if m.app.Running {
		status = "running"
		statusColor = lipgloss.Color("#50fa7b")
	} else {
		status = "stopped"
		statusColor = lipgloss.Color("#ff5555")
	}

	stateStyle := lipgloss.NewStyle().Foreground(statusColor)
	stateDisplay := fmt.Sprintf("State: %s", stateStyle.Render(status))

	if m.app.Running && !m.app.RunningSince.IsZero() && !m.upgrading {
		stateDisplay += fmt.Sprintf(" (up %s)", formatDuration(time.Since(m.app.RunningSince)))
	}

	content := lipgloss.NewStyle().PaddingLeft(2).Render(stateDisplay)

	// Help string (last line, centered)
	helpView := m.help.View(dashboardKeys)
	helpLine := lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(helpView)

	// Progress bar (second-to-last line, only during upgrade)
	var bottomContent string
	if m.upgrading {
		bottomContent = m.progress.View() + "\n" + helpLine
	} else {
		bottomContent = helpLine
	}

	// Calculate available height for main content
	topContent := title + "\n\n" + content

	topHeight := lipgloss.Height(topContent)
	bottomHeight := lipgloss.Height(bottomContent)
	middleHeight := max(m.height-topHeight-bottomHeight, 0)

	middle := strings.Repeat("\n", middleHeight)

	return topContent + middle + bottomContent
}

// Private

func (m Dashboard) runUpgrade() tea.Cmd {
	return func() tea.Msg {
		err := m.app.Update(context.Background(), nil)
		return upgradeFinishedMsg{err: err}
	}
}

// Helpers

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		mins := int(d.Minutes()) % 60
		if mins == 0 {
			return fmt.Sprintf("%dh", hours)
		}
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	if hours == 0 {
		return fmt.Sprintf("%dd", days)
	}
	return fmt.Sprintf("%dd %dh", days, hours)
}
