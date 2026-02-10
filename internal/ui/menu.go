package ui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

type menuKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Select key.Binding
}

var menuKeys = menuKeyMap{
	Up:     key.NewBinding(key.WithKeys("up", "k")),
	Down:   key.NewBinding(key.WithKeys("down", "j")),
	Select: key.NewBinding(key.WithKeys("enter")),
}

type MenuItem struct {
	Label    string
	Key      int
	Shortcut key.Binding
}

type MenuSelectMsg struct{ Key int }

type Menu struct {
	items    []MenuItem
	selected int
	prefix   string
	padWidth int
}

func NewMenu(prefix string, items ...MenuItem) Menu {
	m := Menu{
		items:  items,
		prefix: prefix,
	}
	m.measureItems()
	return m
}

func (m Menu) Update(msg tea.Msg) (Menu, tea.Cmd) {
	count := len(m.items)
	if count == 0 {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			for i := range m.items {
				if zi := zone.Get(m.zoneID(i)); zi != nil && zi.InBounds(msg) {
					m.selected = i
					return m, m.selectItem(m.items[i].Key)
				}
			}
		}

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, menuKeys.Up):
			m.selected = (m.selected - 1 + count) % count
		case key.Matches(msg, menuKeys.Down):
			m.selected = (m.selected + 1) % count
		case key.Matches(msg, menuKeys.Select):
			return m, m.selectItem(m.items[m.selected].Key)
		default:
			for i, item := range m.items {
				if key.Matches(msg, item.Shortcut) {
					m.selected = i
					return m, m.selectItem(item.Key)
				}
			}
		}
	}

	return m, nil
}

func (m Menu) View() string {
	itemStyle := lipgloss.NewStyle()
	selectedStyle := lipgloss.NewStyle().Reverse(true)
	keyStyle := lipgloss.NewStyle().Foreground(Colors.Border)

	lines := make([]string, len(m.items))
	for i, item := range m.items {
		padding := strings.Repeat(" ", m.padWidth-len(item.Label))
		shortcutKeys := item.Shortcut.Keys()
		shortcutStr := ""
		if len(shortcutKeys) > 0 {
			shortcutStr = shortcutKeys[0]
		}
		styledKey := keyStyle.Render(shortcutStr)

		var line string
		if m.selected == i {
			line = selectedStyle.Render(item.Label) + padding + styledKey
		} else {
			line = itemStyle.Render(item.Label) + padding + styledKey
		}
		lines[i] = zone.Mark(m.zoneID(i), line)
	}

	return strings.Join(lines, "\n")
}

// Private

func (m Menu) zoneID(index int) string {
	return fmt.Sprintf("%s_%d", m.prefix, index)
}

func (m *Menu) measureItems() {
	maxLen := 0
	for _, item := range m.items {
		if len(item.Label) > maxLen {
			maxLen = len(item.Label)
		}
	}
	m.padWidth = maxLen + 2
}

func (m Menu) selectItem(key int) tea.Cmd {
	return func() tea.Msg { return MenuSelectMsg{Key: key} }
}
