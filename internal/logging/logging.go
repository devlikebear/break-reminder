package logging

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DefaultLogPath returns ~/.break-reminder.log
func DefaultLogPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".break-reminder.log")
}

// Log appends a timestamped line to the log file.
func Log(path, msg string) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), msg)
}

// Rotate trims the log file to maxLines if it exceeds that limit.
func Rotate(path string, maxLines int) {
	f, err := os.Open(path)
	if err != nil {
		return
	}

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	f.Close()

	if len(lines) <= maxLines {
		return
	}

	lines = lines[len(lines)-maxLines:]
	out, err := os.Create(path)
	if err != nil {
		return
	}
	defer out.Close()

	w := bufio.NewWriter(out)
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
	w.Flush()
}

// Tail returns the last n lines from the log file.
func Tail(path string, n int) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if len(lines) <= n {
		return lines
	}
	return lines[len(lines)-n:]
}
