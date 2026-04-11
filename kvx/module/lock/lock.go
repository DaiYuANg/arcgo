// Package lock provides distributed lock functionality.
//
//revive:disable:file-length-limit Lock module operations are kept together as one cohesive API surface.
package lock

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strconv"
	"sync"
	"time"

	collectionmapping "github.com/DaiYuANg/arcgo/collectionx/mapping"
	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/samber/lo"
	"github.com/samber/oops"
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
		return oops.In("kvx/module/lock").
			With("op", "acquire", "key", l.key, "ttl", l.ttl, "auto_extend", l.autoExtend).
			Wrapf(err, "acquire lock")
	}
	if !acquired {
		return oops.In("kvx/module/lock").
			With("op", "acquire", "key", l.key, "ttl", l.ttl, "auto_extend", l.autoExtend).
			Wrapf(ErrLockNotAcquired, "acquire lock")
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
			return oops.In("kvx/module/lock").
				With("op", "try_acquire", "key", l.key, "timeout", timeout, "ttl", l.ttl).
				Wrapf(ErrLockNotAcquired, "acquire lock before deadline")
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
			return oops.In("kvx/module/lock").
				With("op", "try_acquire", "key", l.key, "timeout", timeout, "ttl", l.ttl).
				Wrapf(ctx.Err(), "lock acquisition canceled")
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
		return oops.In("kvx/module/lock").
			With("op", "release", "key", l.key).
			Wrapf(err, "release lock")
	}
	if !released {
		return oops.In("kvx/module/lock").
			With("op", "release", "key", l.key).
			Wrapf(ErrLockNotHeld, "release lock")
	}
	return nil
}

// Extend extends the lock TTL.
func (l *Lock) Extend(ctx context.Context, ttl time.Duration) error {
	extended, err := l.client.Extend(ctx, l.key, l.identifier, ttl)
	if err != nil {
		return oops.In("kvx/module/lock").
			With("op", "extend", "key", l.key, "ttl", ttl).
			Wrapf(err, "extend lock")
	}
	if !extended {
		return oops.In("kvx/module/lock").
			With("op", "extend", "key", l.key, "ttl", ttl).
			Wrapf(ErrLockNotHeld, "extend lock")
	}
	return nil
}

// IsHeld checks if the lock is still held.
func (l *Lock) IsHeld(ctx context.Context) (bool, error) {
	// Try to extend with 0 TTL - this will only succeed if we hold the lock
	held, err := l.client.Extend(ctx, l.key, l.identifier, l.ttl)
	if err != nil {
		return false, oops.In("kvx/module/lock").
			With("op", "is_held", "key", l.key, "ttl", l.ttl).
			Wrapf(err, "check lock state")
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
	return oops.In("kvx/module/lock").
		With("op", "manager_release", "key", key).
		Wrapf(ErrLockNotHeld, "release managed lock")
}

// ReleaseAll releases all managed locks.
func (m *Manager) ReleaseAll(ctx context.Context) error {
	errs := lo.FilterMap(m.locks.Keys(), func(key string, _ int) (error, bool) {
		lock, ok := m.locks.LoadAndDelete(key)
		if !ok {
			return nil, false
		}
		err := lock.Release(ctx)
		return err, err != nil
	})
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
		return oops.In("kvx/module/lock").
			With("op", "release_on_exit", "key", lock.key).
			Wrapf(err, "release lock on exit")
	}
	return nil
}
