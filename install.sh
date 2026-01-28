#!/bin/bash
# 🐹 Break Reminder Installer
set -euo pipefail

echo "🐹 Break Reminder Installer"
echo "==========================="
echo ""

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_DIR="$HOME/.local/bin"
PLIST_NAME="com.user.break-reminder.plist"
LAUNCH_AGENTS_DIR="$HOME/Library/LaunchAgents"

# Create install directory
mkdir -p "$INSTALL_DIR"
mkdir -p "$LAUNCH_AGENTS_DIR"

# Copy script
echo "📦 Installing break-reminder.sh to $INSTALL_DIR..."
cp "$SCRIPT_DIR/break-reminder.sh" "$INSTALL_DIR/"
chmod +x "$INSTALL_DIR/break-reminder.sh"

# Create plist with correct path
echo "⚙️  Setting up launch agent..."
sed "s|INSTALL_PATH|$INSTALL_DIR|g" "$SCRIPT_DIR/$PLIST_NAME" > "$LAUNCH_AGENTS_DIR/$PLIST_NAME"

# Unload if already loaded
launchctl unload "$LAUNCH_AGENTS_DIR/$PLIST_NAME" 2>/dev/null || true

# Load the agent
launchctl load "$LAUNCH_AGENTS_DIR/$PLIST_NAME"

# Create symlink for easy access
if [[ -d "$HOME/.local/bin" ]] && [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
    echo ""
    echo "💡 Tip: Add ~/.local/bin to your PATH for easy access:"
    echo '   echo '\''export PATH="$HOME/.local/bin:$PATH"'\'' >> ~/.zshrc'
fi

# Create alias command
ln -sf "$INSTALL_DIR/break-reminder.sh" "$INSTALL_DIR/break-reminder" 2>/dev/null || true

echo ""
echo "✅ Installation complete!"
echo ""
echo "📋 Quick commands:"
echo "   break-reminder status  - Check current status"
echo "   break-reminder reset   - Reset the timer"
echo ""
echo "🚀 Break Reminder is now running!"
echo "   It will remind you to take a break after 50 minutes of work."
echo ""
echo "🐹 Stay healthy!"
