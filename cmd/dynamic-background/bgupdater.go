package main

import (
	"os/exec"
	"sync"
	"time"
)

// BgUpdater manages swaybg updates with throttling to reduce flicker.
type BgUpdater struct {
	mu         sync.Mutex
	lastUpdate time.Time
	interval   time.Duration
	output     string
	imagePath  string
	mode       string
	pending    bool
}

// NewBgUpdater creates a new background updater.
func NewBgUpdater(output, imagePath, mode string, interval time.Duration) *BgUpdater {
	return &BgUpdater{
		output:    output,
		imagePath: imagePath,
		mode:      mode,
		interval:  interval,
	}
}

// Update triggers a background update. Throttled to interval.
func (b *BgUpdater) Update() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if time.Since(b.lastUpdate) < b.interval {
		b.pending = true
		return
	}

	b.apply()
}

// apply executes swaymsg. Must be called with lock held.
func (b *BgUpdater) apply() {
	cmd := exec.Command("swaymsg", "output", b.output, "bg", b.imagePath, b.mode)
	_ = cmd.Run()
	b.lastUpdate = time.Now()
	b.pending = false
}

// Flush applies pending update if any.
func (b *BgUpdater) Flush() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.pending {
		b.apply()
	}
}
