class BreakReminder < Formula
  desc "Smart work/break cycle enforcer for macOS with guided breaks and AI integration"
  homepage "https://github.com/devlikebear/break-reminder"
  url "https://github.com/devlikebear/break-reminder/releases/download/v0.16.0/break-reminder-v0.16.0-darwin-arm64.tar.gz"
  version "0.16.0"
  sha256 "86501e818a15b7ce6ad99550849235baa075a22f697a52bcc70ae470f426624b"
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
