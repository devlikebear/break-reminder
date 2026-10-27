package engine

import (
	"context"
	"time"

	"github.com/devlikebear/break-reminder/internal/agent"
	"github.com/devlikebear/break-reminder/internal/ai"
	"github.com/devlikebear/break-reminder/internal/assistant"
	"github.com/devlikebear/break-reminder/internal/i18n"
	"github.com/devlikebear/break-reminder/internal/logging"
	"github.com/devlikebear/break-reminder/internal/reminder"
	"github.com/devlikebear/break-reminder/internal/workcontext"
)

// AIService is what the engine needs from internal/agent (agent.Service).
type AIService interface {
	Write(ctx context.Context, system, prompt string) (string, error)
	Ask(ctx context.Context, question string, history []agent.Turn, allowActions bool, onDelta func(string)) (agent.Answer, error)
	Status() agent.ProviderStatus
	Test(ctx context.Context) error
	SetKey(kind, key string) error
}

const assistantTimeout = 2 * time.Minute

// afterTransitions starts the briefing on a session start and the retro on a
// session end. Both run once a day and never block the tick.
func (e *Engine) afterTransitions(events []reminder.Event) {
	for _, ev := range events {
		switch ev.Type {
		case reminder.TypeSessionStart:
			e.goBackground(e.briefing)
		case reminder.TypeSessionEnd:
			e.goBackground(e.retro)
		}
	}
}

func (e *Engine) goBackground(fn func()) {
	e.background.Add(1)
	go func() { defer e.background.Done(); fn() }()
}

// WaitBackground waits for briefing/retro work started so far.
func (e *Engine) WaitBackground() { e.background.Wait() }

// briefing writes today's briefing if there is none yet, with the AI when it
// is enabled and consented, a template otherwise.
func (e *Engine) briefing() {
	e.assistantMu.Lock()
	defer e.assistantMu.Unlock()
	now := e.opts.Now()
	day, err := e.assistant.Load(now.Format("2006-01-02"))
	if err != nil || day.Briefing != "" {
		return
	}
	cfg, f, inMeeting := e.assistantFacts(now)
	text, source := e.write(assistant.BriefingPrompt(f), func() string { return assistant.TemplateBriefing(f) })
	day.Briefing, day.BriefingSource, day.BriefingAt, day.Dismissed = text, source, now.Unix(), false
	if err := e.assistant.Save(day); err != nil {
		logging.Log(e.opts.LogPath, err.Error())
		return
	}
	ev := reminder.Event{Type: reminder.TypeBriefing, Title: i18n.Text("Today's briefing"), Body: text, Sound: "Glass"}
	if cfg.BriefingVoice && !inMeeting {
		ev.Text, ev.Voice = text, cfg.Voice
	}
	e.opts.Emit(ev)
	e.publish()
}

// retro writes today's retrospective once; the AI reply also carries the
// insights report, so a day costs at most one briefing and one retro call.
func (e *Engine) retro() {
	e.assistantMu.Lock()
	defer e.assistantMu.Unlock()
	now := e.opts.Now()
	day, err := e.assistant.Load(now.Format("2006-01-02"))
	if err != nil || day.Retro != "" {
		return
	}
	_, f, _ := e.assistantFacts(now)
	text, source := e.template(assistant.TemplateRetro(f))
	if e.opts.AI != nil {
		ctx, cancel := context.WithTimeout(context.Background(), assistantTimeout)
		raw, err := e.opts.AI.Write(ctx, assistant.SystemPrompt(), assistant.RetroPrompt(f))
		cancel()
		if err == nil {
			if retro, report, perr := assistant.ParseRetro(raw); perr == nil && retro != "" {
				text, source = retro, assistant.SourceAI
				if report != nil {
					report.GeneratedAt = now.Format(time.RFC3339)
					if err := e.opts.SaveInsights(report); err != nil {
						logging.Log(e.opts.LogPath, err.Error())
					}
				}
			}
		} else {
			logging.Log(e.opts.LogPath, i18n.Text("AI retrospective unavailable, using the template")+": "+err.Error())
		}
	}
	day.Retro, day.RetroSource, day.RetroAt, day.Dismissed = text, source, now.Unix(), false
	if err := e.assistant.Save(day); err != nil {
		logging.Log(e.opts.LogPath, err.Error())
		return
	}
	e.opts.Emit(reminder.Event{Type: reminder.TypeRetro, Title: i18n.Text("Today's retrospective"), Body: text})
	e.publish()
}

