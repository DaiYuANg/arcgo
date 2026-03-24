package configx

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── helpers ───────────────────────────────────────────────────────────────────

// writeYAML writes content to path, failing the test on error.
func writeYAML(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}

// tempYAML creates a temp YAML config file and returns its path.
func tempYAML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	writeYAML(t, path, content)
	return path
}

// startWatcher starts w.Start in a background goroutine with a cancellable
// context and registers t.Cleanup to cancel the context and close the watcher.
func startWatcher(t *testing.T, w *Watcher) context.CancelFunc {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		_ = w.Close()
	})
	go func() { _ = w.Start(ctx) }()
	// Give fsnotify a moment to register its directory watch before we write.
	time.Sleep(50 * time.Millisecond)
	return cancel
}

// waitForChange waits up to timeout for the next value on ch.
func waitForChange(t *testing.T, ch <-chan *Config, timeout time.Duration) *Config {
	t.Helper()
	select {
	case cfg := <-ch:
		return cfg
	case <-time.After(timeout):
		t.Fatal("timed out waiting for config reload")
		return nil
	}
}

// ── construction ──────────────────────────────────────────────────────────────

func TestNewWatcher_InitialLoad(t *testing.T) {
	path := tempYAML(t, "name: arcgo\nport: 8080\n")

	w, err := NewWatcher(WithFiles(path))
	require.NoError(t, err)

	assert.Equal(t, "arcgo", w.Config().GetString("name"))
	assert.Equal(t, 8080, w.Config().GetInt("port"))
}

func TestNewWatcher_WithDefaults(t *testing.T) {
	w, err := NewWatcher(
		WithDefaults(map[string]any{
			"name": "default-app",
			"port": 9090,
		}),
	)
	require.NoError(t, err)
	assert.Equal(t, "default-app", w.Config().GetString("name"))
	assert.Equal(t, 9090, w.Config().GetInt("port"))
}

func TestNewWatcher_NoFiles_InitialConfigStillWorks(t *testing.T) {
	w, err := NewWatcher(
		WithDefaults(map[string]any{"key": "val"}),
	)
	require.NoError(t, err)
	assert.Equal(t, "val", w.Config().GetString("key"))
}

func TestNewWatcher_BadFile_ReturnsError(t *testing.T) {
	_, err := NewWatcher(WithFiles("/nonexistent/path/config.yaml"))
	// loadConfigFromOptions → loadFiles → file.Provider.Load returns an error
	assert.Error(t, err)
}

// ── hot reload ────────────────────────────────────────────────────────────────

func TestWatcher_HotReload_ValueChanges(t *testing.T) {
	path := tempYAML(t, "name: before\nport: 1111\n")

	w, err := NewWatcher(
		WithFiles(path),
		WithWatchDebounce(30*time.Millisecond),
	)
	require.NoError(t, err)

	changed := make(chan *Config, 1)
	w.OnChange(func(cfg *Config, err error) {
		require.NoError(t, err)
		changed <- cfg
	})

	startWatcher(t, w)

	writeYAML(t, path, "name: after\nport: 2222\n")

	newCfg := waitForChange(t, changed, 3*time.Second)
	assert.Equal(t, "after", newCfg.GetString("name"))
	assert.Equal(t, 2222, newCfg.GetInt("port"))

	// w.Config() must also reflect the new values.
	assert.Equal(t, "after", w.Config().GetString("name"))
}

func TestWatcher_HotReload_MultipleReloads(t *testing.T) {
	path := tempYAML(t, "version: 1\n")

	w, err := NewWatcher(
		WithFiles(path),
		WithWatchDebounce(30*time.Millisecond),
	)
	require.NoError(t, err)

	var reloadCount atomic.Int32
	w.OnChange(func(cfg *Config, err error) {
		if err == nil {
			reloadCount.Add(1)
		}
	})

	startWatcher(t, w)

	// Perform three sequential writes, each separated by more than the debounce
	// window so each one triggers its own reload.
	for i := range 3 {
		writeYAML(t, path, fmt.Sprintf("version: %d\n", i+2))
		time.Sleep(120 * time.Millisecond)
	}

	// Allow the last debounce timer to fire.
	time.Sleep(100 * time.Millisecond)

	assert.EqualValues(t, 3, reloadCount.Load())
	assert.Equal(t, 4, w.Config().GetInt("version"))
}

// ── debounce ──────────────────────────────────────────────────────────────────

