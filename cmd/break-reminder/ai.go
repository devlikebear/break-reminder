package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/devlikebear/break-reminder/internal/ai"
	"github.com/devlikebear/break-reminder/internal/config"
)

func newAICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ai",
		Short: "AI-powered features",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if !cfg.AIEnabled {
				return fmt.Errorf("AI features are disabled. Enable with: break-reminder config edit (set ai_enabled: true)")
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
		Short: "AI-powered optimal break timing analysis",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := ai.NewClient(cfg.AICLI)
			if !client.Available() {
				return fmt.Errorf("%s CLI not found in PATH", cfg.AICLI)
			}

			history, err := ai.LoadHistory()
			if err != nil {
				return fmt.Errorf("load history: %w", err)
			}

			if len(history) == 0 {
				fmt.Println("Not enough history data yet. Use break-reminder for a few days first.")
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

			fmt.Println("Analyzing your patterns...")
			resp, err := client.Query(context.Background(), prompt)
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
		Short: "AI-powered productivity report",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := ai.NewClient(cfg.AICLI)
			if !client.Available() {
				return fmt.Errorf("%s CLI not found in PATH", cfg.AICLI)
			}

			history, err := ai.LoadHistory()
			if err != nil {
				return fmt.Errorf("load history: %w", err)
			}

			if len(history) == 0 {
				fmt.Println("Not enough history data yet.")
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

Keep it concise and encouraging.`, label, string(historyJSON))

			fmt.Printf("Generating %s report...\n", label)
			resp, err := client.Query(context.Background(), prompt)
			if err != nil {
				return err
			}
			fmt.Println(resp)
			return nil
		},
	}

	cmd.Flags().BoolVar(&weekly, "weekly", false, "Generate weekly report instead of daily")
	return cmd
}

func newAIConfigureCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "configure [description]",
		Short: "Configure settings using natural language",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := ai.NewClient(cfg.AICLI)
			if !client.Available() {
				return fmt.Errorf("%s CLI not found in PATH", cfg.AICLI)
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

			fmt.Println("Proposed changes:")
			fmt.Println(resp)
			fmt.Print("\nApply these changes? (y/N): ")

			var answer string
			fmt.Scanln(&answer)
			if answer != "y" && answer != "Y" {
				fmt.Println("Cancelled.")
				return nil
			}

			// Merge changes into config file
			if err := config.EnsureConfigFile(); err != nil {
				return err
			}

			// Read existing, apply changes
			existing, err := os.ReadFile(config.ConfigPath())
			if err != nil {
				return err
			}

			var cfgMap map[string]any
			if err := yaml.Unmarshal(existing, &cfgMap); err != nil {
				cfgMap = make(map[string]any)
			}

			var changes map[string]any
			if err := yaml.Unmarshal([]byte(resp), &changes); err != nil {
				return fmt.Errorf("could not parse AI response as YAML: %w", err)
			}

			for k, v := range changes {
				cfgMap[k] = v
			}

			data, err := yaml.Marshal(cfgMap)
			if err != nil {
				return err
			}

			if err := os.WriteFile(config.ConfigPath(), data, 0o644); err != nil {
				return err
			}

			fmt.Println("Configuration updated!")
			return nil
		},
	}
}
