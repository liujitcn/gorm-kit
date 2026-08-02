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
- 支持 `-database` 选择命名数据源；指定名称始终用于生成目录，不存在时连接配置回退到默认数据源
- `table` 支持逗号分隔的多表，例如 `user,user2`
- 默认数据源输出到 `base_path`，命名数据源输出到 `base_path/<数据源>`
- 全量生成只清理目标数据源目录下的 `query`、`data`、`repo`，保留 `models` 和其他目录
- 每套 `data` 生成 `Models()`、`NewClient()`、`NewData()`、不含客户端的 `RepositoryProviderSet` 与完整的 `ProviderSet`
- 默认数据源的 `NewClient` 接收单个 `*configv1.Data_Database`；命名数据源的 `NewClient` 接收 `databases map[string]*configv1.Data_Database`，优先按当前目录对应的数据源名称取配置，不存在时回退到 `default`
- 命令行只负责选择配置、数据源、表和生成根目录，连接与驱动统一从服务配置读取
- 生成模板拆分在 `gen/internal/generator/templates/*.tmpl`，并通过 `go:embed` 嵌入生成器
- 生成模型、Repository 与字段名称时保留统一缩写表全大写，包含 GORM 内置缩写以及 `SKU`、`SPU`、`LLM` 等业务扩展缩写
- 全量生成清空对应数据源目录下的 `query`、`data`、`repo`；单表生成保留其他表产物

示例：

```bash
cd gen
go run ./cmd/gorm-gen -h
go run ./cmd/gorm-gen
go run ./cmd/gorm-gen -database=main
go run ./cmd/gorm-gen -config=./configs/data.yaml -database=main -table=user,user2
```

当前支持的参数：

- `config`
- `database`
- `table`
- `base_path`

更完整说明见：

- [gen/README.md](./gen/README.md)
