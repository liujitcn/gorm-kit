package main

import (
	"flag"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	_ "github.com/liujitcn/kratos-kit/database/gorm/driver/mysql"
)

type setFlags []string

// String 返回命令行覆盖项的字符串表示。
func (s *setFlags) String() string {
	return strings.Join(*s, ",")
}

// Set 收集重复传入的覆盖项。
func (s *setFlags) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func main() {
	configFile := flag.String("config", defaultConfigFile, "配置文件路径")
	var sets setFlags
	flag.Var(&sets, "set", "覆盖配置项，例如 -set model_pkg_path=models1 或 -set acronyms.api=API")
	flag.Parse()

	opts, err := loadOptionsFromConfigFileAndOverrides(*configFile, parseSetOverrides(sets))
	if err != nil {
		log.Fatal(err)
	}

	g := &Gen{opts: opts}
	tables, err := g.GenerateAllTable()
	if err != nil {
		log.Fatal(err)
	}
	if err = g.Execute(); err != nil {
		log.Fatal(err)
	}
	if err = generateDataFiles(opts, tables); err != nil {
		log.Fatal(err)
	}
}

// parseSetOverrides 解析命令行传入的 key=value 覆盖项。
func parseSetOverrides(values []string) map[string]string {
	overrides := make(map[string]string, len(values))
	for _, item := range values {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			continue
		}
		overrides[key] = value
	}
	return overrides
}
