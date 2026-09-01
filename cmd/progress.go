package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/term"
)

const progressRefreshInterval = 150 * time.Millisecond

type progressReporter interface {
	Add(int)
	Finish()
}

type terminalProgress struct {
	total   int64
	current atomic.Int64
	started time.Time

	writer io.Writer
	stop   chan struct{}
	done   chan struct{}
	once   sync.Once
}

type noopProgress struct{}

func (noopProgress) Add(int) {}

func (noopProgress) Finish() {}

func newProgressReporter(total int, writer io.Writer, enabled bool) progressReporter {
	if total <= 0 || writer == nil || !enabled {
		return noopProgress{}
	}

	progress := &terminalProgress{
		total:   int64(total),
		started: time.Now(),
		writer:  writer,
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}
	go progress.refreshLoop()
	return progress
}

func progressEnabled() bool {
	if quietFlag || verboseFlag || os.Getenv("CI") != "" {
		return false
	}
	return term.IsTerminal(int(os.Stderr.Fd()))
}

func (p *terminalProgress) Add(delta int) {
	if delta > 0 {
		p.current.Add(int64(delta))
	}
}

func (p *terminalProgress) refreshLoop() {
	ticker := time.NewTicker(progressRefreshInterval)
	defer ticker.Stop()
	defer close(p.done)

	for {
		select {
		case <-ticker.C:
			p.render(false)
		case <-p.stop:
			return
		}
	}
}

func (p *terminalProgress) Finish() {
	p.once.Do(func() {
		close(p.stop)
		<-p.done
		p.current.Store(p.total)
		p.render(true)
	})
}

func (p *terminalProgress) render(done bool) {
	current := p.current.Load()
	if current > p.total {
		current = p.total
	}
	elapsed := time.Since(p.started)
	if elapsed < time.Millisecond {
		elapsed = time.Millisecond
	}

	percent := float64(current) * 100 / float64(p.total)
	rate := float64(current) / elapsed.Seconds()

	eta := "--"
	if rate > 0 && current < p.total {
		remaining := time.Duration(float64(p.total-current)/rate) * time.Second
		eta = formatProgressDuration(remaining)
	}
	if done {
		eta = "done"
	}

	line := fmt.Sprintf("Arit · analyzing %d/%d · %5.1f%% · %s/s · elapsed %s · ETA %s",
		current,
		p.total,
		percent,
		formatProgressRate(rate),
		formatProgressDuration(elapsed),
		eta,
	)
	// Clear the previous line before drawing the next one. This keeps the
	// renderer to one terminal line and avoids emitting output for every file.
	fmt.Fprintf(p.writer, "\r\033[2K%s", line)
	if done {
		fmt.Fprintln(p.writer)
	}
}

func formatProgressRate(rate float64) string {
	switch {
	case rate >= 1000:
		return fmt.Sprintf("%.1fk", rate/1000)
	case rate >= 100:
		return fmt.Sprintf("%.0f", rate)
	default:
		return fmt.Sprintf("%.1f", rate)
	}
}

func formatProgressDuration(duration time.Duration) string {
	if duration < time.Second {
		return "0s"
	}
	if duration < time.Minute {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	}
	minutes := int(duration / time.Minute)
	seconds := int(duration/time.Second) % 60
	return strings.TrimSpace(fmt.Sprintf("%dm%02ds", minutes, seconds))
}
