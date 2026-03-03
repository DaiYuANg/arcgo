package configx

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
	envProvider "github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/v2"
)

// loadDotenv 加载 .env 文件
// ignoreErr: 是否忽略文件不存在的错误
func loadDotenv(k *koanf.Koanf, files []string, ignoreErr bool) error {
	for _, f := range files {
		// 检查文件是否存在
		if _, err := os.Stat(f); os.IsNotExist(err) {
			if !ignoreErr {
				return err
			}
			// 忽略不存在的文件
			continue
		}
		if err := godotenv.Load(f); err != nil && !ignoreErr {
			return err
		}
		// 忽略加载错误
	}
	return nil
}

// loadEnv 加载环境变量到 koanf
// prefix: 环境变量前缀，如 "APP_"
// 支持将环境变量映射到 koanf 的 key (使用 . 作为分隔符)
// 例如：APP_DATABASE_HOST=db.example.com -> database.host
func loadEnv(k *koanf.Koanf, prefix string) error {
	normalizedPrefix := normalizeEnvPrefix(prefix)

	p := envProvider.Provider(".", envProvider.Opt{
		Prefix: normalizedPrefix,
		TransformFunc: func(k, v string) (string, any) {
			keyWithoutPrefix := strings.TrimPrefix(k, normalizedPrefix)
			keyWithoutPrefix = strings.TrimPrefix(keyWithoutPrefix, "_")

			// 转换为小写并将 _ 替换为 .
			key := strings.ReplaceAll(strings.ToLower(keyWithoutPrefix), "_", ".")
			return key, v
		},
		EnvironFunc: os.Environ,
	})

	return k.Load(p, nil)
}

func normalizeEnvPrefix(prefix string) string {
	clean := strings.TrimSpace(prefix)
	if clean == "" {
		return ""
	}
	if strings.HasSuffix(clean, "_") {
		return clean
	}
	return clean + "_"
}
