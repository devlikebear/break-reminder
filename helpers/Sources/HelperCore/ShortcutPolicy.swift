/// App-level letter shortcuts must never consume text input.
public func shouldHandlePlainShortcut(isEditing: Bool, hasModifiers: Bool) -> Bool {
    !isEditing && !hasModifiers
}
