# gorm-kit

`gorm-kit` 是基于 GORM 和 `gorm/gen` 的 Go 工具仓库，提供通用仓储实现与数据库代码生成器。仓库包含两个相互独立的 Go module：

| module | 目录 | 作用 |
| --- | --- | --- |
| `github.com/liujitcn/gorm-kit` | 根目录 | `repository` 通用仓储、分页和强类型查询选项 |
| `github.com/liujitcn/gorm-kit/gen` | `gen/` | 根据数据库结构生成 `models`、`query` 和 `data` 代码 |

生成器输出的 `data` 包会引用根 module 的 `repository` 包，因此使用生成代码的业务 module 通常需要同时依赖这两个 module。

## 能力概览

- 泛型 `BaseRepository[T]`：覆盖创建、批量创建、查询、分页、统计、更新和删除。
- 强类型查询选项：直接接收 `gorm/gen` 字段表达式，支持条件、排序、连接、分组、预加载和底层子句。
- 写操作保护：`Delete` 和 `Update` 要求最终 DAO 中存在实际 `WHERE` 条件，避免无条件全表写入。
- 自适应批量写入：依据实体字段数估算单批大小，兼容常见 SQL 参数上限并限制最大批次。
- 配置驱动生成：读取服务配置中的 `data.database` 或 `data.databases`，按数据源和表范围生成代码。
- 多数据库生成：支持 MySQL、Doris、PostgreSQL、SQLite、SQL Server、Oracle 和 BigQuery 的驱动注册。
- 生成运行时骨架：输出查询入口、事务封装、数据访问对象、Wire `ProviderSet` 和按表生成的基础仓储。

## 环境要求

- Go 1.27.0 或更高版本。
- 运行生成器时，目标数据库必须可访问，连接串由配置文件提供。
- 执行 `make fmt` 需要 `goimports`；执行 `make init` 可安装日常开发工具和生成器命令。

## 快速开始

### 验证两个 module

```bash
go test ./...
(cd gen && go test ./...)
```

### 初始化工具

```bash
make init
```

`make init` 会安装当前仓库的生成器命令和格式化所需的 `goimports`。需要格式化时执行：

```bash
make fmt
```

### 使用仓储

仓储需要三个与生成查询代码绑定的函数：返回当前上下文的 DAO、返回主键字段和读取实体主键。下面的 `query.User`、`model.User` 是生成器输出的示例名称：

```go
userRepository := repository.NewBaseRepository(
	func(ctx context.Context) gen.Dao {
		return query.Use(db).User.WithContext(ctx)
	},
	func(ctx context.Context) field.Int64 {
		return query.Use(db).User.WithContext(ctx).ID
	},
	func(entity *model.User) int64 {
		return entity.ID
	},
)
```

仓储方法、查询选项和边界行为见 [docs/repository.md](docs/repository.md)。

### 使用生成器

```bash
cd gen
go run ./cmd/gorm-gen -h
go run ./cmd/gorm-gen -config=./configs/data.yaml
go run ./cmd/gorm-gen -config=./configs/data.yaml -database=main
go run ./cmd/gorm-gen -config=./configs/data.yaml -database=main -table=user,user_profile
```

默认配置文件是执行目录下的 `./configs/data.yaml`，默认输出根目录是 `gen`。生成器的配置格式、数据源选择和输出清理规则见 [gen/README.md](gen/README.md)。

## 目录结构

```text
.
├── repository/       # 通用仓储与查询选项
├── gen/              # 独立的代码生成 module
│   ├── cmd/gorm-gen/ # 命令行入口
│   ├── internal/     # 配置解析、元数据读取和模板生成
│   └── configs/      # 默认配置示例
├── scripts/          # 仓库维护脚本
├── Makefile          # 初始化、格式化和发布辅助命令
├── go.mod            # 根 module
└── gen/go.mod        # 生成器 module
```

## 生成结果

以默认输出目录 `gen` 为例，生成器会创建以下目录：

```text
gen/
├── models/  # GORM 模型、字段标签和表注释方法
├── query/   # gorm/gen 强类型查询入口
└── data/    # Data、事务、迁移模型、ProviderSet 和基础仓储
```

命名数据源会生成到 `base_path/<数据源目录>`，目录名由数据源名称中的字母和数字组成。全量生成只清理目标数据源下的 `query`、`models` 和 `data`；指定表生成时保留其他表的既有产物，并重建聚合入口。

生成的 `data.Models()` 会排除数据库视图，避免把视图加入自动迁移模型；视图仍可生成查询和基础仓储代码。`NewData` 接收外部创建的数据库客户端映射，并把事务查询对象写入上下文，仓储可在事务内透明复用。

## Makefile 命令

| 命令 | 作用 |
| --- | --- |
| `make init` | 安装生成器命令、格式化工具和日常 CLI |
| `make plugin` | 仅安装 `gorm-gen` 命令 |
| `make cli` | 安装 `goimports` 和 import 规范化命令 |
| `make fmt` | 规范化 Go import 并执行 `goimports` |
| `make tag` | 运行仓库内置的递归版本标记脚本 |

## 验证约定

代码改动至少应分别验证根 module 和 `gen` module：

```bash
go test ./...
(cd gen && go test ./...)
```

生成器改动还应使用一个可访问的数据源执行实际生成，并检查 `models`、`query`、`data` 三个目录中的产物能通过业务 module 编译。
