package generator

import (
	"strconv"
	"strings"

	"github.com/liujitcn/kratos-kit/database/gorm/driver"
	"gorm.io/gen"
	"gorm.io/gen/field"
)

// buildFieldTypeStrategy 构建字段类型归一化策略，把源库方言类型改写为多数据库通用的中立类型。
func (g *Gen) buildFieldTypeStrategy() gen.ModelOpt {
	return gen.FieldModify(func(field gen.Field) gen.Field {
		if field == nil || field.ColumnName == "" {
			return field
		}
		normalizeFieldGormTag(field.GORMTag, field.Type)
		return field
	})
}

// normalizeFieldGormTag 就地归一化字段 type 标签；方言敏感类型直接去掉标签，交由 GORM 按目标驱动推导。
func normalizeFieldGormTag(tag field.GormTag, goType string) {
	types, ok := tag["type"]
	if !ok || len(types) == 0 {
		return
	}
	kind := driverFieldKind(strings.TrimPrefix(goType, "*"))
	normalized := make([]string, 0, len(types))
	for _, rawType := range types {
		value, size, keep := driver.NormalizeColumnType(rawType, kind)
		if !keep {
			tag.Remove("type")
			if size > 0 {
				// 带长度的二进制列把长度迁入 size 标签，MySQL 据此生成可索引的 varbinary(N)。
				tag.Set("size", strconv.Itoa(size))
			}
			return
		}
		normalized = append(normalized, value)
	}
	tag.Set("type", normalized...)
}

// driverFieldKind 把 gorm-gen 的 Go 类型字符串归纳为共用映射的字段语义类别。
func driverFieldKind(goType string) driver.FieldKind {
	switch goType {
	case "string":
		return driver.FieldKindString
	case "bool":
		return driver.FieldKindBool
	case "time.Time":
		return driver.FieldKindTime
	case "[]byte", "[]uint8":
		return driver.FieldKindBytes
	case "float32", "float64":
		return driver.FieldKindFloat
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
		return driver.FieldKindInt
	default:
		return driver.FieldKindOther
	}
}
