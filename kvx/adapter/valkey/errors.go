package valkey

import (
	"errors"
	"fmt"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/valkey-io/valkey-go"
)

func wrapValkeyError(op string, err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("valkey %s: %w", op, err)
}

func wrapValkeyNilError(op string, err error) error {
	if err == nil {
		return nil
	}

	if valkey.IsValkeyNil(err) {
		return kvx.ErrNil
	}

	return wrapValkeyError(op, err)
}

func errValkeyUnsupported(feature string) error {
	return fmt.Errorf("%w: valkey %s is not implemented", kvx.ErrUnsupportedOption, feature)
}

func bytesFromResult(op string, resp valkey.ValkeyResult) ([]byte, error) {
	if err := wrapValkeyNilError(op, resp.Error()); err != nil {
		return nil, err
	}

	value, err := resp.AsBytes()
	if err == nil {
		return value, nil
	}

	text, stringErr := resp.ToString()
	if stringErr == nil {
		return []byte(text), nil
	}

	return nil, errors.Join(wrapValkeyError(op, err), wrapValkeyError(op, stringErr))
}

func boolFromResult(op string, resp valkey.ValkeyResult) (bool, error) {
	if err := wrapValkeyError(op, resp.Error()); err != nil {
		return false, err
	}

	value, err := resp.AsBool()
	if err != nil {
		return false, wrapValkeyError(op, err)
	}

	return value, nil
}

func int64FromResult(op string, resp valkey.ValkeyResult) (int64, error) {
	if err := wrapValkeyError(op, resp.Error()); err != nil {
		return 0, err
	}

	value, err := resp.AsInt64()
	if err != nil {
		return 0, wrapValkeyError(op, err)
	}

	return value, nil
}

func stringFromResult(op string, resp valkey.ValkeyResult) (string, error) {
	if err := wrapValkeyError(op, resp.Error()); err != nil {
		return "", err
	}

	value, err := resp.ToString()
	if err != nil {
		return "", wrapValkeyError(op, err)
	}

	return value, nil
}

func stringSliceFromResult(op string, resp valkey.ValkeyResult) (collectionx.List[string], error) {
	if err := wrapValkeyError(op, resp.Error()); err != nil {
		return nil, err
	}

	value, err := resp.AsStrSlice()
	if err != nil {
		return nil, wrapValkeyError(op, err)
	}

	return collectionx.NewListWithCapacity(len(value), value...), nil
}

func stringMapFromResult(op string, resp valkey.ValkeyResult) (map[string]string, error) {
	if err := wrapValkeyError(op, resp.Error()); err != nil {
		return nil, err
	}

	value, err := resp.AsStrMap()
	if err != nil {
		return nil, wrapValkeyError(op, err)
	}

	return value, nil
}

func ftSearchDocsFromResult(op string, resp valkey.ValkeyResult) ([]valkey.FtSearchDoc, error) {
	if err := wrapValkeyError(op, resp.Error()); err != nil {
		return nil, err
	}

	_, docs, err := resp.AsFtSearch()
	if err != nil {
		return nil, wrapValkeyError(op, err)
	}

	return docs, nil
}

func ftAggregateDocsFromResult(op string, resp valkey.ValkeyResult) ([]map[string]string, error) {
	if err := wrapValkeyError(op, resp.Error()); err != nil {
		return nil, err
	}

	_, docs, err := resp.AsFtAggregate()
	if err != nil {
		return nil, wrapValkeyError(op, err)
	}

	return docs, nil
}

func xReadEntriesFromResult(op string, resp valkey.ValkeyResult) (map[string][]valkey.XRangeEntry, error) {
	if err := resp.Error(); err != nil {
		if valkey.IsValkeyNil(err) {
			return map[string][]valkey.XRangeEntry{}, nil
		}

		return nil, wrapValkeyError(op, err)
	}

	entries, err := resp.AsXRead()
	if err != nil {
		return nil, wrapValkeyError(op, err)
	}

	return entries, nil
}

func xRangeEntriesFromResult(op string, resp valkey.ValkeyResult) ([]valkey.XRangeEntry, error) {
	if err := wrapValkeyError(op, resp.Error()); err != nil {
		return nil, err
	}

	entries, err := resp.AsXRange()
	if err != nil {
		return nil, wrapValkeyError(op, err)
	}

	return entries, nil
}
