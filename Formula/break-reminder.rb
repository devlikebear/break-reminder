class BreakReminder < Formula
  desc "Smart work/break cycle enforcer for macOS with guided breaks and AI integration"
  homepage "https://github.com/devlikebear/break-reminder"
  url "https://github.com/devlikebear/break-reminder/releases/download/v0.1.1/break-reminder-v0.1.1-darwin-arm64.tar.gz"
  sha256 "9b997791b24e1a41d5221142063a62a52dba735c5a6c7894be25ae82265e4819"
  version "0.1.1"
  license "MIT"

  depends_on :macos

  def install
    bin.install "break-reminder"
    bin.install "break-screen"
    bin.install "break-dashboard"
  end

  def post_install
    ohai "Run 'break-reminder service install' to set up the launchd agent"
    ohai "Run 'break-reminder doctor' to verify your setup"
    ohai "Run 'break-reminder dashboard' for the TUI dashboard"
    ohai "Optional: run 'break-reminder tts install kittentts' to enable KittenTTS"
  end

  def caveats
    <<~EOS
      To start break-reminder as a background service:
        break-reminder service install

      To check system status:
        break-reminder doctor

      To enable KittenTTS:
        break-reminder tts install kittentts

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