func (e *Engine) template(text string) (string, string) { return text, assistant.SourceTemplate }

func (e *Engine) write(prompt string, fallback func() string) (string, string) {
	if e.opts.AI == nil {
		return fallback(), assistant.SourceTemplate
	}
	ctx, cancel := context.WithTimeout(context.Background(), assistantTimeout)
	defer cancel()
	text, err := e.opts.AI.Write(ctx, assistant.SystemPrompt(), prompt)
	if err != nil || text == "" {
		if err != nil {
			logging.Log(e.opts.LogPath, i18n.Text("AI briefing unavailable, using the template")+": "+err.Error())
		}
		return fallback(), assistant.SourceTemplate
	}
	return text, assistant.SourceAI
}

func (e *Engine) assistantFacts(now time.Time) (cfgOut configView, f assistant.Facts, inMeeting bool) {
	e.mu.Lock()
	cfg := e.config()
	st, _ := e.statusLocked()
	events := e.calendar.Visible(cfg.CalendarExcludedIDs)
	stale := e.calendar.Stale(now)
	e.mu.Unlock()

	f.Date = now.Format("2006-01-02")
	f.NextEventInMin = -1
	if hist, err := ai.LoadHistory(); err == nil {
		if len(hist) > 0 {
			last := hist[len(hist)-1]
			f.YesterdayWorkMin, f.YesterdayBreakMin = last.WorkMin, last.BreakMin
		}
		if len(hist) > 7 {
			hist = hist[len(hist)-7:]
		}
		f.Week = hist
	}
	if !stale {
		y, m, d := now.Date()
		start := time.Date(y, m, d, 0, 0, 0, 0, now.Location()).Unix()
		end := start + 24*3600
		for _, ev := range events {
			if !ev.AllDay && ev.Start >= start && ev.Start < end {
				f.TodayEvents++
			}
		}
		if next := workcontext.NextEvent(events, now); next != nil {
			f.NextEventInMin = int(next.Start-now.Unix()) / 60
			if cfg.ShareEventTitles {
				f.NextEventTitle = next.Title
			}
		}
		f.FocusMinutes, f.FocusPomodoros = workcontext.FocusWindow(events, now, cfg)
		inMeeting = workcontext.CurrentMeeting(events, now, cfg.MeetingRule) != nil
	}
	f.TodayWorkMin = st.State.TodayWorkSeconds / 60
	f.TodayBreakMin = st.State.TodayBreakSeconds / 60
	f.TodayPomodoros = st.State.TodayPomodoros
	return configView{BriefingVoice: cfg.BriefingVoice, Voice: cfg.Voice}, f, inMeeting
}

type configView struct {
	BriefingVoice bool
	Voice         string
}

// retroFor returns the retrospective written for date, for the history file.
func (e *Engine) retroFor(date string) string {
	day, err := e.assistant.Load(date)
	if err != nil {
		return ""
	}
	return day.Retro
}

// DismissAssistant hides today's briefing/retro card.
func (e *Engine) DismissAssistant() (Status, error) {
	e.assistantMu.Lock()
	day, err := e.assistant.Load(e.opts.Now().Format("2006-01-02"))
	if err == nil {
		day.Dismissed = true
		err = e.assistant.Save(day)
	}
	e.assistantMu.Unlock()
	if err != nil {
		return Status{}, err
	}
	st, err := e.Status()
	if err == nil {
		e.opts.Status(st)
	}
	return st, err
}
