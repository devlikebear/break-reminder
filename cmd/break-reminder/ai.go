package main

import (
	"github.com/devlikebear/break-reminder/internal/i18n"

	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/devlikebear/break-reminder/internal/ai"
	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/state"
)

func newAICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ai",
		Short: i18n.Text("AI-powered features"),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Load config explicitly (child PersistentPreRunE overrides parent's in Cobra)
			if err := setAppConfig(); err != nil {
				return err
			}
			if !cfg.AIEnabled {
				return fmt.Errorf("%s", i18n.Text("AI features are disabled. Enable with: break-reminder config edit (set ai_enabled: true)"))
			}
			return nil
		},
	}

	cmd.AddCommand(
		newAISuggestCmd(),
		newAISummaryCmd(),
		newAIConfigureCmd(),
	)

	return cmd
}

func newAISuggestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "suggest",
		Short: i18n.Text("AI-powered optimal break timing analysis"),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := ai.NewClient(cfg.AICLI)
			if !client.Available() {
				return fmt.Errorf(i18n.Text("%s CLI not found in PATH"), cfg.AICLI)
			}

			history, err := ai.LoadHistory()
			if err != nil {
				return fmt.Errorf(i18n.Text("load history: %w"), err)
			}

			if len(history) == 0 {
				fmt.Println(i18n.Text("Not enough history data yet. Use break-reminder for a few days first."))
				return nil
			}

			historyJSON, _ := json.Marshal(history)
			prompt := fmt.Sprintf(`Analyze this break-reminder usage history and suggest optimal work/break timing.
Current settings: work=%dmin, break=%dmin.
History (recent days): %s

Provide:
1. Recommended work/break durations based on patterns
2. Best times for breaks
3. Brief reasoning

Keep response concise (under 200 words).`, cfg.WorkDurationMin, cfg.BreakDurationMin, string(historyJSON))

			fmt.Println(i18n.Text("Analyzing your patterns..."))
			resp, err := client.Query(context.Background(), prompt+"\nWrite explanatory prose in "+i18n.ResponseLanguage()+". Preserve code and configuration keys.")
			if err != nil {
				return err
			}
			fmt.Println(resp)
			return nil
		},
	}
}

func newAISummaryCmd() *cobra.Command {
	var weekly bool

	cmd := &cobra.Command{
		Use:   "summary",
		Short: i18n.Text("AI-powered productivity report"),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := ai.NewClient(cfg.AICLI)
			if !client.Available() {
				return fmt.Errorf(i18n.Text("%s CLI not found in PATH"), cfg.AICLI)
			}

			history, err := ai.LoadHistory()
			if err != nil {
				return fmt.Errorf(i18n.Text("load history: %w"), err)
			}

			// Include today's live data from state file
			today := time.Now().Format("2006-01-02")
			s, _ := state.Load(state.DefaultStatePath())
			now := time.Now().Unix()
			todayWorkSec := s.TodayWorkSeconds
			if s.Mode == "work" && !s.Paused && s.LastCheck > 0 {
				todayWorkSec += int(now - s.LastCheck)
			}
			todaySummary := ai.DailySummary{
				Date:     today,
				WorkMin:  todayWorkSec / 60,
				BreakMin: s.TodayBreakSeconds / 60,
			}

			// Upsert today's entry in history for the prompt
			foundToday := false
			for i, h := range history {
				if h.Date == today {
					history[i] = todaySummary
					foundToday = true
					break
				}
			}
			if !foundToday {
				history = append(history, todaySummary)
			}

			if len(history) == 0 {
				fmt.Println(i18n.Text("Not enough history data yet."))
				return nil
			}

			// Take last 7 days for weekly, last 1 for daily
			n := 1
			label := "daily"
			if weekly {
				n = 7
				label = "weekly"
			}
			if len(history) < n {
				n = len(history)
			}
			recent := history[len(history)-n:]

			historyJSON, _ := json.Marshal(recent)
			prompt := fmt.Sprintf(`Generate a %s productivity report from this break-reminder data: %s

Include:
1. Total work/break time
2. Work/break ratio
3. Patterns observed
4. One actionable suggestion

Keep it concise and encouraging. Respond in %s.`, label, string(historyJSON), i18n.ResponseLanguage())

			fmt.Printf(i18n.Text("Generating %s report...\n"), i18n.Text(label))
			resp, err := client.Query(context.Background(), prompt+"\nWrite explanatory prose in "+i18n.ResponseLanguage()+". Preserve code and configuration keys.")
			if err != nil {
				return err
			}
			fmt.Println(resp)
			return nil
		},
	}

	cmd.Flags().BoolVar(&weekly, "weekly", false, i18n.Text("Generate weekly report instead of daily"))
	return cmd
}

func newAIConfigureCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "configure [description]",
		Short: i18n.Text("Configure settings using natural language"),
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := ai.NewClient(cfg.AICLI)
			if !client.Available() {
				return fmt.Errorf(i18n.Text("%s CLI not found in PATH"), cfg.AICLI)
			}

			description := strings.Join(args, " ")
			currentYAML, _ := yaml.Marshal(&cfg)

			prompt := fmt.Sprintf(`Given this break-reminder config:
%s

The user wants: "%s"

Output ONLY the changed YAML fields (not the full config). For example:
work_duration_min: 25
break_duration_min: 5

No explanation, just the YAML fields to change.`, string(currentYAML), description)

			resp, err := client.Query(context.Background(), prompt)
			if err != nil {
				return err
			}

			fmt.Println(i18n.Text("Proposed changes:"))
			fmt.Println(resp)
			fmt.Print(i18n.Text("\nApply these changes? (y/N): "))

			var answer string
			fmt.Scanln(&answer)
			if answer != "y" && answer != "Y" {
				fmt.Println(i18n.Text("Cancelled."))
				return nil
			}

			updatedCfg, err := config.ApplyYAMLChanges(cfg, []byte(resp))
			if err != nil {
				return fmt.Errorf(i18n.Text("invalid configuration changes: %w"), err)
			}
			if err := config.Save(updatedCfg); err != nil {
				return err
			}

			cfg = updatedCfg
			fmt.Println(i18n.Text("Configuration updated!"))
			return nil
		},
	}
}
