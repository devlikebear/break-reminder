package tts

import "testing"

func TestNormalizeEngine(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{name: "default", input: "", expect: engineSay},
		{name: "say", input: "say", expect: engineSay},
		{name: "kitten alias", input: "kitten", expect: engineKittenTTS},
		{name: "kitten canonical", input: "kittentts", expect: engineKittenTTS},
		{name: "trim and lower", input: "  KiTTeNtts  ", expect: engineKittenTTS},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeEngine(tt.input); got != tt.expect {
				t.Fatalf("normalizeEngine(%q) = %q, want %q", tt.input, got, tt.expect)
			}
		})
	}
}

func TestBuildSpeakCommandSay(t *testing.T) {
	cmd, err := buildSpeakCommand(engineSay, "", "", "Yuna", "rest now")
	if err != nil {
		t.Fatalf("buildSpeakCommand() error = %v", err)
	}

	want := []string{"say", "-v", "Yuna", "rest now"}
	if len(cmd.Args) != len(want) {
		t.Fatalf("args length = %d, want %d (%v)", len(cmd.Args), len(want), cmd.Args)
	}
	for i := range want {
		if cmd.Args[i] != want[i] {
			t.Fatalf("args[%d] = %q, want %q (all args: %v)", i, cmd.Args[i], want[i], cmd.Args)
		}
	}
}

func TestBuildSpeakCommandKittenTTS(t *testing.T) {
	cmd, err := buildSpeakCommand("kitten", "", "", "Jasper", "stretch time")
	if err != nil {
		t.Fatalf("buildSpeakCommand() error = %v", err)
	}

	if cmd.Args[0] != defaultPythonCommand {
		t.Fatalf("command = %q, want %q", cmd.Args[0], defaultPythonCommand)
	}
	if got := cmd.Args[len(cmd.Args)-3]; got != defaultKittenModel {
		t.Fatalf("model arg = %q, want %q", got, defaultKittenModel)
	}
	if got := cmd.Args[len(cmd.Args)-2]; got != "Jasper" {
		t.Fatalf("voice arg = %q, want Jasper", got)
	}
	if got := cmd.Args[len(cmd.Args)-1]; got != "stretch time" {
		t.Fatalf("message arg = %q, want stretch time", got)
	}
}

func TestKittenVoiceAvailable(t *testing.T) {
	if !kittenVoiceAvailable("Jasper") {
		t.Fatal("Jasper should be supported by KittenTTS")
	}
	if kittenVoiceAvailable("Yuna") {
		t.Fatal("Yuna should not be treated as a KittenTTS voice")
	}
}
