package configx

import (
	"errors"
	"fmt"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/v2"
	"github.com/spf13/pflag"
)

type flagGetter interface {
	Get() any
}

type argEntry struct {
	Name  string
	Path  string
	Value any
}

func loadArgs(k *koanf.Koanf, args []string, fs *pflag.FlagSet, nameFunc func(string) string) error {
	if len(args) == 0 && fs == nil {
		return nil
	}
	if nameFunc == nil {
		nameFunc = defaultArgsName
	}

	rawEntries, err := parseRawArgs(args, nameFunc)
	if err != nil {
		return err
	}
	flagEntries, err := changedFlagEntries(fs, nameFunc)
	if err != nil {
		return err
	}
	if rawEntries.IsEmpty() && flagEntries.IsEmpty() {
		return nil
	}

	values := collectionx.NewMapWithCapacity[string, any](rawEntries.Len() + flagEntries.Len())
	if err := applyArgEntries(values, rawEntries, "args"); err != nil {
		return err
	}
	if err := applyArgEntries(values, flagEntries, "flags"); err != nil {
		return err
	}
	if err := k.Load(confmap.Provider(values.All(), "."), nil); err != nil {
		return fmt.Errorf("configx: load args: %w", errors.Join(ErrArgs, err))
	}
	return nil
}

func parseRawArgs(args []string, nameFunc func(string) string) (collectionx.List[argEntry], error) {
	tokens := collectionx.NewListWithCapacity[string](len(args), args...)
	entries := collectionx.NewListWithCapacity[argEntry](tokens.Len())

	for index := 0; index < tokens.Len(); index++ {
		token, ok := tokens.Get(index)
		if !ok {
			break
		}
		if token == "--" {
			break
		}
		if !strings.HasPrefix(token, "--") || len(token) <= 2 {
			continue
		}

		entry, consumedNext, err := parseRawArgEntry(tokens, index, nameFunc)
		if err != nil {
			return nil, err
		}
		entries.Add(entry)
		if consumedNext {
			index++
		}
	}

	return entries, nil
}

func parseRawArgEntry(tokens collectionx.List[string], index int, nameFunc func(string) string) (argEntry, bool, error) {
	token, ok := tokens.Get(index)
	if !ok {
		return argEntry{}, false, fmt.Errorf("%w: missing raw arg token at index %d", ErrArgs, index)
	}

	raw := strings.TrimSpace(strings.TrimPrefix(token, "--"))
	name := raw
	value := any(true)
	consumedNext := false

	if eqIndex := strings.Index(raw, "="); eqIndex >= 0 {
		name = raw[:eqIndex]
		value = raw[eqIndex+1:]
	} else if strings.HasPrefix(raw, "no-") && len(raw) > len("no-") {
		name = strings.TrimPrefix(raw, "no-")
		value = false
	} else if next, ok := tokens.Get(index + 1); ok && next != "--" && !strings.HasPrefix(next, "--") {
		value = next
		consumedNext = true
	}

	path, err := resolveArgsPath(name, nameFunc, "arg")
	if err != nil {
		return argEntry{}, false, err
	}
	return argEntry{
		Name:  name,
		Path:  path,
		Value: value,
	}, consumedNext, nil
}

func changedFlagEntries(fs *pflag.FlagSet, nameFunc func(string) string) (collectionx.List[argEntry], error) {
	if fs == nil {
		return collectionx.NewList[argEntry](), nil
	}

	entries := collectionx.NewListWithCapacity[argEntry](changedFlagCount(fs))
	var visitErr error
	fs.Visit(func(flag *pflag.Flag) {
		if visitErr != nil {
			return
		}

		path, err := resolveArgsPath(flag.Name, nameFunc, "flag")
		if err != nil {
			visitErr = err
			return
		}
		value, err := flagConfigValue(flag)
		if err != nil {
			visitErr = err
			return
		}
		entries.Add(argEntry{
			Name:  flag.Name,
			Path:  path,
			Value: value,
		})
	})
	if visitErr != nil {
		return nil, visitErr
	}
	return entries, nil
}

func applyArgEntries(values collectionx.Map[string, any], entries collectionx.List[argEntry], sourceLabel string) error {
	if values == nil || entries == nil || entries.IsEmpty() {
		return nil
	}

	namesByPath := collectionx.NewMapWithCapacity[string, string](entries.Len())
	var applyErr error
	entries.Range(func(_ int, entry argEntry) bool {
		if existing, ok := namesByPath.Get(entry.Path); ok && existing != entry.Name {
			applyErr = fmt.Errorf("%w: %s %q and %q resolve to the same config path %q", ErrArgs, sourceLabel, existing, entry.Name, entry.Path)
			return false
		}
		namesByPath.Set(entry.Path, entry.Name)
		values.Set(entry.Path, entry.Value)
		return true
	})
	return applyErr
}

func flagConfigValue(flag *pflag.Flag) (any, error) {
	if flag == nil || flag.Value == nil {
		return nil, fmt.Errorf("%w: nil flag value", ErrArgs)
	}
	if getter, ok := flag.Value.(flagGetter); ok {
		return getter.Get(), nil
	}
	if sliceValue, ok := flag.Value.(pflag.SliceValue); ok {
		return sliceValue.GetSlice(), nil
	}
	return flag.Value.String(), nil
}

func resolveArgsPath(name string, nameFunc func(string) string, kind string) (string, error) {
	path := normalizeArgsPath(nameFunc(strings.TrimSpace(name)))
	if path == "" {
		return "", fmt.Errorf("%w: %s %q resolved to an empty config path", ErrArgs, kind, name)
	}
	return path, nil
}

func normalizeArgsPath(path string) string {
	trimmed := strings.Trim(strings.TrimSpace(path), ".")
	for strings.Contains(trimmed, "..") {
		trimmed = strings.ReplaceAll(trimmed, "..", ".")
	}
	return trimmed
}

func changedFlagCount(fs *pflag.FlagSet) int {
	if fs == nil {
		return 0
	}
	count := 0
	fs.Visit(func(_ *pflag.Flag) {
		count++
	})
	return count
}
