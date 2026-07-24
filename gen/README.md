# gen

`gen` 是可复用的 GORM 代码生成模块，命令入口位于 `cmd/gorm-gen`。配置解析、单数据源生成和模板输出由库代码负责，CLI 只负责参数转换。

## 运行

查看帮助：

```bash
go run ./cmd/gorm-gen -h
```

默认读取当前工作目录下的 `./configs/data.yaml`：

```bash
go run ./cmd/gorm-gen
```

仓库内的 [`configs/data.yaml`](configs/data.yaml) 是默认配置模板，使用 SQLite 的 `configs/example.db` 作为示例路径；实际项目中将该文件替换为服务自己的数据库配置即可。

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

命名数据源默认生成到：

```text
gen/main/{models,query,data}
gen/auditprod/{models,query,data}
```

旧的 `data.database` 生成到 `gen/{models,query,data}`。两种字段同时存在时会合并生成，旧字段名称为 `default`，命名数据源生成到 `gen/<key>/{models,query,data}`；`databases.default` 与旧字段冲突时报错。

## 参数

- `config`：服务配置文件，默认 `./configs/data.yaml`
- `database`：只生成指定的数据源
- `table`：指定表，支持 `user,user2`
- `base_path`：生成根目录，默认 `gen`

未传 `database` 时生成合并后的全部数据源；传入 `database` 时只生成指定数据源。传入 `table` 时必须同时指定 `database`。

单库兼容参数仍可使用：

```bash
go run ./cmd/gorm-gen \
  -source='root:password@tcp(127.0.0.1:3306)/shop?charset=utf8mb4&parseTime=True&loc=Local' \
  -table=user,user2
```

单库模式还支持 `driver`、`out_path`、`model_pkg_path`、`data_path`；这些详细输出路径不适用于配置文件多数据源模式。

## 生成规则

- 默认生成数据源全部表；指定 `table` 时先校验全部表，任一表不存在则当前数据源生成失败。
- 多数据源生成按 map 实际遍历顺序执行；某个数据源失败后继续处理其他数据源，命令最后以非零状态汇总错误。
- 全量生成会清空本次目标范围的输出目录：未指定 `database` 时清空整个 `base_path`，指定 `database` 时只清空对应数据源目录；指定表时不清理目录。
- 指定表时保留其他表产物，只更新指定表并重建聚合入口。
- 数据源目录名统一转小写并去掉连接符；规范化后冲突直接报错。
- 每套 `data` 包生成 `Models()`、`NewClient()`、`NewData()` 和 Repository ProviderSet，迁移模型只绑定当前数据源。
- 默认数据源的 `NewClient` 接收单个 `*configv1.Data_Database`；命名数据源的 `NewClient` 接收 `databases map[string]*configv1.Data_Database` 并按 key 取出当前配置。
- 模型、Repository 与字段名称沿用 `go-utils/stringcase` 的缩写规则；`BIGINT deleted_at` 保留 `soft_delete.DeletedAt` 生成策略。

## 结构

```text
gen/
├── api.go                    # 对外公开的生成器与配置入口
├── cmd/gorm-gen/             # CLI 适配层
└── internal/
    ├── config/               # data.yaml 读取、选择与批量编排
    └── generator/            # 单数据源生成核心
        └── templates/        # models/query/data 模板
```
