// Package lock provides distributed lock functionality.
package lock

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	collectionmapping "github.com/DaiYuANg/arcgo/collectionx/mapping"
	"github.com/DaiYuANg/arcgo/kvx"
)

var (
	// ErrLockNotAcquired is returned when the lock could not be acquired.
	ErrLockNotAcquired = errors.New("lock: could not acquire lock")
	// ErrLockNotHeld is returned when the lock is not held by the caller.
	ErrLockNotHeld = errors.New("lock: lock not held")
	// ErrLockExpired is returned when the lock has expired.
	ErrLockExpired = errors.New("lock: lock has expired")
)

// Lock represents a distributed lock.
type Lock struct {
	client     kvx.Lock
	key        string
	identifier string
	ttl        time.Duration
	autoExtend bool
	stopExtend chan struct{}
	extendWG   sync.WaitGroup
}

// Options contains options for creating a lock.
type Options struct {
	TTL        time.Duration
	AutoExtend bool
}

// DefaultOptions returns default lock options.
func DefaultOptions() *Options {
	return &Options{
		TTL:        30 * time.Second,
		AutoExtend: true,
	}
}

// New creates a new Lock instance.
func New(client kvx.Lock, key string, opts *Options) *Lock {
	opts = resolveOptions(opts)

	return &Lock{
		client:     client,
		key:        key,
		identifier: generateIdentifier(),
		ttl:        opts.TTL,
		autoExtend: opts.AutoExtend,
		stopExtend: make(chan struct{}),
	}
}

func resolveOptions(opts *Options) *Options {
	if opts != nil {
		return opts
	}
	return DefaultOptions()
}

// Acquire acquires the lock.
func (l *Lock) Acquire(ctx context.Context) error {
	acquired, err := l.client.Acquire(ctx, l.key, l.identifier, l.ttl)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	if !acquired {
		return ErrLockNotAcquired
	}

	if l.autoExtend {
		l.startAutoExtend(ctx)
	}

	return nil
}

// TryAcquire tries to acquire the lock with a timeout.
func (l *Lock) TryAcquire(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if time.Now().After(deadline) {
			return ErrLockNotAcquired
		}

		err := l.Acquire(ctx)
		if err == nil {
			return nil
		}
		if !errors.Is(err, ErrLockNotAcquired) {
			return err
		}

		// Wait a bit before retrying
		select {
		case <-ctx.Done():
			return fmt.Errorf("lock acquisition canceled: %w", ctx.Err())
		case <-time.After(100 * time.Millisecond):
			continue
		}
	}
}

// Release releases the lock.
func (l *Lock) Release(ctx context.Context) error {
	l.stopAutoExtend()

	released, err := l.client.Release(ctx, l.key, l.identifier)
	if err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}
	if !released {
		return ErrLockNotHeld
	}
	return nil
}

// Extend extends the lock TTL.
func (l *Lock) Extend(ctx context.Context, ttl time.Duration) error {
	extended, err := l.client.Extend(ctx, l.key, l.identifier, ttl)
	if err != nil {
		return fmt.Errorf("failed to extend lock: %w", err)
	}
	if !extended {
		return ErrLockNotHeld
	}
	return nil
}

// IsHeld checks if the lock is still held.
func (l *Lock) IsHeld(ctx context.Context) (bool, error) {
	// Try to extend with 0 TTL - this will only succeed if we hold the lock
	held, err := l.client.Extend(ctx, l.key, l.identifier, l.ttl)
	if err != nil {
		return false, fmt.Errorf("check lock state: %w", err)
	}
	return held, nil
}

// startAutoExtend starts the auto-extend goroutine.
func (l *Lock) startAutoExtend(ctx context.Context) {
	l.extendWG.Go(func() {
		// Extend at 1/3 of TTL intervals
		ticker := time.NewTicker(l.ttl / 3)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-l.stopExtend:
				return
			case <-ticker.C:
				_, err := l.client.Extend(ctx, l.key, l.identifier, l.ttl)
				if err != nil {
					// Log error but continue trying
					return
				}
			}
		}
	})
}

// stopAutoExtend stops the auto-extend goroutine.
func (l *Lock) stopAutoExtend() {
	if l.autoExtend {
		close(l.stopExtend)
		l.extendWG.Wait()
	}
}

// generateIdentifier generates a unique identifier for this lock instance.
func generateIdentifier() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fallbackIdentifier()
	}
	return hex.EncodeToString(b)
}

func fallbackIdentifier() string {
	return hex.EncodeToString([]byte(strconv.FormatInt(time.Now().UnixNano(), 10)))
}

// Manager manages multiple locks.
type Manager struct {
	client kvx.Lock
	locks  *collectionmapping.ConcurrentMap[string, *Lock]
}

// NewManager creates a new Manager.
func NewManager(client kvx.Lock) *Manager {
	return &Manager{
		client: client,
		locks:  collectionmapping.NewConcurrentMap[string, *Lock](),
	}
}

// Acquire acquires a lock with the given key.
func (m *Manager) Acquire(ctx context.Context, key string, opts *Options) (*Lock, error) {
	lock := New(m.client, key, opts)
	if err := lock.Acquire(ctx); err != nil {
		return nil, err
	}
	m.locks.Set(key, lock)
	return lock, nil
}

// TryAcquire tries to acquire a lock with timeout.
func (m *Manager) TryAcquire(ctx context.Context, key string, timeout time.Duration, opts *Options) (*Lock, error) {
	lock := New(m.client, key, opts)
	if err := lock.TryAcquire(ctx, timeout); err != nil {
		return nil, err
	}
	m.locks.Set(key, lock)
	return lock, nil
}

// Release releases a lock by key.
func (m *Manager) Release(ctx context.Context, key string) error {
	if lock, ok := m.locks.LoadAndDelete(key); ok {
		return lock.Release(ctx)
	}
	return ErrLockNotHeld
}

// ReleaseAll releases all managed locks.
func (m *Manager) ReleaseAll(ctx context.Context) error {
	errs := make([]error, 0)
	for _, key := range m.locks.Keys() {
		lock, ok := m.locks.LoadAndDelete(key)
		if !ok {
			continue
		}
		if err := lock.Release(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// IsHeld checks if a lock is held.
func (m *Manager) IsHeld(ctx context.Context, key string) (bool, error) {
	if lock, ok := m.locks.Get(key); ok {
		return lock.IsHeld(ctx)
	}
	return false, nil
}

// WithLock executes a function while holding a lock.
func WithLock(ctx context.Context, client kvx.Lock, key string, opts *Options, fn func() error) (err error) {
	lock := New(client, key, opts)
	if acquireErr := lock.Acquire(ctx); acquireErr != nil {
		return acquireErr
	}
	defer func() {
		err = errors.Join(err, releaseLockOnExit(ctx, lock))
	}()

	return fn()
}

// WithTryLock executes a function while holding a lock, with a timeout for acquisition.
func WithTryLock(ctx context.Context, client kvx.Lock, key string, timeout time.Duration, opts *Options, fn func() error) (err error) {
	lock := New(client, key, opts)
	if acquireErr := lock.TryAcquire(ctx, timeout); acquireErr != nil {
		return acquireErr
	}
	defer func() {
		err = errors.Join(err, releaseLockOnExit(ctx, lock))
	}()

	return fn()
}

func releaseLockOnExit(ctx context.Context, lock *Lock) error {
	if err := lock.Release(ctx); err != nil {
		return fmt.Errorf("release lock %q: %w", lock.key, err)
	}
	return nil
}
