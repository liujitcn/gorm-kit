package main

import (
	"strings"
	"unicode"
)

// buildModelName 将表名转换为模型名，保留常见缩写的大写形式。
func buildModelName(tableName string) string {
	parts := strings.Split(tableName, "_")
	for i, part := range parts {
		lowerPart := strings.ToLower(part)
		parts[i] = upperFirst(lowerPart)
	}
	return strings.Join(parts, "")
}

// buildRepoName 将表名转换为仓储名称，统一使用普通驼峰命名。
func buildRepoName(tableName string) string {
	parts := strings.Split(tableName, "_")
	for i, part := range parts {
		parts[i] = upperFirst(strings.ToLower(part))
	}
	return strings.Join(parts, "")
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
