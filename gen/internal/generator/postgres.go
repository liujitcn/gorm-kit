package generator

import (
	"database/sql"
	"reflect"
	"strings"
)

// applyPostgresTableComment 为 postgres 数据源的生成模型补写表注释。
//
// gorm 的 postgres 驱动未实现 Migrator.TableType（固定返回 not support），
// gorm/gen 的 getTableComment 因此拿不到表注释，模型注释与 table_comment.gen.go 都会缺失；
// 这里直接查询 pg_description，并通过反射写回 gorm/gen 的生成模型
// （internal 包不可跨模块导入，字段访问只能走反射），
// 供模型与查询文件渲染、迁移期表注释恢复共同使用。
func (g *Gen) applyPostgresTableComment(tableName string, tableModel interface{}) {
	var comment sql.NullString
	// 未限定 schema 的表名按连接 search_path 解析，与 gorm/gen 自身的表查找保持一致。
	if err := g.db.Raw(`SELECT obj_description(?::regclass, 'pg_class')`, tableName).Scan(&comment).Error; err != nil {
		return
	}
	if !comment.Valid || strings.TrimSpace(comment.String) == "" {
		return
	}
	value := reflect.ValueOf(tableModel)
	if value.Kind() != reflect.Ptr || value.Elem().Kind() != reflect.Struct {
		return
	}
	field := value.Elem().FieldByName("TableComment")
	if field.IsValid() && field.CanSet() && field.Kind() == reflect.String {
		field.SetString(comment.String)
	}
}

// postgresModelTableName 从 gorm/gen 生成模型上反射读取表名，供全表生成路径补写表注释。
func postgresModelTableName(tableModel interface{}) string {
	value := reflect.ValueOf(tableModel)
	if value.Kind() != reflect.Ptr || value.Elem().Kind() != reflect.Struct {
		return ""
	}
	field := value.Elem().FieldByName("TableName")
	if !field.IsValid() || field.Kind() != reflect.String {
		return ""
	}
	return field.String()
}
