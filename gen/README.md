# gen

当前目录提供 gorm/gen 的代码生成封装。

```go
package main

import (
	kitgen "github.com/liujitcn/gorm-kit/gen"
)

func main() {
	g := kitgen.New(
		kitgen.WithDriver("mysql"),
		kitgen.WithSource("root:112233@tcp(127.0.0.1:3306)/shop?charset=utf8&parseTime=True&loc=Local&timeout=1000ms"),
		kitgen.WithOutPath("query"),
		kitgen.WithModelPkgPath("models"),
	)
	g.Execute()
}
```

说明：
- `WithDriver` 未设置时，默认 `mysql`。
- `WithSource` 未设置时，优先读取环境变量 `GORM_GEN_DSN`，再使用内置默认值。
- `WithOutPath` 未设置时，默认 `query`。
- `WithModelPkgPath` 未设置时，默认 `models`。
