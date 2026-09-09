// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tuikit

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ProgressUnit specifies how progress amounts and throughput are formatted.
type ProgressUnit int

const (
	// UnitCount formats progress as integer item counts (e.g. 1,420 items, 350 items/s).
	UnitCount ProgressUnit = iota
	// UnitBytes formats progress as human-readable byte sizes (e.g. 45.2 MB, 12.8 MB/s).
	UnitBytes
	// UnitNone formats progress without specific units.
	UnitNone
)

// ProgressOption configures a Progress instance.
type ProgressOption func(*Progress)

// WithWriter sets the destination output writer (defaults to os.Stdout).
func WithWriter(w io.Writer) ProgressOption {
	return func(p *Progress) {
		p.writer = w
	}
}

// WithTotal sets the expected total work units.
func WithTotal(total int64) ProgressOption {
	return func(p *Progress) {
		p.total = total
	}
}

// WithUnit sets the formatting unit (UnitCount, UnitBytes, UnitNone).
func WithUnit(u ProgressUnit) ProgressOption {
	return func(p *Progress) {
		p.unit = u
	}
}

// WithBarWidth sets the visual character width of the progress bar fill.
func WithBarWidth(w int) ProgressOption {
	return func(p *Progress) {
		if w > 0 {
			p.barWidth = w
		}
	}
}

// WithPrefix sets the task prefix description (e.g. "Packing", "Extracting").
func WithPrefix(prefix string) ProgressOption {
	return func(p *Progress) {
		p.prefix = prefix
	}
}

// WithInteractive explicitly forces interactive (TTY) or non-interactive mode.
func WithInteractive(interactive bool) ProgressOption {
	return func(p *Progress) {
		p.isTTY = interactive
		p.ttyForced = true
	}
}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Progress manages live terminal feedback, throughput, ETA calculations, and progress rendering.
type Progress struct {
	mu          sync.Mutex
	writer      io.Writer
	prefix      string
	total       int64
	current     int64
	extraBytes  int64 // secondary counter (e.g. byte size when primary is item count)
	unit        ProgressUnit
	barWidth    int
	isTTY       bool
	ttyForced   bool
	startTime   time.Time
	lastRender  time.Time
	lastLogTime time.Time
	lastLogVal  int64
	spinnerIdx  int
	throttle    time.Duration
	logInterval time.Duration
	done        bool
}

// NewProgress creates and initializes a new Progress widget.
func NewProgress(opts ...ProgressOption) *Progress {
	p := &Progress{
		writer:      os.Stdout,
		barWidth:    20,
		unit:        UnitCount,
		throttle:    60 * time.Millisecond,
		logInterval: 4 * time.Second,
		startTime:   time.Now(),
	}

	for _, opt := range opts {
		opt(p)
	}

	if !p.ttyForced {
		p.isTTY = ProbeTerminal(p.writer)
	}

	return p
}

// SetTotal dynamically updates the total expected work units.
func (p *Progress) SetTotal(total int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.total = total
}

// SetPrefix changes the leading operation label.
func (p *Progress) SetPrefix(prefix string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.prefix = prefix
}

// SetExtraBytes updates an optional secondary byte counter (useful when unit is UnitCount but bytes are tracked).
func (p *Progress) SetExtraBytes(bytes int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.extraBytes = bytes
}

// Add increments the current progress by delta and renders if throttled interval elapsed.
func (p *Progress) Add(delta int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.current += delta
	p.renderLocked(false)
}

// SetCurrent sets the current progress value and renders if throttled interval elapsed.
func (p *Progress) SetCurrent(val int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.current = val
	p.renderLocked(false)
}

// Render forces an immediate render of the current progress state.
func (p *Progress) Render() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.renderLocked(true)
}

func (p *Progress) renderLocked(force bool) {
	if p.done || p.writer == nil {
		return
	}

	now := time.Now()
	if !force && p.lastRender.Add(p.throttle).After(now) {
		return
	}
	p.lastRender = now

	elapsed := now.Sub(p.startTime).Seconds()
	if elapsed <= 0 {
		elapsed = 0.001
	}

	if p.isTTY {
		p.renderTTY(now, elapsed)
	} else {
		p.renderNonTTY(now, elapsed, force)
	}
}

