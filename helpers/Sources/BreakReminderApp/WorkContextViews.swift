import EventKit
import HelperCore
import SwiftUI

/// "Now": current meeting or next event, and how much focus time is left.
struct NowCard: View {
    @ObservedObject var model: AppModel
    @ObservedObject var calendar: CalendarCollector
    let context: WorkContextStatus
    let sessionActive: Bool

    var body: some View {
        let lines = nowCardLines(context: context, calendarAuthorized: calendar.authorized, now: model.nowUnix)
        PopoverCard {
            Text(L10n.text("Now")).font(.subheadline)
            Text(lines.headline).font(.callout).lineLimit(2)
            if let focus = lines.focus {
                HStack {
                    Text(focus).font(.caption).foregroundStyle(.secondary)
                    Spacer()
                    if lines.canStartFocus {
                        Button(L10n.text("Start")) {
                            model.perform("pomodoro.set", ["enabled": true])
                            if !sessionActive { model.perform("session.start") }
                        }
                        .disabled(model.busy)
                    }
                }
            }
            switch calendar.authorization {
            case .notDetermined:
                Button(L10n.text("Connect Calendar")) { calendar.requestAccess() }
            case .denied, .restricted, .writeOnly:
                HStack {
                    Text(L10n.text("Calendar access is off.")).font(.caption).foregroundStyle(.secondary)
                    Button(L10n.text("Open System Settings")) { calendar.openPrivacySettings() }
                }
            default:
                EmptyView()
            }
        }
    }
}

/// Today's most used apps, folded by default.
struct AppUsageSection: View {
    let context: WorkContextStatus
    @State private var expanded = false

    var body: some View {
        DisclosureGroup(isExpanded: $expanded) {
            if !context.appUsageEnabled {
                Text(L10n.text("App usage tracking is off.")).font(.caption).foregroundStyle(.secondary)
            } else if context.appUsageToday.isEmpty {
                Text(L10n.text("No app usage recorded yet today.")).font(.caption).foregroundStyle(.secondary)
            } else {
                ForEach(context.appUsageToday, id: \.bundleID) { item in
                    HStack {
                        Text(item.name).lineLimit(1)
                        Spacer()
                        Text(formatUsageDuration(item.seconds)).font(.caption.monospacedDigit()).foregroundStyle(.secondary)
                    }
                }
            }
        } label: {
            Text(L10n.text("Today's apps")).font(.subheadline)
        }
    }
}

/// Calendar selection, meeting-aware breaks, app usage switch and the
/// diagnostics behind them.
struct WorkContextSettings: View {
    @ObservedObject var model: AppModel
    @ObservedObject var calendar: CalendarCollector
    let context: WorkContextStatus
    @State private var expanded = false
    @State private var confirmDelete = false

    var body: some View {
        DisclosureGroup(isExpanded: $expanded) {
            VStack(alignment: .leading, spacing: 8) {
                Toggle(L10n.text("Pause during meetings"), isOn: binding(context.calendarAwareBreaks) { model.perform("context.settings.set", ["calendar_aware_breaks": $0]) })
                Picker(L10n.text("Counts as a meeting"), selection: binding(context.meetingRule) { model.perform("context.settings.set", ["meeting_rule": $0]) }) {
                    Text(L10n.text("Events with attendees")).tag("attendees")
                    Text(L10n.text("Any busy event")).tag("busy")
                }
                .disabled(!context.calendarAwareBreaks)
                if calendar.authorized && !calendar.calendars.isEmpty {
                    Text(L10n.text("Calendars")).font(.caption).foregroundStyle(.secondary)
                    ForEach(calendar.calendars) { cal in
                        Toggle(cal.title, isOn: binding(!context.calendarExcludedIDs.contains(cal.id)) { include in
                            var excluded = Set(context.calendarExcludedIDs)
                            if include { excluded.remove(cal.id) } else { excluded.insert(cal.id) }
                            model.perform("context.settings.set", ["calendar_excluded_ids": excluded.sorted()])
                        })
                        .lineLimit(1)
                    }
                }
                Divider()
                Toggle(L10n.text("Record app usage"), isOn: binding(context.appUsageEnabled) { model.perform("context.settings.set", ["app_usage_enabled": $0]) })
                Text(L10n.text("Only app names and time in front are stored on this Mac for 30 days. Window titles and contents are never read."))
                    .font(.caption2).foregroundStyle(.secondary).fixedSize(horizontal: false, vertical: true)
                Button(L10n.text("Delete app usage data…")) { confirmDelete = true }
                Divider()
                Text(diagnostics).font(.caption2).foregroundStyle(.secondary).textSelection(.enabled)
            }
            .disabled(model.busy)
        } label: {
            Text(L10n.text("Work context settings")).font(.subheadline)
        }
        .confirmationDialog(L10n.text("Delete all recorded app usage?"), isPresented: $confirmDelete) {
            Button(L10n.text("Delete"), role: .destructive) { model.perform("context.app_usage.clear") }
        }
    }

    private var diagnostics: String {
        let access: String
        switch calendar.authorization {
        case .fullAccess: access = L10n.text("allowed")
        case .notDetermined: access = L10n.text("not asked yet")
        default: access = L10n.text("denied")
        }
        var lines = [L10n.text("Calendar access: {0}", access)]
        if context.calendar.updatedAt > 0 {
            let time = Date(timeIntervalSince1970: TimeInterval(context.calendar.updatedAt)).formatted(date: .omitted, time: .shortened)
            lines.append(L10n.text("Last calendar update: {0}", time) + (context.calendar.stale ? " · " + L10n.text("out of date") : ""))
        }
        if let error = calendar.lastError { lines.append(error) }
        lines.append(L10n.text("App usage: {0}", context.appUsageEnabled ? L10n.text("on") : L10n.text("off")))
        return lines.joined(separator: "\n")
    }

    private func binding<T>(_ value: T, _ set: @escaping (T) -> Void) -> Binding<T> {
        Binding(get: { value }, set: { set($0) })
    }
}

/// Shared card chrome for popover sections.
struct PopoverCard<Content: View>: View {
    @ViewBuilder var content: Content
    var body: some View {
        VStack(alignment: .leading, spacing: 8) { content }
            .padding(10)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(.quaternary.opacity(0.6), in: RoundedRectangle(cornerRadius: 10))
    }
}
