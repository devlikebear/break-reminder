package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/tts"
)

func newTTSCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tts",
		Short: "Manage optional text-to-speech backends",
	}

	var bootstrapPython string
	var voice string
	var model string
	var force bool

	installCmd := &cobra.Command{
		Use:       "install [engine]",
		Short:     "Install an optional TTS backend into a managed environment",
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"kittentts"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[0] != "kittentts" {
				return fmt.Errorf("unsupported TTS engine %q", args[0])
			}

			if err := config.EnsureConfigFile(); err != nil {
				return fmt.Errorf("ensure config file: %w", err)
			}

			updatedCfg, result, err := tts.InstallKittenTTS(cfg, tts.InstallOptions{
				BootstrapPython: bootstrapPython,
				Model:           model,
				Voice:           voice,
				Force:           force,
			})
			if err != nil {
				return err
			}

			if err := config.Save(updatedCfg); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			cfg = updatedCfg

			status := "KittenTTS already present; configuration updated"
			if result.Installed {
				status = "KittenTTS installed and configuration updated"
			}

			fmt.Println(status)
			fmt.Printf("Python: %s\n", result.PythonCmd)
			fmt.Printf("Model: %s\n", result.Model)
			fmt.Printf("Voice: %s\n", result.Voice)
			fmt.Printf("Config: %s\n", config.ConfigPath())
			return nil
		},
	}

	installCmd.Flags().StringVar(&bootstrapPython, "bootstrap-python", "python3", "Python used to create the managed virtual environment")
	installCmd.Flags().StringVar(&voice, "voice", "", "KittenTTS voice to activate after install")
	installCmd.Flags().StringVar(&model, "model", "", "KittenTTS model to activate after install")
	installCmd.Flags().BoolVar(&force, "force", false, "Re-run pip installation even if kittentts is already installed")

	cmd.AddCommand(installCmd)
	return cmd
}
