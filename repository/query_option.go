package repository

import (
	"gorm.io/gen"
	"gorm.io/gen/field"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// QueryOption 定义通用查询选项。
// 业务层应基于 gorm/gen 生成的强类型字段构建条件，避免 tag 与反射带来的不确定性。
type QueryOption func(dao gen.Dao) gen.Dao

// ApplyQueryOptions 按顺序应用查询选项。
func ApplyQueryOptions(dao gen.Dao, opts ...QueryOption) gen.Dao {
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		dao = opt(dao)
	}
	return dao
}

// As 为当前查询指定别名。
func As(alias string) QueryOption {
	return func(dao gen.Dao) gen.Dao {
		if alias == "" {
			return dao
		}
		return dao.As(alias)
	}
}

// Not 追加 NOT 条件。
func Not(conds ...gen.Condition) QueryOption {
	return func(dao gen.Dao) gen.Dao {
		if len(conds) == 0 {
			return dao
		}
		return dao.Not(conds...)
	}
}

// Or 追加 OR 条件。
func Or(conds ...gen.Condition) QueryOption {
	return func(dao gen.Dao) gen.Dao {
		if len(conds) == 0 {
			return dao
		}
		return dao.Or(conds...)
	}
}

// Order 将排序表达式追加到 dao。
func Order(columns ...field.Expr) QueryOption {
	return func(dao gen.Dao) gen.Dao {
		if len(columns) == 0 {
			return dao
		}
		return dao.Order(columns...)
	}
}

// Where 将条件表达式追加到 dao。
func Where(conds ...gen.Condition) QueryOption {
	return func(dao gen.Dao) gen.Dao {
		if len(conds) == 0 {
			return dao
		}
		return dao.Where(conds...)
	}
}

// Select 指定查询列。
func Select(columns ...field.Expr) QueryOption {
	return func(dao gen.Dao) gen.Dao {
		if len(columns) == 0 {
			return dao
		}
		return dao.Select(columns...)
	}
}

// Distinct 指定去重列。
func Distinct(columns ...field.Expr) QueryOption {
	return func(dao gen.Dao) gen.Dao {
		if len(columns) == 0 {
			return dao
		}
		return dao.Distinct(columns...)
	}
}

// Omit 排除列。
func Omit(columns ...field.Expr) QueryOption {
	return func(dao gen.Dao) gen.Dao {
		if len(columns) == 0 {
			return dao
		}
		return dao.Omit(columns...)
	}
}

// Join 追加内连接。
func Join(table schema.Tabler, conds ...field.Expr) QueryOption {
	return func(dao gen.Dao) gen.Dao {
		return dao.Join(table, conds...)
	}
}

// LeftJoin 追加左连接。
func LeftJoin(table schema.Tabler, conds ...field.Expr) QueryOption {
	return func(dao gen.Dao) gen.Dao {
		return dao.LeftJoin(table, conds...)
	}
}

// RightJoin 追加右连接。
func RightJoin(table schema.Tabler, conds ...field.Expr) QueryOption {
	return func(dao gen.Dao) gen.Dao {
		return dao.RightJoin(table, conds...)
	}
}

// Group 指定分组列。
func Group(columns ...field.Expr) QueryOption {
	return func(dao gen.Dao) gen.Dao {
		if len(columns) == 0 {
			return dao
		}
		return dao.Group(columns...)
	}
}

// Having 追加 Having 条件。
func Having(conds ...gen.Condition) QueryOption {
	return func(dao gen.Dao) gen.Dao {
		if len(conds) == 0 {
			return dao
		}
		return dao.Having(conds...)
	}
}

// Limit 指定返回条数。
func Limit(limit int) QueryOption {
	return func(dao gen.Dao) gen.Dao {
		return dao.Limit(limit)
	}
}

// Offset 指定偏移量。
func Offset(offset int) QueryOption {
	return func(dao gen.Dao) gen.Dao {
		return dao.Offset(offset)
	}
}

// Scopes 批量复用 Dao 处理函数。
func Scopes(funcs ...func(gen.Dao) gen.Dao) QueryOption {
	return func(dao gen.Dao) gen.Dao {
		if len(funcs) == 0 {
			return dao
		}
		return dao.Scopes(funcs...)
	}
}

// Unscoped 关闭软删除过滤。
func Unscoped() QueryOption {
	return func(dao gen.Dao) gen.Dao {
		return dao.Unscoped()
	}
}

// Attrs 指定 FirstOrInit/FirstOrCreate 的默认字段。
func Attrs(attrs ...field.AssignExpr) QueryOption {
	return func(dao gen.Dao) gen.Dao {
		if len(attrs) == 0 {
			return dao
		}
		return dao.Attrs(attrs...)
	}
}

// Assign 指定 FirstOrCreate 的赋值字段。
func Assign(attrs ...field.AssignExpr) QueryOption {
	return func(dao gen.Dao) gen.Dao {
		if len(attrs) == 0 {
			return dao
		}
		return dao.Assign(attrs...)
	}
}

// Joins 按关联字段追加 Joins 查询。
func Joins(field field.RelationField) QueryOption {
	return func(dao gen.Dao) gen.Dao {
		return dao.Joins(field)
	}
}

// Preload 按关联字段预加载。
func Preload(field field.RelationField) QueryOption {
	return func(dao gen.Dao) gen.Dao {
		return dao.Preload(field)
	}
}

// Clauses 追加底层 GORM 子句。
func Clauses(conds ...clause.Expression) QueryOption {
	return func(dao gen.Dao) gen.Dao {
		if len(conds) == 0 {
			return dao
		}
		return dao.Clauses(conds...)
	}
}
