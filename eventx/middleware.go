package eventx

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"
)

// HandlerFunc is the runtime event handler signature after type adaptation.
type HandlerFunc func(context.Context, Event) error

// Middleware wraps HandlerFunc.
type Middleware func(HandlerFunc) HandlerFunc

func chain(handler HandlerFunc, mws []Middleware) HandlerFunc {
	return lo.ReduceRight(mws, func(out HandlerFunc, mw Middleware, _ int) HandlerFunc {
		if mw == nil {
			return out
		}
		return mw(out)
	}, handler)
}

// RecoverMiddleware turns panic into normal error so dispatch can continue.
func RecoverMiddleware() Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, event Event) (err error) {
			defer func() {
				if recovered := recover(); recovered != nil {
					err = fmt.Errorf("eventx: recovered panic: %v", recovered)
				}
			}()
			return next(ctx, event)
		}
	}
}

// ObserveMiddleware reports per-dispatch execution result.
func ObserveMiddleware(observer func(ctx context.Context, event Event, duration time.Duration, err error)) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, event Event) error {
			start := time.Now()
			err := next(ctx, event)
			if observer != nil {
				observer(ctx, event, time.Since(start), err)
			}
			return err
		}
	}
}
