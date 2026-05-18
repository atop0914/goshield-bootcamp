package config

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// Watcher watches a config file for changes and triggers a reload callback.
// It uses polling (no external dependencies) at a configurable interval.
type Watcher struct {
	path     string
	interval time.Duration
	onChange func(*Config)
	onError  func(error)

	mu       sync.Mutex
	lastMod  time.Time
	lastSize int64
	stopCh   chan struct{}
	done     chan struct{}
	running  bool
}

// WatcherOption configures a Watcher.
type WatcherOption func(*Watcher)

// WithInterval sets the polling interval. Default is 5 seconds.
func WithInterval(d time.Duration) WatcherOption {
	return func(w *Watcher) {
		w.interval = d
	}
}

// WithOnError sets the error callback. Default is to log to stderr.
func WithOnError(fn func(error)) WatcherOption {
	return func(w *Watcher) {
		w.onError = fn
	}
}

// NewWatcher creates a new file watcher for the given config path.
// The onChange callback is called with the new Config whenever the file changes.
func NewWatcher(path string, onChange func(*Config), opts ...WatcherOption) (*Watcher, error) {
	if onChange == nil {
		return nil, fmt.Errorf("config: onChange callback is required")
	}

	// Verify file exists and is readable
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("config: cannot watch %s: %w", path, err)
	}

	w := &Watcher{
		path:     path,
		interval: 5 * time.Second,
		onChange: onChange,
		lastMod:  info.ModTime(),
		lastSize: info.Size(),
		stopCh:   make(chan struct{}),
		done:     make(chan struct{}),
		onError: func(err error) {
			fmt.Fprintf(os.Stderr, "config watcher: %v\n", err)
		},
	}

	for _, opt := range opts {
		opt(w)
	}

	return w, nil
}

// Start begins watching the config file. It returns immediately.
// Call Stop() to stop watching.
func (w *Watcher) Start() {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return
	}
	w.running = true
	w.mu.Unlock()

	go w.watch()
}

// Stop stops watching and waits for the goroutine to exit.
func (w *Watcher) Stop() {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return
	}
	w.running = false
	w.mu.Unlock()

	close(w.stopCh)
	<-w.done
}

// IsRunning returns true if the watcher is currently running.
func (w *Watcher) IsRunning() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.running
}

func (w *Watcher) watch() {
	defer close(w.done)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.check()
		}
	}
}

func (w *Watcher) check() {
	info, err := os.Stat(w.path)
	if err != nil {
		w.onError(fmt.Errorf("stat %s: %w", w.path, err))
		return
	}

	modTime := info.ModTime()
	size := info.Size()

	if modTime.Equal(w.lastMod) && size == w.lastSize {
		return
	}

	w.lastMod = modTime
	w.lastSize = size

	cfg, err := LoadFile(w.path)
	if err != nil {
		w.onError(fmt.Errorf("reload config: %w", err))
		return
	}

	if err := cfg.Validate(); err != nil {
		w.onError(fmt.Errorf("invalid config after reload: %w", err))
		return
	}

	w.onChange(cfg)
}
