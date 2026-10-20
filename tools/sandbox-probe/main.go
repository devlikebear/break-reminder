// Command sandbox-probe checks what the engine sidecar can do inside the App
// Sandbox. It is bundled only into the spike build (scripts/build-app.sh
// --sandbox --spike) and prints one JSON report; see
// docs/plans/renewal-spike-sandbox.md.
package main

/*
#include <libproc.h>
#include <sys/resource.h>
static int rusage_for(int pid, unsigned long long *rss, unsigned long long *cpu) {
	struct rusage_info_v2 ri;
	int rc = proc_pid_rusage(pid, RUSAGE_INFO_V2, (rusage_info_t *)&ri);
	if (rc == 0) { *rss = ri.ri_resident_size; *cpu = ri.ri_user_time + ri.ri_system_time; }
	return rc;
}
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

type result struct {
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

func check(fn func() (string, error)) result {
	detail, err := fn()
	if err != nil {
		return result{false, err.Error()}
	}
	return result{true, detail}
}

func main() {
	// Several probes try writes that only the sandbox should refuse; never
	// run them against the real home directory.
	if os.Getenv("APP_SANDBOX_CONTAINER_ID") == "" || os.Getenv("BREAK_REMINDER_HOME") == "" {
		fmt.Fprintln(os.Stderr, "sandbox-probe: run only inside the sandboxed spike app")
		os.Exit(2)
	}
	report := map[string]result{}
	home := os.Getenv("BREAK_REMINDER_HOME")

	report["env"] = result{true, fmt.Sprintf("HOME=%s BREAK_REMINDER_HOME=%s container=%s", os.Getenv("HOME"), home, os.Getenv("APP_SANDBOX_CONTAINER_ID"))}

	report["write_container"] = check(func() (string, error) {
		path := filepath.Join(home, "probe-write.txt")
		if err := os.WriteFile(path, []byte("ok"), 0o600); err != nil {
			return "", err
		}
		return path, nil
	})
	report["write_legacy_home"] = check(func() (string, error) {
		// The real home (outside the container) must be off limits.
		real := "/Users/" + os.Getenv("USER") + "/.break-reminder-probe"
		if err := os.WriteFile(real, []byte("x"), 0o600); err != nil {
			return "", err
		}
		os.Remove(real)
		return real, nil
	})

	report["http_localhost"] = check(func() (string, error) {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return "", err
		}
		defer ln.Close()
		srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { io.WriteString(w, "pong") })}
		go srv.Serve(ln)
		defer srv.Close()
		client := http.Client{Timeout: 3 * time.Second}
		resp, err := client.Get("http://" + ln.Addr().String() + "/")
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return ln.Addr().String() + " -> " + string(body), nil
	})

	report["sysctl_proc_list"] = check(func() (string, error) {
		procs, err := unix.SysctlKinfoProcSlice("kern.proc.all")
		if err != nil {
			return "", err
		}
		return strconv.Itoa(len(procs)) + " processes", nil
	})

	report["proc_pid_rusage_other"] = check(func() (string, error) {
		pid := os.Getppid() // the app: another process of the same user
		if out, err := exec.Command("/usr/bin/pgrep", "-x", "Finder").Output(); err == nil {
			if p, err := strconv.Atoi(strings.TrimSpace(strings.Split(string(out), "\n")[0])); err == nil {
				pid = p
			}
		}
		var rss, cpu C.ulonglong
		if rc := C.rusage_for(C.int(pid), &rss, &cpu); rc != 0 {
			return "", fmt.Errorf("proc_pid_rusage(%d) rc=%d", pid, rc)
		}
		return fmt.Sprintf("pid %d rss=%dMB cpu_ns=%d", pid, uint64(rss)>>20, uint64(cpu)), nil
	})

	for name, cmd := range map[string][]string{
		"exec_ps":        {"/bin/ps", "-axo", "pid,pcpu,rss"},
		"exec_ioreg":     {"/usr/sbin/ioreg", "-c", "IOHIDSystem"},
		"exec_say":       {"/usr/bin/say", "-o", filepath.Join(home, "probe.aiff"), "probe"},
		"exec_osascript": {"/usr/bin/osascript", "-e", "return 1"},
		"exec_defaults":  {"/usr/bin/defaults", "read", "-g", "AppleLanguages"},
	} {
		cmd := cmd
		report[name] = check(func() (string, error) {
			out, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
			if err != nil {
				return "", fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
			}
			return fmt.Sprintf("%d bytes", len(out)), nil
		})
	}

	report["write_launchagents"] = check(func() (string, error) {
		path := "/Users/" + os.Getenv("USER") + "/Library/LaunchAgents/com.devlikebear.probe.plist"
		if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
			return "", err
		}
		os.Remove(path)
		return path, nil
	})

	json.NewEncoder(os.Stdout).Encode(report)
}
