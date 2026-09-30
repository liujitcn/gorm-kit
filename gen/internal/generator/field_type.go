package generator

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/liujitcn/kratos-kit/database/gorm/driver"
	"gorm.io/gen"
	"gorm.io/gen/field"
)

// binarySizeDefPattern 匹配长度检查约束定义中的列名与长度，
// 约束由运行时迁移生成，形如 CHECK (octet_length("digest") <= 32)；
// PG 会把表达式规范化出多层括号，闭合括号与比较符之间需容忍。
var binarySizeDefPattern = regexp.MustCompile(`octet_length\(\s*"?([A-Za-z_0-9]+)"?\s*\)\s*\)*\s*<=\s*(\d+)`)

// buildFieldTypeStrategy 构建字段类型归一化策略，把源库方言类型改写为多数据库通用的中立类型。
func (g *Gen) buildFieldTypeStrategy() gen.ModelOpt {
	return gen.FieldModify(func(field gen.Field) gen.Field {
		if field == nil || field.ColumnName == "" {
			return field
		}
		table := ""
		if field.Column != nil {
			table = field.Column.TableName
		}
		normalizeFieldGormTag(field.GORMTag, field.Type, table, field.ColumnName, g.binarySizes)
		return field
	})
}

// normalizeFieldGormTag 就地归一化字段 type 标签；方言敏感类型直接去掉标签，交由 GORM 按目标驱动推导。
// binarySizes 承载 PostgreSQL 源库中由长度检查约束找回的二进制列长度，其余源库传空。
func normalizeFieldGormTag(tag field.GormTag, goType, table, column string, binarySizes map[string]map[string]int) {
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
			// bytea 无长度概念，长度信息由检查约束找回，保证与 MySQL 源库生成的模型一致。
			if size == 0 && kind == driver.FieldKindBytes {
				size = binarySizes[table][column]
			}
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

// loadPostgresBinarySizes 收集 PostgreSQL 源库中 gormsize_ 前缀检查约束承载的二进制列长度。
func (g *Gen) loadPostgresBinarySizes() error {
	type constraintRow struct {
		TableName string `gorm:"column:table_name"`
		Def       string `gorm:"column:def"`
	}
	rows := make([]constraintRow, 0)
	if err := g.db.Raw(`SELECT c.relname AS table_name, pg_get_constraintdef(con.oid) AS def
		FROM pg_constraint con
		JOIN pg_class c ON c.oid = con.conrelid
		WHERE con.contype = 'c' AND con.conname LIKE 'gormsize\_%'`).Scan(&rows).Error; err != nil {
		return err
	}
	g.binarySizes = make(map[string]map[string]int)
	for _, row := range rows {
		matches := binarySizeDefPattern.FindStringSubmatch(row.Def)
		if matches == nil {
			continue
		}
		size, err := strconv.Atoi(matches[2])
		if err != nil || size <= 0 {
			continue
		}
		if g.binarySizes[row.TableName] == nil {
			g.binarySizes[row.TableName] = make(map[string]int)
		}
		g.binarySizes[row.TableName][matches[1]] = size
	}
	return nil
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
