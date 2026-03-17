package dashboard

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/idle"
	"github.com/devlikebear/break-reminder/internal/launchd"
	"github.com/devlikebear/break-reminder/internal/logging"
	"github.com/devlikebear/break-reminder/internal/schedule"
	"github.com/devlikebear/break-reminder/internal/state"
)

type tickMsg time.Time

type Model struct {
	cfg     config.Config
	state   state.State
	idleSec int
	logs    []string
	width   int
	height  int

	// Break activity overlay
	showBreakMenu    bool
	breakMenuCursor  int
	breakActivity    tea.Model
	showingActivity  bool
}

func New(cfg config.Config) Model {
	s, _ := state.Load(state.DefaultStatePath())
	return Model{
		cfg:   cfg,
		state: s,
		logs:  logging.Tail(logging.DefaultLogPath(), 5),
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Init() tea.Cmd {
	return tickCmd()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// If showing a break activity, delegate
	if m.showingActivity && m.breakActivity != nil {
		updated, cmd := m.breakActivity.Update(msg)
		m.breakActivity = updated
		// Check if the activity is done (we'll use a done message)
		if _, ok := msg.(tea.KeyMsg); ok {
			if msg.(tea.KeyMsg).String() == "esc" {
				m.showingActivity = false
				m.breakActivity = nil
				return m, tickCmd()
			}
		}
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		if newCfg, err := config.Load(); err == nil {
			m.cfg = newCfg
		}
		m.state, _ = state.Load(state.DefaultStatePath())
		m.idleSec = idle.NewDetector().IdleSeconds()
		m.logs = logging.Tail(logging.DefaultLogPath(), 5)

		// Show break menu when entering break mode
		if m.state.Mode == "break" && !m.showBreakMenu && !m.showingActivity && m.cfg.BreakActivitiesEnabled {
			m.showBreakMenu = true
			m.breakMenuCursor = 0
		} else if m.state.Mode == "work" {
			m.showBreakMenu = false
		}

		return m, tickCmd()

	case tea.KeyMsg:
		if m.showBreakMenu {
			return m.handleBreakMenu(msg)
		}

		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, keys.Reset):
			s := state.New()
			s.LastCheck = time.Now().Unix()
			_ = state.Save(state.DefaultStatePath(), s)
			m.state = s
		case key.Matches(msg, keys.Break):
			// Force break mode
			m.state.Mode = "break"
			m.state.BreakStart = time.Now().Unix()
			m.state.WorkSeconds = 0
			_ = state.Save(state.DefaultStatePath(), m.state)
			if m.cfg.BreakActivitiesEnabled {
				m.showBreakMenu = true
				m.breakMenuCursor = 0
			}
		}
	}

	return m, nil
}

func (m Model) handleBreakMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.breakMenuCursor > 0 {
			m.breakMenuCursor--
		}
	case "down", "j":
		if m.breakMenuCursor < 4 {
			m.breakMenuCursor++
		}
	case "enter":
		m.showBreakMenu = false
		switch m.breakMenuCursor {
		case 0: // eye
			m.showingActivity = true
			m.breakActivity = NewEyeActivity()
			return m, m.breakActivity.Init()
		case 1: // stretch
			m.showingActivity = true
			m.breakActivity = NewStretchActivity()
			return m, m.breakActivity.Init()
		case 2: // breathe
			m.showingActivity = true
			m.breakActivity = NewBreatheActivity()
			return m, m.breakActivity.Init()
		case 3: // walk
			m.showingActivity = true
			m.breakActivity = NewWalkActivity()
			return m, m.breakActivity.Init()
		case 4: // skip
			// do nothing
		}
	case "esc":
		m.showBreakMenu = false
	}
	return m, nil
}

