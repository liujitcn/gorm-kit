package generator

import (
	"database/sql"
	"fmt"
	"strings"

	"gorm.io/gen/field"
	"gorm.io/gen/helper"
)

// dorisColumn 保存 Doris information_schema.columns 返回的字段元数据。
type dorisColumn struct {
	ColumnName    string         `gorm:"column:column_name"`
	DataType      string         `gorm:"column:data_type"`
	ColumnType    string         `gorm:"column:column_type"`
	IsNullable    string         `gorm:"column:is_nullable"`
	ColumnDefault sql.NullString `gorm:"column:column_default"`
	ColumnComment sql.NullString `gorm:"column:column_comment"`
	ColumnKey     string         `gorm:"column:column_key"`
}

// dorisTable 保存 Doris 表注释。
type dorisTable struct {
	TableComment sql.NullString `gorm:"column:table_comment"`
}

// dorisObject 将 Doris 元数据适配为 gorm/gen helper.Object，避免 gorm/gen 对 Doris 追加 LIMIT 语句。
type dorisObject struct {
	tableName    string
	structName   string
	tableComment string
	fields       []helper.Field
}

// TableName 返回 Doris 表名。
func (o *dorisObject) TableName() string {
	return o.tableName
}

// StructName 返回生成的 Go 模型名。
func (o *dorisObject) StructName() string {
	return o.structName
}

// FileName 返回生成文件名。
func (o *dorisObject) FileName() string {
	return o.tableName
}

// ImportPkgPaths 返回模型所需的额外导入包。
func (o *dorisObject) ImportPkgPaths() []string {
	for _, modelField := range o.fields {
		if modelField.Type() == softDeleteFieldType {
			return []string{fmt.Sprintf("%q", softDeleteImportPkgPath)}
		}
	}
	return nil
}

// Fields 返回 Doris 字段定义。
func (o *dorisObject) Fields() []helper.Field {
	return o.fields
}

// dorisField 保存单个 Doris 字段的生成信息。
type dorisField struct {
	name       string
	dataType   string
	columnName string
	gormTag    string
	jsonTag    string
	comment    string
}

// Name 返回 Go 字段名。
func (f *dorisField) Name() string {
	return f.name
}

// Type 返回 Go 字段类型。
func (f *dorisField) Type() string {
	return f.dataType
}

// ColumnName 返回数据库字段名。
func (f *dorisField) ColumnName() string {
	return f.columnName
}

// GORMTag 返回 GORM 标签内容。
func (f *dorisField) GORMTag() string {
	return f.gormTag
}

// JSONTag 返回 JSON 标签内容。
func (f *dorisField) JSONTag() string {
	return f.jsonTag
}

// Tag 返回字段标签。
func (f *dorisField) Tag() field.Tag {
	return field.Tag{}
}

// Comment 返回字段注释。
func (f *dorisField) Comment() string {
	return f.comment
}

// generateAllDorisTables 根据 Doris 元数据生成当前数据库全部模型。
func (g *Gen) generateAllDorisTables() []interface{} {
	if g.db == nil {
		panic("Doris 数据库连接未初始化")
	}
	tableNames, err := g.db.Migrator().GetTables()
	if err != nil {
		panic(fmt.Errorf("读取 Doris 数据表失败: %w", err))
	}
	tableModels := make([]interface{}, 0, len(tableNames))
	for _, tableName := range tableNames {
		tableModels = append(tableModels, g.generateDorisModel(tableName))
	}
	return tableModels
}

