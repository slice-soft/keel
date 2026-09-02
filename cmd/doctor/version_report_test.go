package doctor

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

// captureStdout runs fn and returns everything it printed. The check helpers
// write straight to stdout, so there is no seam to assert on other than this.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	done := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()
	fn()
	_ = w.Close()
	os.Stdout = orig
	return <-done
}

// The bug this covers: every failed lookup was dropped with a bare `continue`,
// so a machine with no network produced a report as green as one where every
// addon was current. Silence there reads as "all up to date", which is the one
// thing it does not mean.
func TestReportVersionCheckFailures_AllFailedSaysUnreachable(t *testing.T) {
	var warnings bool
	out := captureStdout(t, func() {
		reportVersionCheckFailures([]string{"gorm", "jwt"}, 2, false, &warnings)
	})

	if !warnings {
		t.Error("a check that could not run must not leave the verdict green")
	}
	if !strings.Contains(out, "could not check any addon version") {
		t.Errorf("expected an unreachable message, got:\n%s", out)
	}
	if !strings.Contains(out, "rest of this report is unaffected") {
		t.Errorf("the reader needs to know the other checks still hold, got:\n%s", out)
	}
}

func TestReportVersionCheckFailures_RateLimitAdvisesAToken(t *testing.T) {
	var warnings bool
	out := captureStdout(t, func() {
		reportVersionCheckFailures([]string{"gorm"}, 3, true, &warnings)
	})

	if !warnings {
		t.Error("expected a warning")
	}
	if !strings.Contains(out, "rate limit") {
		t.Errorf("expected the rate limit named, got:\n%s", out)
	}
	// The advice has to be actionable: GITHUB_TOKEN is honoured by the fetcher.
	if !strings.Contains(out, "GITHUB_TOKEN") {
		t.Errorf("expected the remedy, got:\n%s", out)
	}
}

// A partial failure names the addons, because "2 of 5 failed" without saying
// which ones leaves the reader unable to check them by hand.
func TestReportVersionCheckFailures_PartialNamesThem(t *testing.T) {
	var warnings bool
	out := captureStdout(t, func() {
		reportVersionCheckFailures([]string{"redis", "otel"}, 5, false, &warnings)
	})

	if !warnings {
		t.Error("expected a warning")
	}
	for _, want := range []string{"2 of 5", "redis", "otel"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in:\n%s", want, out)
		}
	}
}

func TestReportVersionCheckFailures_NoFailuresIsSilent(t *testing.T) {
	var warnings bool
	out := captureStdout(t, func() {
		reportVersionCheckFailures(nil, 3, false, &warnings)
	})

	if warnings {
		t.Error("no failure must not raise a warning")
	}
	if strings.TrimSpace(out) != "" {
		t.Errorf("expected silence, got:\n%s", out)
	}
}

func TestIsRateLimit(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"403 from the API", fmt.Errorf("GitHub API 403 for ss-keel-gorm"), true},
		{"404 is a missing repo", fmt.Errorf("GitHub API 404 for ss-keel-nope"), false},
		{"network failure", errors.New("dial tcp: lookup api.github.com: no such host"), false},
		{"nil", nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isRateLimit(c.err); got != c.want {
				t.Errorf("isRateLimit(%v) = %v, want %v", c.err, got, c.want)
			}
		})
	}
}