func TestWatcher_Debounce_CollapsesRapidWrites(t *testing.T) {
	path := tempYAML(t, "counter: 0\n")

	w, err := NewWatcher(
		WithFiles(path),
		// Large debounce so all rapid writes are collapsed.
		WithWatchDebounce(300*time.Millisecond),
	)
	require.NoError(t, err)

	var reloadCount atomic.Int32
	w.OnChange(func(cfg *Config, _ error) {
		if cfg != nil {
			reloadCount.Add(1)
		}
	})

	startWatcher(t, w)

	// Write 5 times in rapid succession (well within the debounce window).
	for i := 1; i <= 5; i++ {
		writeYAML(t, path, fmt.Sprintf("counter: %d\n", i))
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for debounce to fire and reload.
	time.Sleep(500 * time.Millisecond)

	// Only one reload should have occurred despite 5 writes.
	assert.EqualValues(t, 1, reloadCount.Load())
	// And it should reflect the last written value.
	assert.Equal(t, 5, w.Config().GetInt("counter"))
}

// ── OnChange ──────────────────────────────────────────────────────────────────

func TestWatcher_OnChange_NilHandlerIsIgnored(t *testing.T) {
	w, err := NewWatcher(
		WithDefaults(map[string]any{"x": 1}),
	)
	require.NoError(t, err)

	// Should not panic.
	w.OnChange(nil)
	assert.Len(t, w.loadSubscribers(), 0)
}

func TestWatcher_OnChange_MultipleSubscribers(t *testing.T) {
	path := tempYAML(t, "val: 10\n")

	w, err := NewWatcher(
		WithFiles(path),
		WithWatchDebounce(30*time.Millisecond),
	)
	require.NoError(t, err)

	const n = 3
	channels := make([]chan int, n)
	for i := range channels {
		ch := make(chan int, 1)
		channels[i] = ch
		capturedCh := ch // capture for closure
		w.OnChange(func(cfg *Config, err error) {
			if err == nil {
				capturedCh <- cfg.GetInt("val")
			}
		})
	}

	startWatcher(t, w)

	writeYAML(t, path, "val: 99\n")

	// All subscribers must receive the new value.
	for i, ch := range channels {
		select {
		case got := <-ch:
			assert.Equal(t, 99, got, "subscriber %d", i)
		case <-time.After(3 * time.Second):
			t.Fatalf("subscriber %d timed out", i)
		}
	}
}

func TestWatcher_OnChange_OrderIsPreserved(t *testing.T) {
	path := tempYAML(t, "n: 0\n")

	w, err := NewWatcher(
		WithFiles(path),
		WithWatchDebounce(30*time.Millisecond),
	)
	require.NoError(t, err)

	var mu sync.Mutex
	order := make([]int, 0, 3)

	for i := range 3 {
		i := i
		w.OnChange(func(_ *Config, _ error) {
			mu.Lock()
			order = append(order, i)
			mu.Unlock()
		})
	}

	startWatcher(t, w)

	writeYAML(t, path, "n: 1\n")
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []int{0, 1, 2}, order)
}

func TestWatcher_OnChange_RegisterDuringNotify_AppliesOnNextNotify(t *testing.T) {
	w, err := NewWatcher(
		WithDefaults(map[string]any{"n": 1}),
	)
	require.NoError(t, err)

	registered := false
	lateCalls := make(chan int, 1)
	var registerOnce sync.Once

	w.OnChange(func(_ *Config, err error) {
		if err != nil {
			return
		}
		registerOnce.Do(func() {
			w.OnChange(func(cfg *Config, err error) {
				if err == nil {
					lateCalls <- cfg.GetInt("n")
				}
			})
			registered = true
		})
	})

	w.notify(w.Config(), nil)
	assert.True(t, registered)

	select {
	case got := <-lateCalls:
		t.Fatalf("late subscriber should not run on the same notify, got %d", got)
	default:
	}

	w.notify(w.Config(), nil)
	assert.Equal(t, 1, <-lateCalls)
}

// ── error handling ────────────────────────────────────────────────────────────

func TestWatcher_WatchErrHandler_CalledOnReloadError(t *testing.T) {
	// Start with a valid YAML file.
	path := tempYAML(t, "ok: true\n")

	var handledErr atomic.Value
	errCh := make(chan error, 1)

	w, err := NewWatcher(
		WithFiles(path),
		WithWatchDebounce(30*time.Millisecond),
		WithWatchErrHandler(func(e error) {
			if handledErr.CompareAndSwap(nil, e) {
				errCh <- e
			}
		}),
	)
	require.NoError(t, err)

	startWatcher(t, w)

	// Write invalid YAML to force a parse error on reload.
	require.NoError(t, os.WriteFile(path, []byte(":\tinvalid: yaml: [\n"), 0o600))

	select {
	case e := <-errCh:
		assert.Error(t, e)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for watch error handler")
	}

	// The previous (valid) config must still be accessible.
	assert.True(t, w.Config().GetBool("ok"))
}

func TestWatcher_OnChange_CalledWithErrorOnReloadFailure(t *testing.T) {
	path := tempYAML(t, "healthy: true\n")

	w, err := NewWatcher(
		WithFiles(path),
		WithWatchDebounce(30*time.Millisecond),
	)
	require.NoError(t, err)

	errCh := make(chan error, 1)
	w.OnChange(func(cfg *Config, err error) {
		if err != nil {
			errCh <- err
		}
	})

	startWatcher(t, w)

	require.NoError(t, os.WriteFile(path, []byte(":\tinvalid: yaml: [\n"), 0o600))

	select {
	case e := <-errCh:
		assert.Error(t, e)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for OnChange error callback")
	}
}

