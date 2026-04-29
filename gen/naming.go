package main

import (
	"strings"
	"unicode"

	"github.com/liujitcn/go-utils/set"
)

// commonAbbreviations 定义生成命名时需要保持全大写的 Go 常见缩写。
var commonAbbreviations = set.New[string](
	"ACL",
	"API",
	"ASCII",
	"CDN",
	"COS",
	"CPU",
	"CRM",
	"CSS",
	"DNS",
	"EOF",
	"ERP",
	"GPS",
	"GUID",
	"HTML",
	"HTTP",
	"HTTPS",
	"ID",
	"IM",
	"IP",
	"JSON",
	"JWT",
	"LBS",
	"LHS",
	"LLM",
	"MFA",
	"MQ",
	"OMS",
	"OSS",
	"OTP",
	"POS",
	"QPS",
	"QR",
	"RAM",
	"RBAC",
	"RHS",
	"RPC",
	"S3",
	"SKU",
	"SLA",
	"SMTP",
	"SMS",
	"SPU",
	"SSH",
	"SSO",
	"TLS",
	"TTL",
	"UID",
	"UI",
	"UUID",
	"URI",
	"URL",
	"UTF8",
	"VM",
	"WMS",
	"XML",
	"XSRF",
	"XSS",
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
	parts := strings.Split(name, "_")
	for i, part := range parts {
		parts[i] = buildCamelNamePart(part)
	}
	return strings.Join(parts, "")
}

// buildCamelNamePart 转换单个名称片段，命中常见缩写时保持全大写。
func buildCamelNamePart(part string) string {
	upperPart := strings.ToUpper(part)
	if commonAbbreviations.Contains(upperPart) {
		// 保持 URI、ID、SKU、LLM 等统一缩写表中的词全大写。
		return upperPart
	}
	return upperFirst(strings.ToLower(part))
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

// upperFirst 将字符串首字母转换为大写。
func upperFirst(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