func (m Model) View() string {
	if m.showingActivity && m.breakActivity != nil {
		return m.breakActivity.View()
	}

	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")) // cyan
	greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	blueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	yellowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))

	b.WriteString(titleStyle.Render("🐹 Break Reminder Dashboard") + " (q:quit r:reset b:break)\n")
	b.WriteString("══════════════════════════════════════════════════\n")

	// System status
	b.WriteString("System: " + launchd.Status() + "\n")

	now := time.Now()
	if !schedule.IsWorkingTime(m.cfg, now) {
		b.WriteString("Status: " + yellowStyle.Render("SLEEPING (Outside Working Hours)") + "\n")
	} else if m.state.Mode == "work" {
		b.WriteString("Status: " + greenStyle.Render("WORKING") + "\n")
	} else {
		b.WriteString("Status: " + blueStyle.Render("ON BREAK") + "\n")
	}

	b.WriteString(fmt.Sprintf("Idle: %ds / Limit: %ds\n\n", m.idleSec, m.cfg.IdleThresholdSec))

	// Progress bar
	if m.state.Mode == "work" {
		workDur := m.cfg.WorkDurationSec()
		pct := 0
		if workDur > 0 {
			pct = (m.state.WorkSeconds * 100) / workDur
		}
		if pct > 100 {
			pct = 100
		}
		bar := renderBar(pct, 30, greenStyle)
		b.WriteString(fmt.Sprintf("Session Work: %s (%d / %d min)\n",
			bar, m.state.WorkSeconds/60, m.cfg.WorkDurationMin))
	} else {
		breakDur := m.cfg.BreakDurationSec()
		breakElapsed := int(now.Unix() - m.state.BreakStart)
		pct := 0
		if breakDur > 0 {
			pct = (breakElapsed * 100) / breakDur
		}
		if pct > 100 {
			pct = 100
		}
		bar := renderBar(pct, 30, blueStyle)
		b.WriteString(fmt.Sprintf("Break Timer:  %s (%d / %d min)\n",
			bar, breakElapsed/60, m.cfg.BreakDurationMin))
	}

	b.WriteString("\n")

	// Daily stats
	dailyWorkMin := m.state.TodayWorkSeconds / 60
	dailyBreakMin := m.state.TodayBreakSeconds / 60
	totalMin := dailyWorkMin + dailyBreakMin

	b.WriteString("Daily Statistics:\n")
	b.WriteString(fmt.Sprintf("  Work: %d min\n", dailyWorkMin))
	b.WriteString(fmt.Sprintf("  Rest: %d min\n", dailyBreakMin))
	if totalMin > 0 {
		ratio := (dailyWorkMin * 100) / totalMin
		bar := renderBar(ratio, 20, yellowStyle)
		b.WriteString(fmt.Sprintf("  Ratio: %s\n", bar))
	}

	b.WriteString("\n")

	// Logs
	b.WriteString("Recent Logs:\n")
	b.WriteString("──────────────────────────────────────────────────\n")
	if len(m.logs) == 0 {
		b.WriteString("  (No logs yet)\n")
	} else {
		for _, line := range m.logs {
			b.WriteString("  " + line + "\n")
		}
	}
	b.WriteString("──────────────────────────────────────────────────\n")

	// Break menu overlay
	if m.showBreakMenu {
		b.WriteString("\n")
		b.WriteString(blueStyle.Render("🧘 휴식 시간! 활동을 선택하세요 (Esc로 건너뛰기):") + "\n")
		items := []string{
			"👁  눈 운동 (20-20-20 규칙) - 2분",
			"🤸 스트레칭 - 5분",
			"🌬  호흡 운동 - 4분",
			"🚶 산책 - 5분",
			"⏭  건너뛰기",
		}
		for i, item := range items {
			if i == m.breakMenuCursor {
				b.WriteString("  > " + item + "\n")
			} else {
				b.WriteString("    " + item + "\n")
			}
		}
	}

	return b.String()
}

func renderBar(pct, length int, style lipgloss.Style) string {
	fillLen := (pct * length) / 100
	emptyLen := length - fillLen

	fill := style.Render(strings.Repeat("█", fillLen))
	empty := strings.Repeat("░", emptyLen)

	return fmt.Sprintf("[%s%s] %d%%", fill, empty, pct)
}

// Key bindings
type keyMap struct {
	Quit  key.Binding
	Reset key.Binding
	Break key.Binding
}

var keys = keyMap{
	Quit:  key.NewBinding(key.WithKeys("q", "ctrl+c")),
	Reset: key.NewBinding(key.WithKeys("r")),
	Break: key.NewBinding(key.WithKeys("b")),
}
