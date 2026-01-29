#!/bin/bash
# 🐹 Break Reminder - Smart work/break cycle enforcer for macOS
# https://github.com/devlikebear/break-reminder
#
# Work 50 minutes → Rest 10 minutes → Repeat!

set -eu

# Configuration
WORK_DURATION=$((50 * 60))    # 50 minutes in seconds
BREAK_DURATION=$((10 * 60))   # 10 minutes in seconds
IDLE_THRESHOLD=120            # 2 minutes - idle longer than this = not working
CHECK_INTERVAL=60             # Check every 60 seconds (daemon mode)
NATURAL_BREAK_THRESHOLD=300   # 5 minutes - auto-reset if idle this long

# Smart Scheduling
WORK_DAYS="1 2 3 4 5"         # 1=Mon, 7=Sun. Default: Mon-Fri
WORK_START_HOUR=9             # 09:00
WORK_END_HOUR=18              # 18:00

# Daily Stats & Logs
MAX_LOG_LINES=1000            # Log rotation threshold

# File locations
STATE_FILE="$HOME/.break-reminder-state"
LOG_FILE="$HOME/.break-reminder.log"

# Voice settings (run `say -v '?'` to see available voices)
VOICE="Yuna"  # Korean voice. Change to "Samantha" for English, etc.

#=============================================================================
# Functions
#=============================================================================

# Check if within working hours
check_working_hours() {
    local current_hour
    current_hour=$(date +%H)
    local current_day
    current_day=$(date +%u) # 1=Mon, 7=Sun

    # Check Day
    if ! echo "$WORK_DAYS" | grep -q "$current_day"; then
        return 1 # Not a working day
    fi

    # Check Time
    if [[ 10#$current_hour -lt 10#$WORK_START_HOUR ]] || [[ 10#$current_hour -ge 10#$WORK_END_HOUR ]]; then
        return 1 # Outside working hours
    fi

    return 0 # Within working hours
}

# Rotate logs if too large
rotate_logs() {
    if [[ -f "$LOG_FILE" ]]; then
        local line_count
        line_count=$(wc -l < "$LOG_FILE")
        if [[ $line_count -gt $MAX_LOG_LINES ]]; then
            local temp_file="${LOG_FILE}.tmp"
            tail -n "$MAX_LOG_LINES" "$LOG_FILE" > "$temp_file"
            mv "$temp_file" "$LOG_FILE"
        fi
    fi
}

# Check installation status
check_install_status() {
    local plist_path="$HOME/Library/LaunchAgents/com.devlikebear.break-reminder.plist"
    local status="Not Installed"
    
    if [[ -f "$plist_path" ]]; then
        if launchctl list | grep -q "com.devlikebear.break-reminder"; then
            status="Installed & Running"
        else
            status="Installed (Not Loaded)"
        fi
    fi
    echo "$status"
}

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" >> "$LOG_FILE"
}

# Get idle time in seconds (time since last keyboard/mouse input)
get_idle_time() {
    local idle
    idle=$(/usr/sbin/ioreg -c IOHIDSystem | awk '/HIDIdleTime/ {print int($NF/1000000000); exit}' 2>/dev/null || echo "0")
    echo "${idle:-0}"
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

# Read state from file (safe key-value parsing)
read_state() {
    WORK_SECONDS=0
    MODE="work"
    LAST_CHECK=$(date +%s)
    BREAK_START=0

    # Daily Stats
    TODAY_WORK_SECONDS=0
    TODAY_BREAK_SECONDS=0
    LAST_UPDATE_DATE=$(date +%Y-%m-%d)

    if [[ -f "$STATE_FILE" ]]; then
        while IFS='=' read -r key value; do
            # Skip empty lines
            [[ -z "$key" ]] && continue

            case "$key" in
                WORK_SECONDS)
                    [[ "$value" =~ ^[0-9]+$ ]] && WORK_SECONDS=$value
                    ;;
                MODE)
                    [[ "$value" =~ ^(work|break)$ ]] && MODE=$value
                    ;;
                LAST_CHECK)
                    [[ "$value" =~ ^[0-9]+$ ]] && LAST_CHECK=$value
                    ;;
                BREAK_START)
                    [[ "$value" =~ ^[0-9]+$ ]] && BREAK_START=$value
                    ;;
                TODAY_WORK_SECONDS)
                    [[ "$value" =~ ^[0-9]+$ ]] && TODAY_WORK_SECONDS=$value
                    ;;
                TODAY_BREAK_SECONDS)
                    [[ "$value" =~ ^[0-9]+$ ]] && TODAY_BREAK_SECONDS=$value
                    ;;
                LAST_UPDATE_DATE)
                    [[ "$value" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ ]] && LAST_UPDATE_DATE=$value
                    ;;
            esac
        done < "$STATE_FILE"
    fi
}

