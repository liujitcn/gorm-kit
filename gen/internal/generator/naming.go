package generator

import (
	"unicode"

	"github.com/liujitcn/go-utils/stringcase"
)

// buildModelName 将表名转换为模型名，保留常见缩写的大写形式。
func buildModelName(tableName string) string {
	return buildCamelName(tableName)
}

// buildRepositoryName 将表名转换为仓储名称，保留常见缩写的大写形式。
func buildRepositoryName(tableName string) string {
	return buildCamelName(tableName)
}

// buildCamelName 将下划线名称转换为驼峰名称，并按 Go 习惯保留常见缩写。
func buildCamelName(name string) string {
	return stringcase.ToGoPascalCase(name)
}

// lowerFirst 将字符串首字母转换为小写。
func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}
