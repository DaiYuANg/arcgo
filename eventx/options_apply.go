package eventx

import "github.com/samber/lo"

func applyOptions[T any, O ~func(*T)](target *T, opts ...O) {
	lo.ForEach(opts, func(opt O, _ int) {
		if opt != nil {
			opt(target)
		}
	})
}

func buildSubscribeOptions(opts ...SubscribeOption) subscribeOptions {
	cfg := defaultSubscribeOptions()
	applyOptions(&cfg, opts...)
	return cfg
}
