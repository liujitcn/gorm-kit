# gorm-kit/gen

`gen` 是独立的 Go module，负责把数据库结构转换为可编译的 GORM 代码。命令行入口位于 `cmd/gorm-gen`，公共调用入口是 `gen.GenerateConfig`；配置解析、数据源选择、元数据读取和模板输出都由 module 内部完成。

生成器输出三层代码：

- `models`：模型结构、GORM/JSON 标签和表注释方法。
- `query`：`gorm/gen` 强类型查询入口、读写连接切换和事务查询对象。
- `data`：数据访问对象、事务上下文、迁移模型集合、Wire `ProviderSet` 和按表生成的基础仓储。

生成的 `data` 包依赖根 module 的 `repository` 包。业务项目应把生成目录放在自己的 Go module 中，并根据生成代码补齐数据库客户端和相关运行时依赖。

## 环境要求

- Go 1.27.0 或更高版本。
- 目标数据库可访问，连接串放在配置文件中。
- 生成目录所在位置或其父级目录必须存在 `go.mod`，生成器会从 `data` 输出目录向上查找 module 路径，用于计算模型和查询包的 import 路径。

## 命令行

在 `gen` 目录执行：

```bash
go run ./cmd/gorm-gen -h
go run ./cmd/gorm-gen
go run ./cmd/gorm-gen -config=./configs/data.yaml -database=main
go run ./cmd/gorm-gen -config=./configs/data.yaml -database=main -table=user,user_profile
```

参数如下：

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `-config` | `./configs/data.yaml` | 服务数据配置文件 |
| `-database` | 空 | 选择一个命名数据源；空值选择默认数据源 |
| `-table` | 空 | 指定表名，多个表用逗号分隔；空值生成当前数据源全部表 |
| `-base_path` | `gen` | 生成根目录 |

CLI 只负责把参数转换为 `gen.ConfigOptions`，实际生成逻辑也可以直接通过 Go API 调用：

```go
err := gen.GenerateConfig(gen.ConfigOptions{
	ConfigPath:   "./configs/data.yaml",
	DatabaseName: "main",
	Table:        "user,user_profile",
	BasePath:     "gen",
})
```

## 配置格式

配置文件只读取 `data.database` 和 `data.databases`：

```yaml
data:
  database:
    driver: sqlite
    source: "file:./configs/example.db"
  databases:
    main:
      driver: mysql
      source: "root:password@tcp(127.0.0.1:3306)/shop"
    audit:
      driver: postgres
      source: "postgres://user:password@127.0.0.1:5432/audit"
```

数据源配置字段：

- `driver`：数据库驱动名称；省略时按生成器默认值 `mysql` 处理。
- `source`：传给 GORM 方言的连接串。

### 数据源选择规则

1. `data.databases` 中的命名数据源按键名选择。
2. 旧式 `data.database` 会被视为名称为 `default` 的默认数据源。
3. `data.database` 与 `data.databases.default` 同时存在会报冲突错误。
4. 不传 `-database` 时选择 `default`；没有默认数据源时返回错误，即使配置中存在其他命名数据源也不会自动猜测。
5. 传入已存在的 `-database` 时使用该数据源的连接配置。
6. 传入不存在的 `-database` 时，如果存在 `default`，则借用默认连接生成，但输出目录和生成代码中的客户端名称仍使用传入的名称；没有默认数据源时返回错误。

默认数据源直接输出到 `base_path`，命名数据源输出到 `base_path/<数据源目录>`。目录名只保留数据源名称中的字母和数字并转换为小写；无法生成有效目录名时返回错误。

## 支持的驱动

当前通过驱动注册表支持：

| `driver` 值 | 生成方式 |
| --- | --- |
| `mysql` | 使用 MySQL 方言读取表结构 |
| `doris` | 使用 MySQL 连接方言，并通过 `information_schema` 读取 Doris 元数据 |
| `postgres` | 读取 PostgreSQL 表、索引、注释和二进制长度约束 |
| `sqlite` | 使用 SQLite 方言读取表结构 |
| `sqlserver` | 使用 SQL Server 方言读取表结构 |
| `oracle` | 使用 Oracle 方言读取表结构 |
| `bigquery` | 使用 BigQuery 方言读取表结构 |

使用 Doris 时，字段元数据来自 `information_schema.columns`，表注释来自 `information_schema.tables`，不会依赖 gorm/gen 对字段采样追加的查询。Doris 视图没有主键时，生成器会为其选择第一个字段作为稳定的仓储主键字段，但仍会把视图从 `Models()` 迁移列表中排除。

