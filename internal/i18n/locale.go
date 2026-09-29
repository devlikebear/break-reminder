// Package i18n localizes application messages. English is the fallback language.
package i18n

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
)

//go:embed catalog.json
var catalogData []byte
var korean = func() map[string]string {
	var values map[string]string
	if err := json.Unmarshal(catalogData, &values); err != nil {
		panic(err)
	}
	return values
}()

func Resolve(tag string) string {
	base := strings.FieldsFunc(strings.ToLower(tag), func(r rune) bool { return r == '-' || r == '_' || r == '.' || r == '@' })
	if len(base) > 0 && base[0] == "ko" {
		return "ko"
	}
	return "en"
}

var systemLanguage = sync.OnceValue(func() string {
	if runtime.GOOS == "darwin" {
		if data, err := exec.Command("/usr/bin/defaults", "read", "-g", "AppleLanguages").Output(); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				tag := strings.Trim(strings.TrimSpace(line), "\",() ")
				if tag != "" {
					return Resolve(tag)
				}
			}
		}
	}
	return "en"
})

func Current() string {
	for _, key := range []string{"BREAK_REMINDER_LANGUAGE", "LC_ALL", "LC_MESSAGES", "LANG"} {
		if value := os.Getenv(key); value != "" {
			return Resolve(value)
		}
	}
	return systemLanguage()
}
func Text(key string, args ...any) string { return For(Current(), key, args...) }
func For(language, key string, args ...any) string {
	template := key
	if Resolve(language) == "ko" {
		if value, ok := korean[key]; ok {
			template = value
		}
	}
	replacements := make([]string, 0, len(args)*2)
	for i, arg := range args {
		replacements = append(replacements, fmt.Sprintf("{%d}", i), fmt.Sprint(arg))
	}
	return strings.NewReplacer(replacements...).Replace(template)
}

// ResponseLanguage is prompt metadata, not localized user content.
func ResponseLanguage() string {
	if Current() == "ko" {
		return "Korean"
	}
	return "English"
}
