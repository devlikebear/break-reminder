#!/bin/bash
# 🐹 Break Reminder Uninstaller
set -euo pipefail

echo "🐹 Break Reminder Uninstaller"
echo "============================="
echo ""

INSTALL_DIR="$HOME/.local/bin"
PLIST_NAME="com.user.break-reminder.plist"
LAUNCH_AGENTS_DIR="$HOME/Library/LaunchAgents"

# Unload launch agent
echo "⏹️  Stopping service..."
launchctl unload "$LAUNCH_AGENTS_DIR/$PLIST_NAME" 2>/dev/null || true

# Remove files
echo "🗑️  Removing files..."
rm -f "$LAUNCH_AGENTS_DIR/$PLIST_NAME"
rm -f "$INSTALL_DIR/break-reminder.sh"
rm -f "$INSTALL_DIR/break-reminder"

# Ask about state files
echo ""
read -p "Remove state and log files? (~/.break-reminder-*) [y/N] " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Yy]$ ]]; then
    rm -f "$HOME/.break-reminder-state"
    rm -f "$HOME/.break-reminder.log"
    echo "   State files removed."
fi

echo ""
echo "✅ Uninstallation complete!"
echo ""
echo "👋 Goodbye! Remember to take breaks anyway! 🐹"
