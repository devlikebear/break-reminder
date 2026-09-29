#!/usr/bin/env python3
"""Reject literal UI prose outside L10n (symbols, formats and brand names are exempt)."""
from pathlib import Path
import re
import sys
import unittest

# Direct SwiftUI/AppKit display sinks. Configuration keys and SF Symbols never
# pass through these arguments; Text(userValue) also deliberately stays literal.
SINK = re.compile(r'(?:\b(?:Text|Button|Label|Toggle|Picker|GroupBox|Section|TextField|ProgressView)\s*\(\s*|\.(?:help|alert|navigationTitle|accessibilityLabel|accessibilityHint)\s*\(\s*|NSMenuItem\s*\(\s*title\s*:\s*)("(?:[^"\\]|\\.)*")')
EXEMPT = {'"Break Reminder"'}

def violations(source):
    for match in SINK.finditer(source):
        literal = match.group(1)
        # Percentages/numeric expressions are locale-neutral; prose is not.
        stripped = re.sub(r'\\\([^)]*\)', '', literal)
        if literal not in EXEMPT and re.search(r'[A-Za-z가-힣]', stripped):
            yield source.count('\n', 0, match.start()) + 1, literal

class GuardTests(unittest.TestCase):
    def test_rejects_legacy_and_new_untranslated_labels(self):
        self.assertEqual(len(list(violations('Text("Open Dashboard")\nButton("일시정지") {}\nNSMenuItem(title: "Reset Timer", action: nil)'))), 3)
    def test_accepts_translation_user_content_and_symbols(self):
        self.assertEqual(list(violations('Text(L10n.text("Timer"))\nText(timer.label)\nText("42%")\nText("Break Reminder")')), [])

if __name__ == '__main__':
    if '--test' in sys.argv:
        unittest.main(argv=[sys.argv[0]])
    else:
        root = Path(__file__).resolve().parents[1]
        errors = []
        for path in (root / 'helpers/Sources').rglob('*.swift'):
            errors += [f'{path.relative_to(root)}:{line}: untranslated UI: {literal}' for line, literal in violations(path.read_text())]
        if errors:
            sys.exit('\n'.join(errors))
