# gorm-kit

`gorm-kit` 是一个基于 GORM 的通用工具仓库，当前主要包含两个模块：

- `repository`：通用仓储能力与函数式查询选项
- `gen`：基于 `gorm/gen` 的代码生成入口

## 目录说明

- `repository/`：仓储接口、分页、批量写入策略、函数式查询选项
- `gen/`：生成 `models`、`query`、`data` 的代码生成器

## 工具链与测试

格式化依赖 `goimports`，执行 `make fmt` 前需先确保本机已安装 `goimports`。

本仓库包含根目录与 `gen` 两个 Go module，提交前需分别执行测试：

```bash
go test ./...
cd gen && go test ./...
```

## repository

`repository` 层直接复用 `gorm/gen` 的强类型字段构建查询。`NewBaseRepository` 需要显式传入：

- `queryDAO`
- 主键字段访问器
- 实体主键读取函数

示例：

```go
userRepository := repository.NewBaseRepository(
    func(ctx context.Context) gen.Dao { return query.Use(db).User.WithContext(ctx) },
    func(ctx context.Context) field.Int64 { return query.Use(db).User.WithContext(ctx).ID },
    func(entity *model.User) int64 { return entity.ID },
)
```

常用 `QueryOption` 包括：

- `Where`、`Not`、`Or`
- `Select`、`Distinct`、`Omit`
- `Join`、`LeftJoin`、`RightJoin`
- `Group`、`Having`
- `Order`、`Limit`、`Offset`
- `Attrs`、`Assign`
- `Joins`、`Preload`
- `Clauses`
- `Scope`、`Scopes`
- `Unscoped`

分页辅助方法：

- `PageDefault`：统一补齐分页默认值，默认 `page=1`、`size=10`
- `PageOffsetLimit`：基于补齐后的分页参数计算 `offset` 与 `limit`

## gen

`gen` 当前支持：

- 默认读取服务 `./configs/data.yaml`，支持 `data.database` 和 `data.databases`
- 一次命令生成多个命名数据源，支持 `-database` 选择单个数据源
- `table` 支持逗号分隔的多表，例如 `user,user2`
- 输出按数据源隔离：旧单库使用 `gen/{models,query,data}`，命名数据源使用 `gen/<name>/{models,query,data}`
- 每套 `data` 生成 `Models()`、`NewClient()`、`NewData()` 与 Repository ProviderSet
- 默认数据源的 `NewClient` 接收单个 `*configv1.Data_Database`；命名数据源的 `NewClient` 接收 `databases map[string]*configv1.Data_Database` 并按 key 取出当前配置
- `source`、`driver`、详细输出路径参数继续兼容单库调用
- 生成模板拆分在 `gen/internal/generator/templates/*.tmpl`，并通过 `go:embed` 嵌入生成器
- 生成模型、Repository 与字段名称时保留统一缩写表全大写，包含 GORM 内置缩写以及 `SKU`、`SPU`、`LLM` 等业务扩展缩写
- 全量生成清空目标范围：未指定 `-database` 时清空整个 `base_path`，指定 `-database` 时只清空对应数据源目录；单表生成保留其他表产物

示例：

```bash
cd gen
go run ./cmd/gorm-gen -h
go run ./cmd/gorm-gen
go run ./cmd/gorm-gen -database=main
go run ./cmd/gorm-gen -config=./configs/data.yaml -database=main -table=user,user2
go run ./cmd/gorm-gen -source='root:123456@tcp(127.0.0.1:3306)/shop?charset=utf8&parseTime=True&loc=Local&timeout=1000ms'
```

当前支持的参数：

- `config`
- `database`
- `driver`
- `source`
- `table`
- `base_path`
- `out_path`（仅单库兼容模式）
- `model_pkg_path`（仅单库兼容模式）
- `data_path`（仅单库兼容模式）

更完整说明见：

- [gen/README.md](./gen/README.md)
