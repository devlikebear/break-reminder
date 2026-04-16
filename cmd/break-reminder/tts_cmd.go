package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/tts"
)

// stdinForSetAPIKey is overridden in tests.
var stdinForSetAPIKey io.Reader = os.Stdin

// stdinIsTerminal reports whether os.Stdin is attached to a terminal. Overridden in tests.
var stdinIsTerminal = func() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

var newTTSSpeaker = tts.NewSpeaker
var ttsVoiceAvailable = tts.VoiceAvailable
var ttsSpeakAndWait = tts.SpeakAndWait

func runTTSTest(message string) error {
	apiKey := tts.ResolveAPIKey(cfg)
	if !ttsVoiceAvailable(cfg.TTSEngine, cfg.TTSModel, cfg.TTSPythonCmd, apiKey, cfg.Voice) {
		return fmt.Errorf("voice %q is not available for engine %q", cfg.Voice, cfg.TTSEngine)
	}
	return ttsSpeakAndWait(cfg.TTSEngine, cfg.TTSModel, cfg.TTSPythonCmd, apiKey, cfg.Voice, message)
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
		Short:     "Install an optional TTS backend into a managed environment (gemini is config-only, no install needed)",
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
			case "gemini":
				return fmt.Errorf("gemini engine requires no install; set tts_engine: gemini and GEMINI_API_KEY (or tts_api_key in config)")
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
		Short: "Speak a test phrase with the current TTS configuration (say, kittentts, supertonic, gemini)",
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

	setAPIKeyCmd := &cobra.Command{
		Use:   "set-api-key [key]",
		Short: "Save the Gemini API key to config (permissions 0600)",
		Long: "Save the Gemini API key to the config file with restricted permissions.\n" +
			"If no key is provided as an argument, the command reads one line from stdin\n" +
			"(piped input). When stdin is a terminal, the command refuses and asks you\n" +
			"to pass the key as an argument instead. The stored key never appears in\n" +
			"command output.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetAPIKey(args)
		},
	}

	cmd.AddCommand(installCmd, testCmd, uninstallCmd, setAPIKeyCmd)
	return cmd
}

func runSetAPIKey(args []string) error {
	key, err := readAPIKey(args)
	if err != nil {
		return err
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("api key must not be empty")
	}
	if strings.ContainsAny(key, "\n\r") {
		return fmt.Errorf("api key must not contain newline characters")
	}

	if err := config.EnsureConfigFile(); err != nil {
		return fmt.Errorf("ensure config file: %w", err)
	}

	updated := cfg
	updated.TTSAPIKey = key
	if err := config.Save(updated); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	cfg = updated

	path := config.ConfigPath()
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("restrict permissions on %s: %w", path, err)
	}

	fmt.Printf("Gemini API key saved (%d chars) to %s with permissions 0600\n", len(key), path)
	return nil
}

func readAPIKey(args []string) (string, error) {
	if len(args) == 1 {
		return args[0], nil
	}
	if stdinIsTerminal() {
		return "", fmt.Errorf("pass key as argument: break-reminder tts set-api-key $GEMINI_API_KEY (or pipe it via stdin)")
	}
	scanner := bufio.NewScanner(stdinForSetAPIKey)
	scanner.Buffer(make([]byte, 0, 4096), 1<<20)
	if scanner.Scan() {
		return scanner.Text(), nil
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("read stdin: %w", err)
	}
	return "", fmt.Errorf("no api key provided on stdin")
}
