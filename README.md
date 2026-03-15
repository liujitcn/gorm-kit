# gorm-kit

`gorm-kit` 是一个基于 GORM 的通用工具仓库，当前包含：

- `repo`：通用仓储与函数式查询选项（`QueryOption`）。
- `gen`：基于 `gorm/gen` 的代码生成封装。

## 目录说明

- `repo/`：仓储接口、批量写入策略、函数式查询选项。
- `gen/`：代码生成器（支持 `Driver`、`Source`、输出目录等配置）。

## 快速开始

### 1. 运行测试

```bash
go test ./...
```

### 2. 常用 Make 目标

当前 `Makefile` 仅保留以下目标：

```bash
make help
make tag
```

其中 `make tag` 会调用 `scripts/tag_release.py`，默认扫描仓库中的 `go.mod` 进行版本打标；也可以通过 `MODULE` 指定起始目录：

```bash
make tag MODULE=repo
```

### 3. 使用 BaseRepo + QueryOption（repo）

`repo` 层直接复用 `gorm/gen` 的强类型字段构建查询。`NewBaseRepo` 当前需要显式传入 `queryDAO`、主键字段访问器和实体主键读取函数：

```go
userRepo := repo.NewBaseRepo(
    func(ctx context.Context) gen.Dao { return query.Use(db).User.WithContext(ctx) },
    func(ctx context.Context) field.Int64 { return query.Use(db).User.WithContext(ctx).ID },
    func(entity *model.User) int64 { return entity.ID },
)

list, total, err := userRepo.ListPage(
    ctx,
    1,
    20,
    repo.Where(query.Use(db).User.Name.Like("%tom%")),
    repo.Where(query.Use(db).User.Status.Eq(1)),
    repo.Order(query.Use(db).User.CreatedAt.Desc()),
)
```

`Delete` 和 `Update` 这两个按条件执行的方法，必须显式传入至少一个非 `nil` 的 `QueryOption`，用于避免误删或误更新全表：

```go
err := userRepo.Delete(
    ctx,
    repo.Where(query.Use(db).User.ID.Eq(1001)),
)

err = userRepo.Update(
    ctx,
    &model.User{Status: 2},
    repo.Where(query.Use(db).User.ID.Eq(1001)),
)
```

如果需要联表、预加载、分组等复杂能力，可以直接使用 `Scope` 或对应的快捷函数：

```go
user := query.Use(db).User
role := query.Use(db).Role

list, err := userRepo.FindAll(
    ctx,
    repo.Select(user.ID, user.Name, role.Name),
    repo.LeftJoin(role, role.ID.EqCol(user.RoleID)),
    repo.Preload(user.Profile),
    repo.Scope(func(dao gen.Dao) gen.Dao {
        return dao.Where(user.DeletedAt.IsNull())
    }),
)
```

当前已提供的快捷查询选项包括：

- `As`
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

设计原则：

- 查询条件由业务层显式编排，避免字符串 tag 与运行时反射。
- 查询字段与操作符全部走 `gorm/gen` 强类型 API，错误尽量前置到编译期。
- `BaseRepo` 负责通用 CRUD、分页以及按条件更新/删除，不再承担 DSL 解释器的职责。

### 4. 使用代码生成器（gen）

参考：

- [gen/README.md](./gen/README.md)
