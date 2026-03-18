package doctor

import "testing"

func TestReportFailCount(t *testing.T) {
	r := Report{}
	r.add("ok", "check1", "fine")
	r.add("fail", "check2", "broken")
	r.add("warn", "check3", "meh")
	r.add("fail", "check4", "also broken")

	if got := r.FailCount(); got != 2 {
		t.Errorf("FailCount() = %d, want 2", got)
	}
}

func TestReportFailCountZero(t *testing.T) {
	r := Report{}
	r.add("ok", "check1", "fine")
	r.add("warn", "check2", "meh")

	if got := r.FailCount(); got != 0 {
		t.Errorf("FailCount() = %d, want 0", got)
	}
}

func TestReportEmpty(t *testing.T) {
	r := Report{}
	if got := r.FailCount(); got != 0 {
		t.Errorf("FailCount() on empty = %d, want 0", got)
	}
}