func (p *Progress) renderTTY(now time.Time, elapsed float64) {
	p.spinnerIdx = (p.spinnerIdx + 1) % len(spinnerFrames)
	spin := spinnerFrames[p.spinnerIdx]

	var sb strings.Builder
	// Carriage return + ANSI line clear
	sb.WriteString("\r\033[K")

	if p.prefix != "" {
		sb.WriteString(p.prefix)
		sb.WriteString(" ")
	}

	if p.total > 0 {
		ratio := float64(p.current) / float64(p.total)
		if ratio > 1.0 {
			ratio = 1.0
		}
		pct := int(ratio * 100)

		bar := RenderBar(ratio, p.barWidth)
		sb.WriteString("[")
		sb.WriteString(bar)
		fmt.Fprintf(&sb, "] %3d%% ", pct)

		if p.unit == UnitBytes {
			fmt.Fprintf(&sb, "(%s / %s)", FormatBytes(uint64(p.current)), FormatBytes(uint64(p.total)))
		} else {
			fmt.Fprintf(&sb, "(%s / %s)", formatNumber(p.current), formatNumber(p.total))
			if p.extraBytes > 0 {
				fmt.Fprintf(&sb, " [%s]", FormatBytes(uint64(p.extraBytes)))
			}
		}

		// Throughput & ETA
		rate := float64(p.current) / elapsed
		if p.unit == UnitBytes {
			fmt.Fprintf(&sb, "  %.1f MB/s", rate/(1024*1024))
		} else if rate > 0 {
			fmt.Fprintf(&sb, "  %.0f/s", rate)
			if p.extraBytes > 0 {
				byteRate := float64(p.extraBytes) / elapsed
				fmt.Fprintf(&sb, " (%.1f MB/s)", byteRate/(1024*1024))
			}
		}

		if rate > 0 && p.current < p.total {
			remSec := float64(p.total-p.current) / rate
			fmt.Fprintf(&sb, "  [ETA: %s]", formatDuration(time.Duration(remSec)*time.Second))
		}
	} else {
		// Indeterminate / streaming progress
		sb.WriteString(spin)
		sb.WriteString(" ")

		if p.unit == UnitBytes {
			sb.WriteString(FormatBytes(uint64(p.current)))
			rate := float64(p.current) / elapsed
			fmt.Fprintf(&sb, "  (%.1f MB/s)", rate/(1024*1024))
		} else {
			sb.WriteString(formatNumber(p.current) + " items")
			if p.extraBytes > 0 {
				fmt.Fprintf(&sb, " (%s)", FormatBytes(uint64(p.extraBytes)))
			}
			rate := float64(p.current) / elapsed
			if rate > 0 {
				fmt.Fprintf(&sb, "  %.0f items/s", rate)
			}
			if p.extraBytes > 0 {
				byteRate := float64(p.extraBytes) / elapsed
				fmt.Fprintf(&sb, " (%.1f MB/s)", byteRate/(1024*1024))
			}
		}
	}

	_, _ = io.WriteString(p.writer, sb.String())
}

func (p *Progress) renderNonTTY(now time.Time, elapsed float64, force bool) {
	// Only log periodically in non-TTY to keep CI/pipes clean
	if !force && p.lastLogTime.Add(p.logInterval).After(now) {
		return
	}
	p.lastLogTime = now
	p.lastLogVal = p.current

	var sb strings.Builder
	if p.prefix != "" {
		sb.WriteString(p.prefix)
		sb.WriteString(": ")
	}

	if p.total > 0 {
		pct := int(float64(p.current) / float64(p.total) * 100)
		if p.unit == UnitBytes {
			fmt.Fprintf(
				&sb,
				"%d%% (%s / %s)",
				pct,
				FormatBytes(uint64(p.current)),
				FormatBytes(uint64(p.total)),
			)
		} else {
			fmt.Fprintf(
				&sb,
				"%d%% (%s / %s)",
				pct,
				formatNumber(p.current),
				formatNumber(p.total),
			)
		}
	} else {
		if p.unit == UnitBytes {
			sb.WriteString(FormatBytes(uint64(p.current)))
		} else {
			sb.WriteString(formatNumber(p.current) + " items")
			if p.extraBytes > 0 {
				fmt.Fprintf(&sb, " (%s)", FormatBytes(uint64(p.extraBytes)))
			}
		}
	}

	rate := float64(p.current) / elapsed
	if p.unit == UnitBytes {
		fmt.Fprintf(&sb, " [%.1f MB/s]\n", rate/(1024*1024))
	} else {
		fmt.Fprintf(&sb, " [%.0f items/s]\n", rate)
	}

	_, _ = io.WriteString(p.writer, sb.String())
}

// Done stops the progress widget, clears the TTY line if needed, and marks it finished.
func (p *Progress) Done() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.done {
		return
	}
	p.done = true

	if p.isTTY && p.writer != nil {
		_, _ = io.WriteString(p.writer, "\r\033[K")
	}
}

// Finish marks the progress complete and prints a formatted summary line.
func (p *Progress) Finish(summaryMsg string) {
	p.Done()
	if summaryMsg != "" && p.writer != nil {
		fmt.Fprintln(p.writer, summaryMsg)
	}
}

// ProgressReader wraps an io.Reader and automatically updates a Progress instance as data is read.
type ProgressReader struct {
	reader io.Reader
	p      *Progress
}

// NewReader returns an io.Reader that forwards read byte counts to the progress tracker.
func (p *Progress) NewReader(r io.Reader) *ProgressReader {
	return &ProgressReader{
		reader: r,
		p:      p,
	}
}

// Read implements io.Reader.
func (pr *ProgressReader) Read(buf []byte) (int, error) {
	n, err := pr.reader.Read(buf)
	if n > 0 && pr.p != nil {
		pr.p.Add(int64(n))
	}
	return n, err
}

// FormatBytes formats byte counts into human-readable strings (e.g. "4.2 MB", "512 KB").
func FormatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func formatNumber(n int64) string {
	if n < 0 {
		return strconv.FormatInt(n, 10)
	}
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return s
	}

	var res []byte
	rem := len(s) % 3
	if rem > 0 {
		res = append(res, s[:rem]...)
	}
	for i := rem; i < len(s); i += 3 {
		if len(res) > 0 {
			res = append(res, ',')
		}
		res = append(res, s[i:i+3]...)
	}
	return string(res)
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	s := int(d.Seconds())
	if s < 60 {
		return fmt.Sprintf("%ds", s)
	}
	m := s / 60
	s %= 60
	if m < 60 {
		return fmt.Sprintf("%dm %02ds", m, s)
	}
	h := m / 60
	m %= 60
	return fmt.Sprintf("%dh %02dm", h, m)
}
