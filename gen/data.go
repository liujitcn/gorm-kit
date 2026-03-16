package main

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"text/template"
)

type tableMeta struct {
	TableName string
	ModelName string
	RepoName  string
}

type dataTemplateContext struct {
	PackageName     string
	ModelPackage    string
	QueryPackage    string
	ModelImportPath string
	QueryImportPath string
	Tables          []tableMeta
}

// generateDataFiles 根据 gorm/gen 导出的全部表结果生成 data 包代码。
func generateDataFiles(opts options, tableModels []interface{}) error {
	dataDir, err := resolveDataPath(opts.dataPath)
	if err != nil {
		return err
	}
	tables, err := loadTables(tableModels)
	if err != nil {
		return err
	}
	ctx, err := buildDataTemplateContext(dataDir, opts, tables)
	if err != nil {
		return err
	}
	if err := generateDataLayer(dataDir, ctx); err != nil {
		return err
	}
	return nil
}

// loadTables 从 gorm/gen 导出的全部表结果中提取表元信息，并按字典序返回。
func loadTables(tableModels []interface{}) ([]tableMeta, error) {
	metas := make([]tableMeta, 0, len(tableModels))
	for _, tableModel := range tableModels {
		meta, ok := extractTableMeta(tableModel)
		if !ok {
			return nil, fmt.Errorf("解析表元信息失败，类型=%T", tableModel)
		}
		metas = append(metas, meta)
	}
	sort.Slice(metas, func(i, j int) bool {
		return metas[i].TableName < metas[j].TableName
	})
	return metas, nil
}

// extractTableMeta 从 gorm/gen 返回对象中提取 data 层所需的表信息。
func extractTableMeta(tableModel any) (tableMeta, bool) {
	if tableModel == nil {
		return tableMeta{}, false
	}
	value := reflect.ValueOf(tableModel)
	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return tableMeta{}, false
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return tableMeta{}, false
	}

	tableName, ok := readStringField(value, "TableName")
	if !ok || tableName == "" {
		return tableMeta{}, false
	}
	modelName, ok := readStringField(value, "ModelStructName")
	if !ok || modelName == "" {
		// 回退到表名推导，兼容返回对象字段变化场景。
		modelName = buildModelName(tableName)
	}
	return tableMeta{
		TableName: tableName,
		ModelName: modelName,
		RepoName:  buildRepoName(tableName),
	}, true
}

// readStringField 读取结构体中的字符串字段。
func readStringField(value reflect.Value, fieldName string) (string, bool) {
	fieldValue := value.FieldByName(fieldName)
	if !fieldValue.IsValid() || fieldValue.Kind() != reflect.String {
		return "", false
	}
	return fieldValue.String(), true
}

// generateDataLayer 生成 data 包中的基础仓储、迁移注册与 ProviderSet。
func generateDataLayer(dataDir string, ctx dataTemplateContext) error {
	// 每次生成前先清理旧目录，避免历史文件残留导致无效代码继续存在。
	if err := os.RemoveAll(dataDir); err != nil {
		return err
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}
	if err := writeTemplateFile(filepath.Join(dataDir, "data.go"), dataFileTemplate, ctx); err != nil {
		return err
	}
	if err := writeTemplateFile(filepath.Join(dataDir, "init.go"), initFileTemplate, ctx); err != nil {
		return err
	}
	for _, table := range ctx.Tables {
		if err := writeTemplateFile(filepath.Join(dataDir, table.TableName+".go"), repoFileTemplate, struct {
			PackageName     string
			ModelPackage    string
			ModelImportPath string
			Table           tableMeta
		}{
			PackageName:     ctx.PackageName,
			ModelPackage:    ctx.ModelPackage,
			ModelImportPath: ctx.ModelImportPath,
			Table:           table,
		}); err != nil {
			return err
		}
	}
	return nil
}

// writeTemplateFile 根据模板渲染 Go 文件，并自动格式化后写入磁盘。
func writeTemplateFile(filename, tpl string, data any) error {
	t, err := template.New(filepath.Base(filename)).Funcs(template.FuncMap{
		"lowerFirst": lowerFirst,
	}).Parse(tpl)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err = t.Execute(&buf, data); err != nil {
		return err
	}
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("格式化文件%s失败: %w", filename, err)
	}
	return os.WriteFile(filename, formatted, 0o644)
}

// resolveDataPath 解析 data 输出目录，兼容相对路径配置。
func resolveDataPath(dataPath string) (string, error) {
	if dataPath == "" {
		return "", fmt.Errorf("data 输出目录不能为空")
	}
	return filepath.Abs(dataPath)
}

// buildDataTemplateContext 构建 data 模板渲染所需的导入路径上下文。
func buildDataTemplateContext(dataDir string, opts options, tables []tableMeta) (dataTemplateContext, error) {
	modulePath, err := resolveModulePath(dataDir)
	if err != nil {
		return dataTemplateContext{}, err
	}
	return dataTemplateContext{
		PackageName:     buildPackageName(dataDir),
		ModelPackage:    buildPackageName(opts.modelPkgPath),
		QueryPackage:    buildPackageName(opts.outPath),
		ModelImportPath: buildImportPath(modulePath, opts.modelPkgPath),
		QueryImportPath: buildImportPath(modulePath, opts.outPath),
		Tables:          tables,
	}, nil
}

// resolveModulePath 从 data 输出目录向上查找 go.mod，并解析模块名。
func resolveModulePath(startDir string) (string, error) {
	current := filepath.Clean(startDir)
	for {
		goModPath := filepath.Join(current, "go.mod")
		content, err := os.ReadFile(goModPath)
		if err == nil {
			modulePath := parseModulePath(string(content))
			if modulePath == "" {
				return "", fmt.Errorf("解析模块路径失败: %s", goModPath)
			}
			return modulePath, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("未找到 go.mod: %s", startDir)
		}
		current = parent
	}
}

// parseModulePath 从 go.mod 内容中提取 module 路径。
func parseModulePath(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

// buildImportPath 根据模块路径和目录配置拼接 import 路径。
func buildImportPath(modulePath, dir string) string {
	cleanDir := strings.Trim(strings.ReplaceAll(filepath.ToSlash(dir), "\\", "/"), "/")
	if cleanDir == "" || cleanDir == "." {
		return modulePath
	}
	return modulePath + "/" + cleanDir
}

// buildPackageName 根据输出目录推导 Go package 名称。
func buildPackageName(dir string) string {
	name := filepath.Base(filepath.Clean(dir))
	if name == "" || name == "." || name == string(filepath.Separator) {
		return "data"
	}
	return name
}
