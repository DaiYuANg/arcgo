package configx

import "github.com/samber/lo"

func buildOptions(opts ...Option) *Options {
	options := NewOptions()
	lo.ForEach(opts, func(opt Option, _ int) {
		if opt != nil {
			opt(options)
		}
	})
	return options
}