# Save state to file
save_state() {
    cat > "$STATE_FILE" << EOF
WORK_SECONDS=$WORK_SECONDS
MODE=$MODE
LAST_CHECK=$LAST_CHECK
BREAK_START=$BREAK_START
TODAY_WORK_SECONDS=$TODAY_WORK_SECONDS
TODAY_BREAK_SECONDS=$TODAY_BREAK_SECONDS
LAST_UPDATE_DATE=$LAST_UPDATE_DATE
EOF
}

# Main check function - called every minute
check_and_remind() {
    read_state
    
    # Check working hours first
    if ! check_working_hours; then
        # If we were in a session, maybe we should end it or just pause?
        # For now, we save state and exit silently.
        # But we should probably reset LAST_CHECK to avoid huge Elapsed times next time.
        LAST_CHECK=$(date +%s)
        save_state
        return
    fi
    
    local now
    now=$(date +%s)
    
    # Daily Stats Reset
    local today
    today=$(date +%Y-%m-%d)
    if [[ "$today" != "$LAST_UPDATE_DATE" ]]; then
        log "New day detected! Resetting daily stats."
        TODAY_WORK_SECONDS=0
        TODAY_BREAK_SECONDS=0
        LAST_UPDATE_DATE="$today"
    fi
    
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
        handle_break_mode "$idle_time" "$elapsed" "$now"
    fi
    
    rotate_logs
    save_state
}

handle_work_mode() {
    local idle_time=$1
    local elapsed=$2
    local now=$3
    
    if [[ $idle_time -lt $IDLE_THRESHOLD ]]; then
        # User is active - accumulate work time
        WORK_SECONDS=$((WORK_SECONDS + elapsed))
        TODAY_WORK_SECONDS=$((TODAY_WORK_SECONDS + elapsed))

        
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
    local elapsed=$2
    local now=$3
    
    TODAY_BREAK_SECONDS=$((TODAY_BREAK_SECONDS + elapsed))

    
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
    local daily_work_min=$((TODAY_WORK_SECONDS / 60))
    local daily_break_min=$((TODAY_BREAK_SECONDS / 60))
    local install_status
    install_status=$(check_install_status)
    
    echo "🐹 Break Reminder Status"
    echo "========================"
    echo "System: $install_status"
    
    if check_working_hours; then
        echo "State:  Active (Within working hours)"
    else
        echo "State:  Inactive (Outside working hours)"
    fi
    
    echo "------------------------"
    echo "Mode: $MODE"
    echo "Session Work: ${work_minutes}min / $((WORK_DURATION / 60))min"
    echo "Daily Stats: Work ${daily_work_min}min / Break ${daily_break_min}min"
    echo "Current idle: ${idle_time}sec"
    
    if [[ "$MODE" == "break" ]]; then
        local now
        now=$(date +%s)
        local break_elapsed=$((now - BREAK_START))
        local break_minutes=$((break_elapsed / 60))
        echo "Break elapsed: ${break_minutes}min / $((BREAK_DURATION / 60))min"
    fi
}

# Draw progress bar
draw_bar() {
    local percent=$1
    local length=$2
    local color=$3
    local fill_len=$(( (percent * length) / 100 ))
    local empty_len=$((length - fill_len))
    
    printf "["
    printf "${color}"
    for ((i=0; i<fill_len; i++)); do printf "#"; done
    printf "\033[0m"
    for ((i=0; i<empty_len; i++)); do printf "-"; done
    printf "] %d%%" "$percent"
}

