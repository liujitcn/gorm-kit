# gorm-kit

`gorm-kit` 是一个基于 GORM 的通用工具仓库，当前主要包含两个模块：

- `repo`：通用仓储能力与函数式查询选项
- `gen`：基于 `gorm/gen` 的代码生成入口

## 目录说明

- `repo/`：仓储接口、分页、批量写入策略、函数式查询选项
- `gen/`：生成 `models`、`query`、`data` 的代码生成器

## 测试

在仓库根目录执行：

```bash
go test ./...
```

## repo

`repo` 层直接复用 `gorm/gen` 的强类型字段构建查询。`NewBaseRepo` 需要显式传入：

- `queryDAO`
- 主键字段访问器
- 实体主键读取函数

示例：

```go
userRepo := repo.NewBaseRepo(
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

## gen

`gen` 当前支持：

- 默认读取 `gen/config.yaml`
- 使用 `-config` 指定配置文件路径
- 使用 `-set key=value` 覆盖配置文件中的单项字段
- 联动生成 `models`、`query`、`data`
- 每次生成 `data` 前自动删除目标目录，避免旧文件残留

示例：

```bash
cd gen
go run .
go run . -config ./config.yaml
go run . -config ./config.yaml -set model_pkg_path=models1/tst -set out_path=query1/tet -set data_path=./data1
```

当前支持覆盖的配置项：

- `driver`
- `source`
- `out_path`
- `model_pkg_path`
- `data_path`
- `acronyms.xxx`

更完整说明见：

- [gen/README.md](./gen/README.md)
