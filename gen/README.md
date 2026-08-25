# gen

`gen` 是可复用的 GORM 代码生成模块，命令入口位于 `cmd/gorm-gen`。配置解析、单数据源生成和模板输出由库代码负责，CLI 只负责参数转换。

## 运行

本模块要求 Go 1.27.0 或更高版本，并继续使用 `kratos-kit` 的数据库驱动注册包。

查看帮助：

```bash
go run ./cmd/gorm-gen -h
```

默认读取当前工作目录下的 `./configs/data.yaml`：

```bash
go run ./cmd/gorm-gen
```

仓库内的 [`configs/data.yaml`](configs/data.yaml) 是默认配置模板，实际项目中可直接替换为服务自己的数据库配置。

配置文件直接复用服务的 `data.yaml`，只读取 `data.database` 和 `data.databases`：

```yaml
data:
  databases:
    main:
      driver: mysql
      source: root:password@tcp(127.0.0.1:3306)/shop
    audit-prod:
      driver: postgres
      source: postgres://user:password@127.0.0.1:5432/audit
```

默认数据源直接生成到 `base_path`，命名数据源生成到 `base_path/<数据源>`：

```text
gen/{models,query,data}
gen/main/{models,query,data}
```

旧的 `data.database` 按名称 `default` 兼容并直接输出到 `base_path`。两种字段同时存在时仍可通过 `-database` 选择 `data.databases` 中的数据源；不传 `-database` 时使用默认数据源。传入 `-database` 时始终使用传入名称生成目录；名称不存在时连接配置回退到默认数据源。`databases.default` 与旧字段冲突时报错。

## 参数

- `config`：服务配置文件，默认 `./configs/data.yaml`
- `database`：只生成指定的数据源
- `table`：指定表，支持 `user,user2`
- `base_path`：生成根目录，默认 `gen`

未传 `database` 时生成默认数据源；传入 `database` 时始终按指定名称生成目录，连接配置不存在时回退到默认数据源。传入 `table` 时也遵循同样的选择规则。

数据库驱动和连接串统一在服务配置中声明；Doris 数据源使用 `driver: doris`。默认数据源的模型、查询和 data 目录生成到 `base_path`，命名数据源生成到 `base_path/<数据源>`，不再单独配置详细输出路径。

## 生成规则

- 默认生成数据源全部表；指定 `table` 时先校验全部表，任一表不存在则当前数据源生成失败。
- 全量生成只清理目标数据源目录下的 `query`、`data`、`repo` 目录，`models` 和其他目录保持不变；指定表时不清理目录。
- 指定表时保留其他表产物，只更新指定表并重建聚合入口。
- 每套 `data` 包生成 `Models()`、`NewData(databases map[string]*databaseGorm.Client)` 和包含仓储依赖的 `ProviderSet`，迁移模型只绑定当前数据源；`NewData` 在客户端 map 为空或缺少默认客户端时返回错误，命名数据源额外按数据源导出独立的客户端名称常量。
- 数据库客户端由外部统一创建并以 map 传入 `NewData`；默认数据源使用 `databaseGorm.DefaultClientName`，命名数据源使用生成的客户端名称常量选择对应客户端。
- Doris 使用 `information_schema` 读取字段元数据，避免默认采样查询追加 Doris 不兼容的 `LIMIT`。
- 模型、Repository 与字段名称沿用 `go-utils/stringcase` 的缩写规则；`BIGINT deleted_at` 保留 `soft_delete.DeletedAt` 生成策略。

## 结构

```text
gen/
├── api.go                    # 对外公开的配置生成入口
├── cmd/gorm-gen/             # CLI 适配层
└── internal/
    ├── config/               # data.yaml 读取、选择与批量编排
    └── generator/            # 单数据源生成核心
        └── templates/        # models/query/data 模板
```
