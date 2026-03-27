package mapping

import (
	"reflect"
	"time"
)

var timeType = reflect.TypeFor[time.Time]()

func isSignedIntKind(kind reflect.Kind) bool {
	return kind >= reflect.Int && kind <= reflect.Int64
}

func isUnsignedIntKind(kind reflect.Kind) bool {
	return kind >= reflect.Uint && kind <= reflect.Uintptr
}

func isFloatKind(kind reflect.Kind) bool {
	return kind >= reflect.Float32 && kind <= reflect.Float64
}
