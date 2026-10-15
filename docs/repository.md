# repository 使用说明

根 module 的 `repository` 包把 `gorm/gen` 生成的 DAO 统一封装为泛型仓储接口。它只负责通用数据访问流程，业务条件仍通过生成字段和 `QueryOption` 表达。

## 创建仓储

`NewBaseRepository` 需要三个函数：

1. `queryDAO`：根据上下文返回当前表的 `gen.Dao`。
2. `idField`：返回当前表的 `field.Int64` 主键字段。
3. `id`：从实体读取主键值。

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

生成器会为每张表创建同样结构的基础仓储。手工创建仓储时，应保证 `idField` 和 `id` 的类型与实体主键一致；泛型仓储的按 ID 方法要求 `int64` 主键。

## BaseRepository 方法

| 方法 | 行为 |
| --- | --- |
| `Create` | 创建一条记录；实体为空时返回错误 |
| `BatchCreate` | 批量创建；空列表直接成功，并根据字段数自动计算批次 |
| `Find` | 按查询选项查询第一条记录；未传选项或选项全部为 `nil` 时返回错误 |
| `FindByID` | 按主键查询一条记录；ID 为 0 时返回错误 |
| `List` | 按查询选项查询列表，允许不带条件 |
| `ListByIDs` | 按 ID 列表查询；空列表返回空切片 |
| `Page` | 按页码、页大小查询列表，同时返回符合条件的总数 |
| `Count` | 按查询选项统计记录数 |
| `Update` | 按查询选项更新实体；必须存在实际 `WHERE` 条件 |
| `UpdateByID` | 使用实体主键更新；实体为空或 ID 为 0 时返回错误 |
| `Delete` | 按查询选项删除；必须存在实际 `WHERE` 条件 |
| `DeleteByID` | 按主键删除；ID 为 0 或未命中记录时按幂等成功处理 |
| `DeleteByIDs` | 按 ID 列表批量删除；空列表或未命中记录时按幂等成功处理 |

记录不存在的底层错误会统一归一为 `gorm.ErrRecordNotFound`，包括底层返回同名字符串错误的情况。

## 写操作保护

`Delete` 和 `Update` 会在应用所有 `QueryOption` 后检查 DAO 的最终 `WHERE` 子句：

```go
err := userRepository.Delete(ctx, repository.Where(query.User.Status.Eq(0)))
```

下面的调用会返回 `where condition is required`，不会执行 SQL：

```go
userRepository.Delete(ctx)
userRepository.Delete(ctx, repository.Order(query.User.ID))
userRepository.Update(ctx, user, repository.Limit(1))
```

检查的是实际生成的 `WHERE` 子句，因此只有排序、分页、选择列或空选项不能满足写操作的安全要求。`DeleteByID`、`DeleteByIDs` 和 `UpdateByID` 会自动追加主键条件。

## 批量写入

`BatchCreate` 根据实体结构估算单行写入字段数：

- 优先统计带 `gorm` 标签的可写字段，并递归展开匿名结构体。
- 没有可用 `gorm` 标签时按导出字段数做保守估算。
- 建议批次按 `999 / 字段数` 计算，最小为 1，最大为 1000。
- 无法估算字段数时使用每批 100 条。
- 实际批次不会超过待写入列表长度。

该策略只决定 `CreateInBatches` 的批次大小，不改变事务边界和 GORM 的字段写入规则。字段很多的实体可能生成更小的批次，字段无法反射识别时会使用兜底批次。

## 分页

`PageDefault` 将非法分页参数统一修正为 `page=1`、`size=10`：

```go
page, size := repository.PageDefault(request.Page, request.Size)
```

`PageOffsetLimit` 在相同规则下返回 `offset` 和 `limit`：

```go
offset, limit := repository.PageOffsetLimit(request.Page, request.Size)
```

`BaseRepository.Page` 会先查询当前页，再移除分页限制统计同一查询条件下的总数。传入的 `Order`、`Where`、`Join` 等选项会同时作用于列表查询和总数查询。

## QueryOption

`QueryOption` 的定义是 `func(dao gen.Dao) gen.Dao`，选项按传入顺序执行，`nil` 选项会跳过。支持的选项如下：

### 条件与结果

- `Where`、`Not`、`Or`：追加条件。
- `Select`、`Distinct`、`Omit`：控制返回列和去重。
- `Order`、`Limit`、`Offset`：控制排序和分页。
- `Group`、`Having`：分组与分组条件。
- `As`：指定当前查询别名。

### 关联与复用

- `Join`、`LeftJoin`、`RightJoin`：按生成表和字段追加连接。
- `Joins`：按关联字段连接查询。
- `Preload`：预加载关联。
- `Scopes`：复用多个 `func(gen.Dao) gen.Dao` 处理函数。
- `Unscoped`：关闭软删除过滤。

### 创建与底层子句

- `Attrs`：为 `FirstOrInit` 或 `FirstOrCreate` 指定默认字段。
- `Assign`：为 `FirstOrCreate` 指定赋值字段。
- `Clauses`：追加任意 GORM `clause.Expression`。

选项函数对空参数通常保持 DAO 不变；业务代码应直接传入生成的字段表达式，避免手工拼接列名和 SQL。

## 事务上下文

生成器输出的 `data` 包实现了 `QueryProvider` 和 `Transaction`。事务开始后，事务态查询对象放入上下文；仓储的 `queryDAO` 每次从上下文取得查询入口，因此同一个仓储实例可以同时服务普通请求和事务请求：

```go
err := data.Transaction(ctx, func(txCtx context.Context) error {
	return userRepository.UpdateByID(txCtx, user)
})
```

直接使用 `repository` 包时，调用方需要自行保证 `queryDAO` 根据上下文返回对应的事务 DAO。
