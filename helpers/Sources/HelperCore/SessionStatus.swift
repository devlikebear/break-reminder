import Foundation

/// Reports whether the Go timer is currently ticking for this state.
///
/// With automatic session detection the answer comes from the recorded work
/// session. Without it the Go side gates on the fixed working hours, so only an
/// explicit `break-reminder stop` turns the timer off here.
public func timerIsRunning(state: AppState, config: AppConfig) -> Bool {
    if config.autoSessionDetect {
        return state.isSessionActive
    }
    if state.sessionState == "ended" && state.sessionManual {
        return false
    }
    return true
}

/// Short menu-bar title for a timer that is not running.
public func sessionOffTitle(state: AppState) -> String {
    (state.sessionState == "ended" && state.sessionManual) ? "Off" : "Idle"
}

/// One-line explanation of why the timer is not running.
public func sessionOffStatusLine(state: AppState) -> String {
    if state.sessionState == "ended" && state.sessionManual {
        return "Stopped by hand · start a session to resume"
    }
    if state.sessionState == "ended" {
        return "Work session ended · starts again on your next activity"
    }
    return "Waiting · starts on your first activity"
}
