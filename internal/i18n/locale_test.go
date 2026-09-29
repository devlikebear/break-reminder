package i18n

import "testing"

func TestResolveLanguage(t *testing.T) {
	for _, tc := range []struct{ tag, want string }{{"ko-KR", "ko"}, {"ko_KR.UTF-8", "ko"}, {"en-US", "en"}, {"ja-JP", "en"}, {"C", "en"}, {"", "en"}} {
		if got := Resolve(tc.tag); got != tc.want {
			t.Errorf("%q: %s", tc.tag, got)
		}
	}
}
func TestTranslatedArgumentsRemainLiteral(t *testing.T) {
	if got := For("ko", "Timer complete: {0}", "Tea {1}"); got != "타이머 완료: Tea {1}" {
		t.Fatal(got)
	}
	if got := For("en", "Timer complete: {0}", "차"); got != "Timer complete: 차" {
		t.Fatal(got)
	}
}
func TestOverride(t *testing.T) {
	t.Setenv("BREAK_REMINDER_LANGUAGE", "ko")
	if Text("Timer") != "타이머" {
		t.Fatal("override ignored")
	}
	t.Setenv("BREAK_REMINDER_LANGUAGE", "en")
	if Text("Timer") != "Timer" {
		t.Fatal("override cached")
	}
}
