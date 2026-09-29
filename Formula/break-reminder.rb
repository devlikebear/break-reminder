class BreakReminder < Formula
  desc "Smart work/break cycle enforcer for macOS with guided breaks and AI integration"
  homepage "https://github.com/devlikebear/break-reminder"
  url "https://github.com/devlikebear/break-reminder/releases/download/v0.16.1/break-reminder-v0.16.1-darwin-arm64.tar.gz"
  version "0.16.1"
  sha256 "794cb021f8b7736fa2a2e03092f533e8dd2973198723545bf7dae996008613a1"
  license "MIT"

  depends_on :macos
  depends_on "terminal-notifier"

  def install
    bin.install "break-reminder"
    bin.install "break-screen"
    bin.install "BreakReminderHelpers_BreakScreenApp.bundle"
    bin.install "break-dashboard"
    bin.install "break-menubar"
  end

  def post_install
    ohai "Run 'break-reminder service install' to set up timer/menu bar/time-tools agents and daily automatic updates"
    ohai "Run 'break-reminder doctor' to verify your setup"
    ohai "Run 'break-reminder dashboard' for the TUI dashboard"
    ohai "Optional: run 'break-reminder tts install kittentts' or " \
         "'break-reminder tts install supertonic' to enable alternate TTS engines"
  end

  def caveats
    <<~EOS
      To start break-reminder as a background service and enable daily updates:
        break-reminder service install

      After upgrading from a version without time tools, run service install once.
      Closing the menu bar does not stop active countdowns; service stop does.

      Homebrew installations check for updates every day at 04:00.
      To check immediately:
        break-reminder update

      To check system status:
        break-reminder doctor

      To enable KittenTTS:
        break-reminder tts install kittentts

      To enable Supertonic:
        break-reminder tts install supertonic

      Configuration file:
        ~/.config/break-reminder/config.yaml

      To uninstall the service:
        break-reminder service uninstall
    EOS
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/break-reminder version")
  end
end
