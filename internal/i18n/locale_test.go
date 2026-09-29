package i18n

import (
	"reflect"
	"regexp"
	"sort"
	"testing"
)

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

func TestCatalogPreservesFormattingArguments(t *testing.T) {
	printf := regexp.MustCompile(`%[-+# 0]*(?:[0-9]+)?(?:\.[0-9]+)?[a-zA-Z%]`)
	indexed := regexp.MustCompile(`\{[0-9]+\}`)
	for english, korean := range korean {
		if korean == "" {
			t.Errorf("empty translation: %q", english)
		}
		want, got := printf.FindAllString(english, -1), printf.FindAllString(korean, -1)
		if !reflect.DeepEqual(want, got) {
			t.Errorf("printf mismatch %q: %v != %v", english, want, got)
		}
		want, got = indexed.FindAllString(english, -1), indexed.FindAllString(korean, -1)
		sort.Strings(want)
		sort.Strings(got) // Indexed placeholders may change order.
		if !reflect.DeepEqual(want, got) {
			t.Errorf("indexed mismatch %q: %v != %v", english, want, got)
		}
	}
}
