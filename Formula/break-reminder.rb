class BreakReminder < Formula
  desc "Smart work/break cycle enforcer for macOS with guided breaks and AI integration"
  homepage "https://github.com/devlikebear/break-reminder"
  url "https://github.com/devlikebear/break-reminder/releases/download/v0.7.1/break-reminder-v0.7.1-darwin-arm64.tar.gz"
  sha256 "778aa5d4a5b024c32bfe3a1b00c3506cf40a63f3ea8ffaea701669d46a78d8a2"
  version "0.7.1"
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
