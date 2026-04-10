class BreakReminder < Formula
  desc "Smart work/break cycle enforcer for macOS with guided breaks and AI integration"
  homepage "https://github.com/devlikebear/break-reminder"
  url "https://github.com/devlikebear/break-reminder/releases/download/v0.4.0/break-reminder-v0.4.0-darwin-arm64.tar.gz"
  sha256 "aa08df5297c0fae7ccc2bc56ee5d52d9e7ba206bafec3daacf5df95e035bc277"
  version "0.4.0"
  license "MIT"

  depends_on :macos

  def install
    bin.install "break-reminder"
    bin.install "break-screen"
    bin.install "break-dashboard"
    bin.install "break-menubar"
  end

  def post_install
    ohai "Run 'break-reminder service install' to set up the launchd agent"
    ohai "Run 'break-reminder doctor' to verify your setup"
    ohai "Run 'break-reminder dashboard' for the TUI dashboard"
    ohai "Optional: run 'break-reminder tts install kittentts' or 'break-reminder tts install supertonic' to enable alternate TTS engines"
  end

  def caveats
    <<~EOS
      To start break-reminder as a background service:
        break-reminder service install

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
