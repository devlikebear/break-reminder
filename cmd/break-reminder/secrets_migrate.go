package main

import (
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/devlikebear/break-reminder/internal/config"
	"github.com/devlikebear/break-reminder/internal/i18n"
	"github.com/devlikebear/break-reminder/internal/logging"
	"github.com/devlikebear/break-reminder/internal/secrets"
)

// migrateLegacySecrets moves a plaintext tts_api_key from config.yaml into the
// keychain once. On failure the key stays where it was and keeps working.
func migrateLegacySecrets() {
	key := strings.TrimSpace(cfg.TTSAPIKey)
	if key == "" {
		return
	}
	if err := secrets.Default.Set(secrets.TTSAPIKey, key); err != nil {
		log.Warn().Err(err).Msg(i18n.Text("Could not move tts_api_key to the keychain; it stays in config.yaml"))
		return
	}
	updated := cfg
	updated.TTSAPIKey = ""
	if err := config.Save(updated); err != nil {
		log.Warn().Err(err).Msg(i18n.Text("Could not remove tts_api_key from config.yaml"))
		return
	}
	cfg = updated
	logging.Log(logging.DefaultLogPath(), i18n.Text("Moved tts_api_key from config.yaml to the keychain"))
}
