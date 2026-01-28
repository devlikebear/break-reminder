#!/bin/bash
# 🐹 Break Reminder - Smart work/break cycle enforcer for macOS
# https://github.com/devlikebear/break-reminder
#
# Work 50 minutes → Rest 10 minutes → Repeat!

set -euo pipefail

# Configuration
WORK_DURATION=$((50 * 60))    # 50 minutes in seconds
BREAK_DURATION=$((10 * 60))   # 10 minutes in seconds
IDLE_THRESHOLD=120            # 2 minutes - idle longer than this = not working
CHECK_INTERVAL=60             # Check every 60 seconds (daemon mode)
NATURAL_BREAK_THRESHOLD=300   # 5 minutes - auto-reset if idle this long

# File locations
STATE_FILE="$HOME/.break-reminder-state"
LOG_FILE="$HOME/.break-reminder.log"

# Voice settings (run `say -v '?'` to see available voices)
VOICE="Yuna"  # Korean voice. Change to "Samantha" for English, etc.

#=============================================================================
# Functions
#=============================================================================

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" >> "$LOG_FILE"
}

# Get idle time in seconds (time since last keyboard/mouse input)
get_idle_time() {
    /usr/sbin/ioreg -c IOHIDSystem | awk '/HIDIdleTime/ {print int($NF/1000000000); exit}'
}

# Send macOS notification
send_notification() {
    local title="$1"
    local message="$2"
    local sound="${3:-Glass}"
    
    osascript -e "display notification \"$message\" with title \"$title\" sound name \"$sound\""
}

# Text-to-speech announcement
speak_message() {
    say -v "$VOICE" "$1" &
}

# Lock the screen (optional - for enforced breaks)
lock_screen() {
    osascript -e 'tell application "System Events" to keystroke "q" using {control down, command down}'
}

# Read state from file
read_state() {
    WORK_SECONDS=0
    MODE="work"
    LAST_CHECK=$(date +%s)
    BREAK_START=0
    
    if [[ -f "$STATE_FILE" ]]; then
        # shellcheck source=/dev/null
        source "$STATE_FILE"
    fi
}

# Save state to file
save_state() {
    cat > "$STATE_FILE" << EOF
WORK_SECONDS=$WORK_SECONDS
MODE=$MODE
LAST_CHECK=$LAST_CHECK
BREAK_START=$BREAK_START
EOF
}

# Main check function - called every minute
check_and_remind() {
    read_state
    
    local now
    now=$(date +%s)
    local idle_time
    idle_time=$(get_idle_time)
    local elapsed=$((now - LAST_CHECK))
    
    # Reset if too much time has passed (computer restart, sleep, etc.)
    if [[ $elapsed -gt 3600 ]]; then
        log "Long gap detected ($elapsed sec), resetting..."
        WORK_SECONDS=0
        MODE="work"
    fi
    
    LAST_CHECK=$now
    
    if [[ "$MODE" == "work" ]]; then
        handle_work_mode "$idle_time" "$elapsed" "$now"
    elif [[ "$MODE" == "break" ]]; then
        handle_break_mode "$idle_time" "$now"
    fi
    
    save_state
}

handle_work_mode() {
    local idle_time=$1
    local elapsed=$2
    local now=$3
    
    if [[ $idle_time -lt $IDLE_THRESHOLD ]]; then
        # User is active - accumulate work time
        WORK_SECONDS=$((WORK_SECONDS + elapsed))
        
        local work_minutes=$((WORK_SECONDS / 60))
        local remaining_minutes=$(( (WORK_DURATION - WORK_SECONDS) / 60 ))
        
        log "Working... ${work_minutes}min elapsed (${remaining_minutes}min remaining)"
        
        # 50 minutes reached - time for a break!
        if [[ $WORK_SECONDS -ge $WORK_DURATION ]]; then
            log "🔔 Break time triggered!"
            
            send_notification "🐹 Break Time!" "50 minutes complete! Take a 10-minute break~" "Blow"
            speak_message "Time for a break! You've been working for 50 minutes."
            
            # Switch to break mode
            MODE="break"
            BREAK_START=$now
            WORK_SECONDS=0
            
            # Optional: Force screen lock
            # lock_screen
        fi
        
        # 5-minute warning
        local warning_start=$((WORK_DURATION - 5 * 60))
        local warning_end=$((warning_start + 60))
        if [[ $WORK_SECONDS -ge $warning_start ]] && [[ $WORK_SECONDS -lt $warning_end ]]; then
            send_notification "⏰ 5 minutes left" "Break time coming up~"
        fi
    else
        # User is idle - might be a natural break
        if [[ $idle_time -gt $NATURAL_BREAK_THRESHOLD ]]; then
            log "Natural break detected (idle ${idle_time}s), resetting work time"
            WORK_SECONDS=0
        fi
    fi
}