// generateDorisModel 从 information_schema.columns 生成单个 Doris 模型。
func (g *Gen) generateDorisModel(tableName string) interface{} {
	if g.db == nil {
		panic("Doris 数据库连接未初始化")
	}

	var columns []dorisColumn
	err := g.db.Raw(`
		SELECT column_name, data_type, column_type, is_nullable, column_default, column_comment, column_key
		FROM information_schema.columns
		WHERE table_schema = DATABASE() AND table_name = ?
		ORDER BY ordinal_position
	`, tableName).Scan(&columns).Error
	if err != nil {
		panic(fmt.Errorf("读取 Doris 表%s字段失败: %w", tableName, err))
	}
	if len(columns) == 0 {
		panic(fmt.Errorf("Doris 表%s未发现字段", tableName))
	}

	tableComment := ""
	var table dorisTable
	err = g.db.Raw(`
		SELECT table_comment
		FROM information_schema.tables
		WHERE table_schema = DATABASE() AND table_name = ?
	`, tableName).Scan(&table).Error
	if err != nil {
		panic(fmt.Errorf("读取 Doris 表%s注释失败: %w", tableName, err))
	}
	if table.TableComment.Valid {
		tableComment = table.TableComment.String
	}

	modelFields := make([]helper.Field, 0, len(columns))
	hasPrimaryKey := false
	for _, column := range columns {
		modelField := buildDorisField(column)
		modelFields = append(modelFields, modelField)
		if strings.EqualFold(column.ColumnKey, "pri") || strings.EqualFold(column.ColumnKey, "uni") {
			hasPrimaryKey = true
		}
	}
	if !hasPrimaryKey {
		// Doris 视图没有主键，仓储模板仍需一个稳定字段作为基础查询键。
		modelField := modelFields[0].(*dorisField)
		modelField.gormTag += ";primaryKey"
	}

	object := &dorisObject{
		tableName:    tableName,
		structName:   buildModelName(tableName),
		tableComment: tableComment,
		fields:       modelFields,
	}
	if g.gormGenerator == nil {
		panic("Doris gorm/gen 生成器未初始化")
	}
	meta := g.gormGenerator.GenerateModelFrom(object)
	meta.TableComment = tableComment
	return meta
}

// buildDorisField 将 Doris 字段元数据转换为 gorm/gen helper.Field。
func buildDorisField(column dorisColumn) helper.Field {
	dataType := dorisGoType(column.DataType, column.ColumnType)
	gormType := column.ColumnType
	if gormType == "" {
		gormType = column.DataType
	}
	if column.ColumnName == softDeleteColumnName && (dataType == "int64" || dataType == "uint64") {
		dataType = softDeleteFieldType
	}
	gormTag := "column:" + column.ColumnName + ";type:" + gormType
	if strings.EqualFold(column.IsNullable, "no") {
		gormTag += ";not null"
	}
	if column.ColumnDefault.Valid && column.ColumnDefault.String != "" && !strings.EqualFold(column.ColumnDefault.String, "null") {
		gormTag += ";default:" + column.ColumnDefault.String
	}
	if strings.EqualFold(column.ColumnKey, "pri") || strings.EqualFold(column.ColumnKey, "uni") {
		gormTag += ";primaryKey"
	}
	if dataType == softDeleteFieldType {
		gormTag += ";softDelete:milli"
	}
	if column.ColumnComment.Valid && column.ColumnComment.String != "" {
		gormTag += ";comment:" + column.ColumnComment.String
	}
	return &dorisField{
		name:       buildModelName(column.ColumnName),
		dataType:   dataType,
		columnName: column.ColumnName,
		gormTag:    gormTag,
		jsonTag:    column.ColumnName,
		comment:    column.ColumnComment.String,
	}
}

// dorisGoType 将 Doris 数据类型映射为 Go 类型。
func dorisGoType(dataType, columnType string) string {
	normalizedType := strings.ToLower(strings.TrimSpace(dataType))
	normalizedColumnType := strings.ToLower(strings.TrimSpace(columnType))
	switch normalizedType {
	case "tinyint":
		if strings.HasPrefix(normalizedColumnType, "tinyint(1)") {
			return "bool"
		}
		return "int8"
	case "smallint":
		return "int16"
	case "int", "integer":
		return "int32"
	case "bigint", "largeint":
		return "int64"
	case "float":
		return "float32"
	case "double", "decimal":
		return "float64"
	case "boolean":
		return "bool"
	case "date", "datetime", "timestamp", "time":
		return "time.Time"
	case "binary", "varbinary", "hll", "bitmap", "quantile_state":
		return "[]byte"
	default:
		return "string"
	}
}
