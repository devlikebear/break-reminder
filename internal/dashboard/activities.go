package dashboard

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Eye Exercise (20-20-20 rule) ──

type EyeActivity struct {
	startTime time.Time
	phase     int // 0,1,2 = three 20-second rounds
	elapsed   time.Duration
	totalDur  time.Duration
}

func NewEyeActivity() *EyeActivity {
	return &EyeActivity{
		startTime: time.Now(),
		totalDur:  2 * time.Minute,
	}
}

func (m *EyeActivity) Init() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m *EyeActivity) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tickMsg:
		m.elapsed = time.Since(m.startTime)
		m.phase = int(m.elapsed.Seconds()) / 20
		if m.phase > 2 {
			m.phase = 2
		}
		if m.elapsed >= m.totalDur {
			return m, nil // done
		}
		return m, tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
	}
	return m, nil
}

func (m *EyeActivity) View() string {
	style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	var b strings.Builder

	b.WriteString(style.Render("👁  눈 운동 - 20-20-20 규칙") + "\n\n")

	phases := []string{
		"20피트(6m) 먼 곳을 20초간 바라보세요 [1/3]",
		"20피트(6m) 먼 곳을 20초간 바라보세요 [2/3]",
		"20피트(6m) 먼 곳을 20초간 바라보세요 [3/3]",
	}

	if m.phase < 3 {
		b.WriteString("  " + phases[m.phase] + "\n\n")
	}

	phaseElapsed := int(m.elapsed.Seconds()) % 20
	remaining := 20 - phaseElapsed
	pct := (phaseElapsed * 100) / 20
	bar := renderBar(pct, 30, lipgloss.NewStyle().Foreground(lipgloss.Color("6")))
	b.WriteString(fmt.Sprintf("  %s  %ds remaining\n\n", bar, remaining))

	totalRemaining := int(m.totalDur.Seconds() - m.elapsed.Seconds())
	if totalRemaining < 0 {
		totalRemaining = 0
	}
	b.WriteString(fmt.Sprintf("  Total: %ds remaining\n", totalRemaining))
	b.WriteString("\n  Press Esc to exit")

	return b.String()
}

// ── Stretch Activity ──

type StretchActivity struct {
	startTime time.Time
	elapsed   time.Duration
	totalDur  time.Duration
}

func NewStretchActivity() *StretchActivity {
	return &StretchActivity{
		startTime: time.Now(),
		totalDur:  5 * time.Minute,
	}
}

func (m *StretchActivity) Init() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m *StretchActivity) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tickMsg:
		m.elapsed = time.Since(m.startTime)
		if m.elapsed >= m.totalDur {
			return m, nil
		}
		return m, tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
	}
	return m, nil
}

func (m *StretchActivity) View() string {
	style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5"))
	var b strings.Builder

	b.WriteString(style.Render("🤸 스트레칭 가이드") + "\n\n")

	steps := []struct {
		name string
		dur  int // seconds
	}{
		{"목 스트레칭 - 좌우로 천천히 기울이기", 60},
		{"어깨 돌리기 - 앞뒤로 크게 원 그리기", 60},
		{"손목 스트레칭 - 손목을 앞뒤로 꺾기", 60},
		{"기립 & 허리 펴기", 60},
		{"자유 스트레칭", 60},
	}

	elapsed := int(m.elapsed.Seconds())
	cumulative := 0
	currentStep := 0
	for i, step := range steps {
		if elapsed >= cumulative+step.dur {
			cumulative += step.dur
			currentStep = i + 1
		} else {
			currentStep = i
			break
		}
	}

	for i, step := range steps {
		prefix := "  "
		if i == currentStep {
			prefix = "▶ "
			stepElapsed := elapsed - cumulative
			remaining := step.dur - stepElapsed
			if remaining < 0 {
				remaining = 0
			}
			b.WriteString(fmt.Sprintf("%s%s (%ds)\n", prefix, step.name, remaining))
		} else if i < currentStep {
			b.WriteString(fmt.Sprintf("  ✓ %s\n", step.name))
		} else {
			b.WriteString(fmt.Sprintf("%s%s\n", prefix, step.name))
		}
	}

	totalRemaining := int(m.totalDur.Seconds() - m.elapsed.Seconds())
	if totalRemaining < 0 {
		totalRemaining = 0
	}
	b.WriteString(fmt.Sprintf("\n  Total: %ds remaining\n", totalRemaining))
	b.WriteString("  Press Esc to exit")

	return b.String()
}