handle_break_mode() {
    local idle_time=$1
    local now=$2
    
    local break_elapsed=$((now - BREAK_START))
    local break_remaining=$(( (BREAK_DURATION - break_elapsed) / 60 ))
    
    log "Break mode... ${break_remaining}min remaining"
    
    # Warn if user is active during break
    if [[ $idle_time -lt 30 ]] && [[ $break_elapsed -lt $BREAK_DURATION ]]; then
        if [[ $((break_elapsed % 120)) -lt 60 ]]; then  # Warn every 2 minutes max
            send_notification "🚫 Still on break!" "${break_remaining} more minutes to rest!"
        fi
    fi
    
    # Break is over!
    if [[ $break_elapsed -ge $BREAK_DURATION ]]; then
        log "Break finished, back to work mode"
        send_notification "💪 Break Over!" "Back to work! 50-minute timer started~" "Hero"
        speak_message "Break time is over! Let's get back to work!"
        MODE="work"
        WORK_SECONDS=0
    fi
}

# Show current status
show_status() {
    read_state
    local idle_time
    idle_time=$(get_idle_time)
    local work_minutes=$((WORK_SECONDS / 60))
    
    echo "🐹 Break Reminder Status"
    echo "========================"
    echo "Mode: $MODE"
    echo "Work time: ${work_minutes}min / $((WORK_DURATION / 60))min"
    echo "Current idle: ${idle_time}sec"
    
    if [[ "$MODE" == "break" ]]; then
        local now
        now=$(date +%s)
        local break_elapsed=$((now - BREAK_START))
        local break_minutes=$((break_elapsed / 60))
        echo "Break elapsed: ${break_minutes}min / $((BREAK_DURATION / 60))min"
    fi
}

# Reset timer
reset_timer() {
    WORK_SECONDS=0
    MODE="work"
    LAST_CHECK=$(date +%s)
    BREAK_START=0
    save_state
    echo "✅ Timer has been reset."
    log "Timer manually reset"
}

# Show help
show_help() {
    cat << EOF
🐹 Break Reminder - Smart work/break cycle enforcer for macOS

Usage: $(basename "$0") <command>

Commands:
  check     Run a single check (used by launchd)
  status    Show current status
  reset     Reset the timer
  daemon    Run as foreground daemon (for testing)
  help      Show this help message

Configuration:
  Edit the variables at the top of this script to customize:
  - WORK_DURATION    Work period (default: 50 minutes)
  - BREAK_DURATION   Break period (default: 10 minutes)
  - IDLE_THRESHOLD   Idle time to count as "not working"
  - VOICE            TTS voice (run 'say -v ?' for options)

Files:
  ~/.break-reminder-state   Current state
  ~/.break-reminder.log     Activity log

More info: https://github.com/devlikebear/break-reminder
EOF
}

#=============================================================================
# Main
#=============================================================================

case "${1:-check}" in
    check)
        check_and_remind
        ;;
    status)
        show_status
        ;;
    reset)
        reset_timer
        ;;
    daemon)
        log "Daemon started"
        echo "🐹 Break Reminder daemon started (Ctrl+C to stop)"
        while true; do
            check_and_remind
            sleep $CHECK_INTERVAL
        done
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        echo "Unknown command: $1"
        echo "Run '$(basename "$0") help' for usage."
        exit 1
        ;;
esac
