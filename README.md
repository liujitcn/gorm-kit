# gorm-kit

`gorm-kit` 是一个基于 GORM 的通用工具仓库，当前包含：

- `repo`：通用仓储与动态查询构建（`BuildDao`）。
- `gen`：基于 `gorm/gen` 的代码生成封装。
- `api`：项目内使用的 protobuf 生成代码（如 `conf.Operator`、`conf.Sort`）。

## 目录说明

- `repo/`：仓储接口、批量写入策略、`search tag` 查询构建。
- `gen/`：代码生成器（支持 `Driver`、`Source`、输出目录等配置）。
- `api/`：proto 定义与生成产物。

## 快速开始

### 1. 运行测试

```bash
go test ./...
```

### 2. 使用 BuildDao（repo）

`condition` 字段通过 `search` tag 描述查询条件：

```go
type UserSearch struct {
    Name string `search:"type:like;column:name"`
    Sort int32  `search:"type:order;column:sort"`
}
```

约束说明：

- `model` 必须实现 `TableName() string`。
- `type` 必须对应 `conf.Operator` 枚举名（内部会转大写匹配，推荐统一使用小写）。
- `type` 可选值：`order`、`eq`、`neq`、`gt`、`gte`、`lt`、`lte`、`like`、`not_like`、`in`、`nin`、`is_null`、`is_not_null`、`between`、`regexp`、`contains`、`starts_with`、`ends_with`。
- `table` 为空时，`column` 必须能在模型 `gorm:"column:xxx"` 中匹配。
- `table` 不为空时，直接使用 `table+column` 构建查询。

`type` 语句映射（`column` 代表 tag 中列名，`value` 代表条件值）：

| type | 最终 SQL 语句（模板） | 中文说明 |
| --- | --- | --- |
| `order` | `ORDER BY {table}.{column} ASC|DESC` | 排序（解析 `conf.Sort`，不匹配默认 `ASC`）。 |
| `eq` | `{table}.{column} = ?` | 等于。 |
| `neq` | `{table}.{column} <> ?` | 不等于。 |
| `gt` | `{table}.{column} > ?` | 大于。 |
| `gte` | `{table}.{column} >= ?` | 大于等于。 |
| `lt` | `{table}.{column} < ?` | 小于。 |
| `lte` | `{table}.{column} <= ?` | 小于等于。 |
| `like` | `{table}.{column} LIKE ?` | 模糊匹配（值原样使用）。 |
| `not_like` | `{table}.{column} NOT LIKE ?` | 反向模糊匹配。 |
| `in` | `{table}.{column} IN (?, ?, ...)` | 在集合中（值必须是切片/数组）。 |
| `nin` | `{table}.{column} NOT IN (?, ?, ...)` | 不在集合中（值必须是切片/数组）。 |
| `is_null` | `{table}.{column} IS NULL` | 为空判断（不使用 value）。 |
| `is_not_null` | `{table}.{column} IS NOT NULL` | 非空判断（不使用 value）。 |
| `between` | `{table}.{column} BETWEEN ? AND ?` | 区间匹配（值必须是长度 2 的切片/数组）。 |
| `regexp` | `{table}.{column} REGEXP ?` | 正则匹配。 |
| `contains` | `{table}.{column} LIKE '%value%'` | 包含匹配（内部自动拼 `%`）。 |
| `starts_with` | `{table}.{column} LIKE 'value%'` | 前缀匹配（内部自动拼 `%`）。 |
| `ends_with` | `{table}.{column} LIKE '%value'` | 后缀匹配（内部自动拼 `%`）。 |

### 3. 使用代码生成器（gen）

参考：

- [gen/README.md](./gen/README.md)