# TUI Dashboard
show_dashboard() {
    # Colors
    local GREEN='\033[0;32m'
    local BLUE='\033[0;34m'
    local RED='\033[0;31m'
    local YELLOW='\033[1;33m'
    local CYAN='\033[0;36m'
    local NC='\033[0m'
    local EL
    EL=$(tput el)  # Clear to end of line

    # Print a line and clear remaining characters
    printl() {
        echo -e "$1${EL}"
    }

    trap "tput cnorm; clear; exit 0" SIGINT
    tput civis  # Hide cursor
    clear       # Initial clear

    while true; do
        read_state
        local idle_time
        idle_time=$(get_idle_time)
        local now
        now=$(date +%s)

        tput cup 0 0  # Move cursor to top-left (no flicker)

        printl "${CYAN}🐹 Break Reminder Dashboard${NC} (Press 'q' to quit)"
        printl "=================================================="

        local install_status
        install_status=$(check_install_status)
        printl "System: ${install_status}"

        # Status Section
        if ! check_working_hours; then
            printl "Status: ${YELLOW}SLEEPING (Outside Working Hours)${NC}"
        else
            if [[ "$MODE" == "work" ]]; then
                printl "Status: ${GREEN}WORKING${NC}"
            else
                printl "Status: ${BLUE}ON BREAK${NC}"
            fi
        fi

        printl "Idle: ${idle_time}s / Limit: ${IDLE_THRESHOLD}s"
        printl ""

        # Session Progress
        local session_pct=0
        if [[ "$MODE" == "work" ]]; then
            session_pct=$(( (WORK_SECONDS * 100) / WORK_DURATION ))
            [ $session_pct -gt 100 ] && session_pct=100
            printf "Session Work: "
            draw_bar "$session_pct" 30 "$GREEN"
            printl " ($((WORK_SECONDS/60)) / $((WORK_DURATION/60)) min)"
        else
            local break_elapsed=$((now - BREAK_START))
            session_pct=$(( (break_elapsed * 100) / BREAK_DURATION ))
            [ $session_pct -gt 100 ] && session_pct=100
            printf "Break Timer:  "
            draw_bar "$session_pct" 30 "$BLUE"
            printl " ($((break_elapsed/60)) / $((BREAK_DURATION/60)) min)"
        fi

        printl ""

        # Daily Stats
        local daily_work_min=$((TODAY_WORK_SECONDS / 60))
        local daily_break_min=$((TODAY_BREAK_SECONDS / 60))
        local total_min=$((daily_work_min + daily_break_min))

        printl "Daily Statistics:"
        printl "  Work: ${daily_work_min} min"
        printl "  Rest: ${daily_break_min} min"
        if [[ $total_min -gt 0 ]]; then
            local work_ratio=$(( (daily_work_min * 100) / total_min ))
            printf "  Ratio: "
            draw_bar "$work_ratio" 20 "$YELLOW"
            printl ""
        else
            printl ""
        fi

        printl ""
        printl "Recent Logs:"
        printl "--------------------------------------------------"
        if [[ -f "$LOG_FILE" ]]; then
            tail -n 5 "$LOG_FILE" | while read -r line; do
                printl "  $line"
            done
        else
            printl "  (No logs yet)"
        fi
        printl "--------------------------------------------------"

        # Clear any remaining lines from previous render
        tput ed

        # Read input non-blocking (|| true to prevent exit on timeout)
        if read -t 1 -n 1 key 2>/dev/null; then
            if [[ "$key" == "q" ]]; then
                tput cnorm
                clear
                break
            fi
        fi
    done
    tput cnorm
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

# Install Launchd Agent
install_launchd() {
    local plist_path="$HOME/Library/LaunchAgents/com.devlikebear.break-reminder.plist"
    local script_path
    script_path=$(realpath "$0")
    
    cat > "$plist_path" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.devlikebear.break-reminder</string>
    <key>ProgramArguments</key>
    <array>
        <string>${script_path}</string>
        <string>check</string>
    </array>
    <key>StartInterval</key>
    <integer>60</integer>
    <key>RunAtLoad</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/break-reminder.out</string>
    <key>StandardErrorPath</key>
    <string>/tmp/break-reminder.err</string>
</dict>
</plist>
EOF
    
    echo "Generated plist at $plist_path"
    
    # Unload if exists, then load
    launchctl unload "$plist_path" 2>/dev/null || true
    launchctl load "$plist_path"
    
    echo "✅ Successfully installed and loaded break-reminder agent!"
    echo "It will now run every minute in the background."
}

# Uninstall Launchd Agent
uninstall_launchd() {
    local plist_path="$HOME/Library/LaunchAgents/com.devlikebear.break-reminder.plist"

    if [[ -f "$plist_path" ]]; then
        launchctl unload "$plist_path" 2>/dev/null || true
        rm "$plist_path"
        echo "✅ Successfully uninstalled break-reminder agent."
    else
        echo "⚠️  Agent not found (not installed?)"
    fi
}

# Show help
show_help() {
    cat << EOF
🐹 Break Reminder - Smart work/break cycle enforcer for macOS

Usage: $(basename "$0") <command>

Commands:
  dashboard Run TUI Dashboard (Real-time view)
  status    Show current status/stats
  install   Install as macOS LaunchAgent (Runs every minute)
  uninstall Uninstall macOS LaunchAgent
  check     Run a single check (used by launchd)
  reset     Reset the timer
  daemon    Run as foreground daemon (for testing)
  help      Show this help message

Configuration:
  Edit the variables at the top of this script to customize:
  - WORK_DURATION    Work period (default: 50 minutes)
  - BREAK_DURATION   Break period (default: 10 minutes)
  - WORK_DAYS        Working days (1=Mon, 5=Fri)
  - WORK_START/END   Working hours (24h format)
  - MAX_LOG_LINES    Log rotation limit

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
    dashboard)
        show_dashboard
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
    install)
        install_launchd
        ;;
    uninstall)
        uninstall_launchd
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