PostgreSQL 生成会额外读取表注释；`int2` 统一映射为 `int32`，`bytea` 映射为 `[]byte`。由 `gormsize_` 检查约束记录的二进制长度会恢复为字段 `size` 标签，便于不同数据库之间保持模型标签一致。

## 生成范围与清理规则

### 全量生成

未传 `-table` 时：

- 读取当前数据源的全部表。
- 在确认数据库结构读取成功后，清理目标数据源目录下的 `query`、`models`、`data`。
- 重新生成三个目录，数据库中已删除的表不会残留旧的仓储或迁移文件。
- 清理范围之外的目录保持不变。

### 指定表生成

传入 `-table=user,user_profile` 时：

- 逗号分隔的名称会去除首尾空白，空名称会被忽略；最终没有有效表名时返回错误。
- 所有指定表都会先读取成功，再开始写文件；任一表不存在或读取失败都会终止本次生成。
- 不清理 `query`、`models`、`data`，保留其他表产物。
- 只覆盖指定表的模型、查询和仓储文件，并重建包含目录中全部表的聚合入口。

## 生成产物

```text
<base_path>/
├── models/
│   ├── <table>.gen.go
│   ├── table_comment.gen.go
│   └── table_name.gen.go
├── query/
│   ├── <table>.gen.go
│   └── gen.go
└── data/
    ├── <table>.go
    ├── data.go
    └── init.go
```

### `models`

- 表名和字段名按下划线转 Go 驼峰命名。
- 常见缩写保持大写，包含生成器内置缩写和 `SKU`、`SPU`、`LLM` 等扩展缩写。
- 生成索引、类型和字段标签，不生成单元测试文件。
- `deleted_at` 为 `BIGINT` 或等价 64 位整数时映射为 `soft_delete.DeletedAt`，并设置毫秒级软删除标签；其他类型保持原始类型。
- 表注释生成到 `TableComment()` 方法，供迁移或业务代码读取。

### `query`

每张表生成强类型字段和 DAO，聚合入口提供：

- `Use(*gorm.DB)`：创建查询根对象。
- `WithContext`：为全部表设置上下文。
- `ReadDB`、`WriteDB`、`ReplaceDB`：切换读库、写库或指定连接。
- `Transaction`、`Begin`、`Commit`、`Rollback`、`SavePoint`、`RollbackTo`：事务和保存点操作。

### `data`

- `Models()`：返回当前数据源的全部迁移模型，视图不会加入该列表。
- `NewData(map[string]*databaseGorm.Client)`：从外部客户端映射构造数据访问对象。
- `QueryProvider`：按上下文取得查询入口。
- `Transaction`：把事务态查询对象写入上下文，使仓储自动复用当前事务。
- `ProviderSet`：提供 `NewData`、`NewTransaction` 和每张表的基础仓储构造器。
- 每张表生成一个实现 `repository.BaseRepository[T]` 的仓储类型，并暴露 `Query(ctx)` 获取当前查询入口。

`NewData` 要求客户端映射非空且包含默认客户端。命名数据源会额外生成客户端名称常量；若命名客户端未提供但默认客户端存在，查询会回退到默认客户端。

联合主键的基础仓储默认选用声明顺序中第一个 `int64` 主键字段；找不到时回退到第一个主键字段。这个选择只用于通用的按 ID 方法，复杂联合主键条件应通过 `QueryOption` 显式构造。

## 直接使用生成器 API

`gen.GenerateConfig` 适合读取服务配置并生成一个数据源：

```go
package main

import "github.com/liujitcn/gorm-kit/gen"

func main() {
	err := gen.GenerateConfig(gen.ConfigOptions{
		ConfigPath: "./configs/data.yaml",
		BasePath:   "./gen",
	})
	if err != nil {
		panic(err)
	}
}
```

生成器会在数据库结构读取完成后才清理全量输出目录；连接失败或表元数据读取失败时，已有生成结果会保留。

## 相关路径

- `api.go`：公共配置类型和生成入口。
- `cmd/gorm-gen/main.go`：命令行参数转换和错误退出。
- `internal/config/`：YAML 解析、数据源选择和多数据源编排。
- `internal/generator/`：数据库连接、表元数据读取、清理策略和模板生成。
- `internal/generator/templates/`：内嵌的模型、查询和 data 模板。
