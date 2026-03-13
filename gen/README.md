# gen

当前目录提供 gorm/gen 的代码生成封装。

```go
package main

import (
	"log"

	kitgen "github.com/liujitcn/gorm-kit/gen"
)

func main() {
	g := kitgen.NewGen(
		kitgen.WithDriver("mysql"),
		kitgen.WithSource("root:112233@tcp(127.0.0.1:3306)/shop?charset=utf8&parseTime=True&loc=Local&timeout=1000ms"),
		kitgen.WithOutputPath("query"),
		kitgen.WithModelPackagePath("models"),
		kitgen.WithAcronym("erp", "ERP"),
		kitgen.WithAcronyms(map[string]string{
			"crm": "CRM",
		}),
	)
	if err := g.Execute(); err != nil {
		log.Fatal(err)
	}
}
```

说明：
- `WithDriver` 未设置时，默认 `mysql`。
- `WithSource` 未设置时，优先读取环境变量 `GORM_GEN_DSN`，再使用内置默认值。
- `WithOutputPath` 未设置时，默认 `query`。
- `WithModelPackagePath` 未设置时，默认 `models`。
- `WithAcronym` / `WithAcronyms` 可追加模型命名缩写映射（外部配置会覆盖同名默认值）。
- `WithInitialism` / `WithInitialisms` 为兼容旧版本的别名方法，建议逐步迁移到 `WithAcronym` / `WithAcronyms`。
- `WithOutPath` / `WithModelPkgPath` / `NewGen` 为兼容旧版本的别名方法，建议逐步迁移到 `WithOutputPath` / `WithModelPackagePath` / `New`。