// ── Breathe Activity (Box Breathing) ──

type BreatheActivity struct {
	startTime time.Time
	elapsed   time.Duration
	totalDur  time.Duration
}

func NewBreatheActivity() *BreatheActivity {
	return &BreatheActivity{
		startTime: time.Now(),
		totalDur:  4 * time.Minute,
	}
}

func (m *BreatheActivity) Init() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m *BreatheActivity) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tickMsg:
		m.elapsed = time.Since(m.startTime)
		if m.elapsed >= m.totalDur {
			return m, nil
		}
		return m, tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
	}
	return m, nil
}

func (m *BreatheActivity) View() string {
	style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4"))
	var b strings.Builder

	b.WriteString(style.Render("🌬  호흡 운동 - 박스 호흡법") + "\n\n")

	// Each cycle: 4s inhale + 4s hold + 4s exhale + 4s hold = 16s
	cycleLen := 16
	elapsed := int(m.elapsed.Seconds())
	cycle := (elapsed / cycleLen) + 1
	phaseTime := elapsed % cycleLen

	phases := []struct {
		name string
		dur  int
	}{
		{"들이쉬기 (Inhale)", 4},
		{"멈추기 (Hold)", 4},
		{"내쉬기 (Exhale)", 4},
		{"멈추기 (Hold)", 4},
	}

	cumulative := 0
	currentPhase := 0
	for i, p := range phases {
		if phaseTime >= cumulative+p.dur {
			cumulative += p.dur
		} else {
			currentPhase = i
			break
		}
	}

	phaseElapsed := phaseTime - cumulative
	phaseRemaining := phases[currentPhase].dur - phaseElapsed

	b.WriteString(fmt.Sprintf("  Cycle %d/15\n\n", cycle))

	for i, p := range phases {
		if i == currentPhase {
			pct := (phaseElapsed * 100) / p.dur
			bar := renderBar(pct, 20, style)
			b.WriteString(fmt.Sprintf("  ▶ %s %s %ds\n", p.name, bar, phaseRemaining))
		} else if i < currentPhase {
			b.WriteString(fmt.Sprintf("    ✓ %s\n", p.name))
		} else {
			b.WriteString(fmt.Sprintf("    %s\n", p.name))
		}
	}

	totalRemaining := int(m.totalDur.Seconds() - m.elapsed.Seconds())
	if totalRemaining < 0 {
		totalRemaining = 0
	}
	b.WriteString(fmt.Sprintf("\n  Total: %ds remaining\n", totalRemaining))
	b.WriteString("  Press Esc to exit")

	return b.String()
}

// ── Walk Activity ──

type WalkActivity struct {
	startTime time.Time
	elapsed   time.Duration
	totalDur  time.Duration
}

func NewWalkActivity() *WalkActivity {
	return &WalkActivity{
		startTime: time.Now(),
		totalDur:  5 * time.Minute,
	}
}

func (m *WalkActivity) Init() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m *WalkActivity) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tickMsg:
		m.elapsed = time.Since(m.startTime)
		if m.elapsed >= m.totalDur {
			return m, nil
		}
		return m, tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
	}
	return m, nil
}

func (m *WalkActivity) View() string {
	style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))
	var b strings.Builder

	b.WriteString(style.Render("🚶 산책 타이머") + "\n\n")
	b.WriteString("  자리에서 일어나 가볍게 걸으세요!\n\n")

	totalRemaining := int(m.totalDur.Seconds() - m.elapsed.Seconds())
	if totalRemaining < 0 {
		totalRemaining = 0
	}

	min := totalRemaining / 60
	sec := totalRemaining % 60

	pct := int(m.elapsed.Seconds()) * 100 / int(m.totalDur.Seconds())
	bar := renderBar(pct, 30, style)

	b.WriteString(fmt.Sprintf("  %s\n\n", bar))
	b.WriteString(fmt.Sprintf("  남은 시간: %d:%02d\n", min, sec))
	b.WriteString("\n  Press Esc to exit")

	return b.String()
}
