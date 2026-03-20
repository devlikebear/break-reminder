package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/tts"
)

var newTTSSpeaker = tts.NewSpeaker
var ttsVoiceAvailable = tts.VoiceAvailable
var ttsSpeakAndWait = tts.SpeakAndWait

func runTTSTest(message string) error {
	if !ttsVoiceAvailable(cfg.TTSEngine, cfg.TTSModel, cfg.TTSPythonCmd, cfg.Voice) {
		return fmt.Errorf("voice %q is not available for engine %q", cfg.Voice, cfg.TTSEngine)
	}
	return ttsSpeakAndWait(cfg.TTSEngine, cfg.TTSModel, cfg.TTSPythonCmd, cfg.Voice, message)
}

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
		ValidArgs: []string{"kittentts", "supertonic"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.EnsureConfigFile(); err != nil {
				return fmt.Errorf("ensure config file: %w", err)
			}

			opts := tts.InstallOptions{
				BootstrapPython:         bootstrapPython,
				BootstrapPythonExplicit: cmd.Flags().Changed("bootstrap-python"),
				Model:                   model,
				Voice:                   voice,
				Force:                   force,
			}

			var (
				updatedCfg config.Config
				result     tts.InstallResult
				err        error
				label      string
			)

			switch args[0] {
			case "kittentts":
				label = "KittenTTS"
				updatedCfg, result, err = tts.InstallKittenTTS(cfg, opts)
			case "supertonic":
				label = "Supertonic"
				updatedCfg, result, err = tts.InstallSupertonic(cfg, opts)
			default:
				return fmt.Errorf("unsupported TTS engine %q", args[0])
			}
			if err != nil {
				return err
			}

			if err := config.Save(updatedCfg); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			cfg = updatedCfg

			status := label + " already present; configuration updated"
			if result.Installed {
				status = label + " installed and configuration updated"
			}

			fmt.Println(status)
			fmt.Printf("Bootstrap Python: %s\n", result.BootstrapPython)
			fmt.Printf("Python: %s\n", result.PythonCmd)
			fmt.Printf("Model: %s\n", result.Model)
			fmt.Printf("Voice: %s\n", result.Voice)
			fmt.Printf("Config: %s\n", config.ConfigPath())
			return nil
		},
	}

	installCmd.Flags().StringVar(&bootstrapPython, "bootstrap-python", "", "Python used to create the managed virtual environment (default: auto-select a compatible interpreter)")
	installCmd.Flags().StringVar(&voice, "voice", "", "Voice to activate after install")
	installCmd.Flags().StringVar(&model, "model", "", "Model identifier to activate after install")
	installCmd.Flags().BoolVar(&force, "force", false, "Re-run pip installation even if the package is already installed")

	testCmd := &cobra.Command{
		Use:   "test [message]",
		Short: "Speak a test phrase with the current TTS configuration",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := runTTSTest(args[0]); err != nil {
				return err
			}
			fmt.Printf("TTS playback completed via %s voice %s\n", cfg.TTSEngine, cfg.Voice)
			return nil
		},
	}

	uninstallCmd := &cobra.Command{
		Use:       "uninstall [engine]",
		Short:     "Remove a managed TTS backend installation",
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"kittentts", "supertonic"},
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				updatedCfg config.Config
				result     tts.UninstallResult
				err        error
				label      string
			)

			switch args[0] {
			case "kittentts":
				label = "KittenTTS"
				updatedCfg, result, err = tts.UninstallKittenTTS(cfg)
			case "supertonic":
				label = "Supertonic"
				updatedCfg, result, err = tts.UninstallSupertonic(cfg)
			default:
				return fmt.Errorf("unsupported TTS engine %q", args[0])
			}
			if err != nil {
				return err
			}

			if result.ConfigUpdated {
				if err := config.Save(updatedCfg); err != nil {
					return fmt.Errorf("save config: %w", err)
				}
				cfg = updatedCfg
			}

			switch {
			case result.Removed && result.ConfigUpdated:
				fmt.Printf("%s removed and configuration restored to say defaults\n", label)
			case result.Removed:
				fmt.Printf("%s managed environment removed\n", label)
			case result.ConfigUpdated:
				fmt.Printf("%s configuration restored to say defaults\n", label)
			default:
				fmt.Printf("%s was not installed; no changes made\n", label)
			}

			fmt.Printf("Managed path: %s\n", result.VenvDir)
			if result.ConfigUpdated {
				fmt.Printf("Config: %s\n", config.ConfigPath())
			}
			return nil
		},
	}

	cmd.AddCommand(installCmd, testCmd, uninstallCmd)
	return cmd
}
