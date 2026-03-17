package schedule

import (
	"testing"
	"time"

	"github.com/devlikebear/break-reminder/internal/config"
)

func TestIsWorkingTime(t *testing.T) {
	cfg := config.Default()

	tests := []struct {
		name string
		time time.Time
		want bool
	}{
		{
			name: "weekday during work hours",
			time: time.Date(2025, 1, 15, 10, 0, 0, 0, time.Local), // Wednesday 10:00
			want: true,
		},
		{
			name: "weekday before work hours",
			time: time.Date(2025, 1, 15, 7, 0, 0, 0, time.Local), // Wednesday 07:00
			want: false,
		},
		{
			name: "weekday after work hours",
			time: time.Date(2025, 1, 15, 19, 0, 0, 0, time.Local), // Wednesday 19:00
			want: false,
		},
		{
			name: "saturday",
			time: time.Date(2025, 1, 18, 10, 0, 0, 0, time.Local), // Saturday
			want: false,
		},
		{
			name: "sunday",
			time: time.Date(2025, 1, 19, 10, 0, 0, 0, time.Local), // Sunday
			want: false,
		},
		{
			name: "weekday at work start boundary",
			time: time.Date(2025, 1, 15, 9, 0, 0, 0, time.Local), // Wednesday 09:00
			want: true,
		},
		{
			name: "weekday at work end boundary",
			time: time.Date(2025, 1, 15, 18, 0, 0, 0, time.Local), // Wednesday 18:00
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsWorkingTime(cfg, tt.time)
			if got != tt.want {
				t.Errorf("IsWorkingTime(%v) = %v, want %v", tt.time, got, tt.want)
			}
		})
	}
}
