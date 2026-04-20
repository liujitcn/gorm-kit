# gen

当前目录提供基于 `gorm/gen` 的代码生成入口，支持：

- 通过与 `options` 一致的命令行参数覆盖默认配置
- 生成 `models`、`query` 与 `data`
- 额外生成 `models/table_comment.gen.go`，为带表注释的模型补充 `TableComment() string`
- 每次生成前自动清空 `models`、`query`、`data` 目标目录，避免旧表旧文件残留

## 使用方式

查看帮助：

```bash
go run . -h
```

使用命令行参数启动生成：

```bash
go run . -source='root:123456@tcp(127.0.0.1:3306)/shop?charset=utf8&parseTime=True&loc=Local&timeout=1000ms'
go run . -source='root:123456@tcp(127.0.0.1:3306)/shop?charset=utf8&parseTime=True&loc=Local&timeout=1000ms' -base_path=test
go run . -out_path=query1/tet -model_pkg_path=models1/tst -data_path=./data1
go run . -source='root:123456@tcp(127.0.0.1:3306)/shop?charset=utf8&parseTime=True&loc=Local&timeout=1000ms' -base_path=.server/pkg -out_path=query1/tet -model_pkg_path=models1/tst -data_path=./data1
```

## 启动参数

当前支持以下命令行参数：

- `driver`：数据库驱动，默认 `mysql`
- `source`：数据库连接串，必填
- `base_path`：统一基础路径，例如传 `test` 后会生成到 `test/query`、`test/models`、`test/data`
- `out_path`：`query` 输出目录，默认 `query`
- `model_pkg_path`：`model` 包路径，默认 `models`
- `data_path`：`data` 输出目录，默认 `data`

## 生成规则

- `model_pkg_path`、`out_path`、`data_path` 会同时影响对应目录的生成结果
- `base_path` 会统一为最终的 `model_pkg_path`、`out_path`、`data_path` 增加前缀
- 例如 `-base_path=.server/pkg -data_path=./data1` 最终会生成到 `.server/pkg/data1`
- `models` 目录会额外生成 `table_comment.gen.go`，用于在运行时暴露表注释，配合自动迁移恢复表注释
- `data` 中引用的 `models`、`query` 会跟随实际导入路径与目标包名变化
- `data` 包名取 `data_path` 最后一层目录名
- `data` 中每个 Repo 默认生成导出结构体，并内嵌通用 `BaseRepo` 与 `*Data`
- `data` 中 Repo 主键优先取模型声明顺序上的第一个 `int64` 主键字段；联合主键表同样按该规则生成，若不存在 `int64` 主键则回退到第一个主键字段
- 生成前会先删除整个 `model_pkg_path`、`out_path`、`data_path`

## 默认值

- `driver` 默认 `mysql`
- `source` 无默认值，必须显式传入
- `base_path` 默认空
- `out_path` 默认 `query`
- `model_pkg_path` 默认 `models`
- `data_path` 默认 `data`
