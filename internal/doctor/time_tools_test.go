package doctor

import (
	"github.com/devlikebear/break-reminder/internal/timetools"
	"strings"
	"testing"
)

func TestTimeToolsDiagnosticExplainsRecovery(t *testing.T) {
	for _, status := range []string{"Not Installed", "Installed (not running)"} {
		check := timeToolsDiagnostic(status, timetools.Runtime{}, 100000)
		if check.Status != "warn" || !strings.Contains(check.Detail, "service") {
			t.Fatal(check)
		}
	}
	check := timeToolsDiagnostic("Running", timetools.Runtime{Error: "broken store"}, 100000)
	if check.Status != "fail" || !strings.Contains(check.Detail, "broken store") {
		t.Fatal(check)
	}
}
