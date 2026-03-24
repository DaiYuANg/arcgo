package configx

import "errors"

var (
	// ErrLoad indicates a failure in the high-level load pipeline.
	ErrLoad = errors.New("configx: load")
	// ErrUnmarshal indicates config decode/unmarshal failures.
	ErrUnmarshal = errors.New("configx: unmarshal")
	// ErrValidate indicates validation failures.
	ErrValidate = errors.New("configx: validate")
	// ErrDefaults indicates invalid default value configuration.
	ErrDefaults = errors.New("configx: defaults")
)