// ── Close / lifecycle ─────────────────────────────────────────────────────────

func TestWatcher_Close_StopsReloads(t *testing.T) {
	path := tempYAML(t, "active: true\n")

	w, err := NewWatcher(
		WithFiles(path),
		WithWatchDebounce(30*time.Millisecond),
	)
	require.NoError(t, err)

	var reloadCount atomic.Int32
	w.OnChange(func(cfg *Config, _ error) {
		if cfg != nil {
			reloadCount.Add(1)
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	started := make(chan struct{})
	go func() {
		close(started)
		_ = w.Start(ctx)
	}()

	<-started
	time.Sleep(50 * time.Millisecond)

	// Close the watcher before any file change.
	require.NoError(t, w.Close())

	// Write after close – should NOT trigger a reload.
	writeYAML(t, path, "active: false\n")
	time.Sleep(200 * time.Millisecond)

	assert.EqualValues(t, 0, reloadCount.Load())
}

func TestWatcher_Close_IsIdempotent(t *testing.T) {
	w, err := NewWatcher(WithDefaults(map[string]any{"x": 1}))
	require.NoError(t, err)

	require.NoError(t, w.Close())
	require.NoError(t, w.Close()) // must not panic or return an error
}

func TestWatcher_Start_ReturnsWhenContextCancelled(t *testing.T) {
	path := tempYAML(t, "x: 1\n")

	w, err := NewWatcher(WithFiles(path))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- w.Start(ctx) }()

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not return after context cancellation")
	}
}

func TestWatcher_Start_NoFiles_ReturnsOnContextCancel(t *testing.T) {
	w, err := NewWatcher(WithDefaults(map[string]any{"k": "v"}))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- w.Start(ctx) }()

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not return after context cancellation (no files)")
	}
}

// ── env separator ─────────────────────────────────────────────────────────────

func TestWatcher_EnvSeparator_DoubleUnderscore(t *testing.T) {
	t.Setenv("APP_DB__HOST", "localhost")
	t.Setenv("APP_MAX_RETRY", "5")

	w, err := NewWatcher(
		WithEnvPrefix("APP"),
		WithEnvSeparator("__"),
		WithPriority(SourceEnv),
	)
	require.NoError(t, err)

	// double-underscore → nested path
	assert.Equal(t, "localhost", w.Config().GetString("db.host"))
	// single underscore → flat key preserved
	assert.Equal(t, "5", w.Config().GetString("max_retry"))
}

// ── unsupported file format ───────────────────────────────────────────────────

func TestNewWatcher_UnsupportedFileFormat_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	iniPath := filepath.Join(dir, "config.ini")
	require.NoError(t, os.WriteFile(iniPath, []byte("[section]\nkey=value\n"), 0o600))

	_, err := NewWatcher(WithFiles(iniPath))
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnsupportedFileFormat))
}

// ── concurrent safety ─────────────────────────────────────────────────────────

func TestWatcher_ConcurrentConfigReads(t *testing.T) {
	path := tempYAML(t, "counter: 0\n")

	w, err := NewWatcher(
		WithFiles(path),
		WithWatchDebounce(20*time.Millisecond),
	)
	require.NoError(t, err)

	cancel := startWatcher(t, w)
	defer cancel()

	var wg sync.WaitGroup

	// Writer goroutine: update the file several times.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 1; i <= 5; i++ {
			writeYAML(t, path, fmt.Sprintf("counter: %d\n", i))
			time.Sleep(80 * time.Millisecond)
		}
	}()

	// Reader goroutines: call w.Config() concurrently with reloads.
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 20 {
				cfg := w.Config()
				// cfg must never be nil – panics here are caught as test failures.
				_ = cfg.GetInt("counter")
				time.Sleep(5 * time.Millisecond)
			}
		}()
	}

	wg.Wait()
}

func TestWatcherT_HotReload_TypedValueChanges(t *testing.T) {
	type typedCfg struct {
		Name string `validate:"required"`
		Port int    `validate:"gte=1"`
	}

	path := tempYAML(t, "name: before\nport: 1111\n")
	w, err := NewWatcherT[typedCfg](
		WithFiles(path),
		WithWatchDebounce(30*time.Millisecond),
		WithValidateLevel(ValidateLevelStruct),
	)
	require.NoError(t, err)

	changed := make(chan typedCfg, 1)
	w.OnChange(func(cfg typedCfg, err error) {
		require.NoError(t, err)
		changed <- cfg
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		_ = w.Close()
	})
	go func() { _ = w.Start(ctx) }()
	time.Sleep(50 * time.Millisecond)

	writeYAML(t, path, "name: after\nport: 2222\n")
	select {
	case got := <-changed:
		assert.Equal(t, "after", got.Name)
		assert.Equal(t, 2222, got.Port)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for typed watcher reload")
	}

	latest := w.Config()
	assert.Equal(t, "after", latest.Name)
	assert.Equal(t, 2222, latest.Port)
}
