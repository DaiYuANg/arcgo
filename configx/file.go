package configx

import (
	"path/filepath"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/toml/v2"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

func loadFiles(k *koanf.Koanf, files []string) error {

	for _, f := range files {
		ext := filepath.Ext(f)

		var parser koanf.Parser

		switch ext {
		case ".yaml", ".yml":
			parser = yaml.Parser()
		case ".json":
			parser = json.Parser()
		case ".toml":
			parser = toml.Parser()
		default:
			continue
		}

		if err := k.Load(file.Provider(f), parser); err != nil {
			return err
		}
	}
	return nil
}
