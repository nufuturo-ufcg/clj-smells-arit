package cmd

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestProgressReporterIsSilentWhenDisabled(t *testing.T) {
	var output bytes.Buffer
	progress := newProgressReporter(2, &output, false)
	progress.Add(1)
	progress.Finish()

	if output.Len() != 0 {
		t.Fatalf("disabled progress should be silent, got %q", output.String())
	}
}

func TestProgressReporterRendersOneTerminalLine(t *testing.T) {
	var output bytes.Buffer
	progress := newProgressReporter(2, &output, true)
	progress.Add(2)
	progress.Finish()

	text := output.String()
	if !strings.Contains(text, "Arit · analyzing 2/2") {
		t.Fatalf("expected completed progress line, got %q", text)
	}
	if !strings.Contains(text, "ETA done") {
		t.Fatalf("expected completed ETA, got %q", text)
	}
	if !strings.Contains(text, "\033[2K") {
		t.Fatalf("expected terminal line clearing sequence, got %q", text)
	}
}

func TestProgressEnabledHonorsQuietAndVerbose(t *testing.T) {
	oldQuiet, oldVerbose := quietFlag, verboseFlag
	t.Cleanup(func() {
		quietFlag, verboseFlag = oldQuiet, oldVerbose
	})

	quietFlag = true
	verboseFlag = false
	if progressEnabled() {
		t.Fatal("quiet mode must disable progress")
	}

	quietFlag = false
	verboseFlag = true
	if progressEnabled() {
		t.Fatal("verbose mode must disable progress")
	}
}

func TestFormatProgressDuration(t *testing.T) {
	tests := []struct {
		name string
		in   time.Duration
		want string
	}{
		{name: "subsecond", in: 500 * time.Millisecond, want: "0s"},
		{name: "seconds", in: 12 * time.Second, want: "12s"},
		{name: "minutes", in: 2*time.Minute + 3*time.Second, want: "2m03s"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := formatProgressDuration(test.in); got != test.want {
				t.Fatalf("formatProgressDuration(%s) = %q, want %q", test.in, got, test.want)
			}
		})
	}
}
