package main

import (
	"github.com/devlikebear/break-reminder/internal/i18n"

	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/devlikebear/break-reminder/internal/ai"
	"github.com/devlikebear/break-reminder/internal/insights"
)

func newInsightsCmd() *cobra.Command {
	var refresh bool

	cmd := &cobra.Command{
		Use:   "insights",
		Short: i18n.Text("Show or refresh AI insights"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if refresh {
				return refreshInsights()
			}
			return showInsights()
		},
	}

	cmd.Flags().BoolVar(&refresh, "refresh", false, i18n.Text("Force regenerate insights via AI CLI"))
	return cmd
}

func refreshInsights() error {
	if !cfg.AIEnabled {
		return fmt.Errorf("%s", i18n.Text("AI is disabled in config (set ai_enabled: true)"))
	}

	client := ai.NewClient(cfg.AICLI)
	if !client.Available() {
		return fmt.Errorf(i18n.Text("AI CLI %q not found in PATH"), cfg.AICLI)
	}

	history, err := ai.LoadHistory()
	if err != nil {
		return fmt.Errorf(i18n.Text("load history: %w"), err)
	}

	recent := trimRecentHistory(history, 7)

	log.Info().Int("entries", len(recent)).Msg(i18n.Text("Generating AI insights"))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	report, err := insights.Generate(ctx, client, recent, time.Now())
	if err != nil {
		return fmt.Errorf(i18n.Text("generate: %w"), err)
	}

	if err := insights.Save(report); err != nil {
		return fmt.Errorf(i18n.Text("save: %w"), err)
	}

	fmt.Println(i18n.Text("Insights refreshed:"))
	fmt.Println(report.DailyReport)
	return nil
}

func showInsights() error {
	report, err := insights.Load()
	if err != nil {
		return err
	}
	if report == nil {
		fmt.Println(i18n.Text("No insights yet. Run with --refresh to generate."))
		return nil
	}
	fmt.Printf(i18n.Text("Generated: %s\n\n"), report.GeneratedAt)
	fmt.Println(report.DailyReport)
	fmt.Println()
	for _, p := range report.Patterns {
		fmt.Printf("[%s] %s\n  %s\n  → %s\n\n", i18n.Text(p.Type), p.Title, p.Description, p.Suggestion)
	}
	return nil
}

func trimRecentHistory(history []ai.DailySummary, days int) []ai.DailySummary {
	if len(history) <= days {
		return history
	}
	return history[len(history)-days:]
}
