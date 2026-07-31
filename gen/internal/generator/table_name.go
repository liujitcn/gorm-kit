package generator

import "path/filepath"

// generateTableNameFile 生成模型到真实表名的 GORM 映射方法。
func generateTableNameFile(opts options, tableModels []interface{}) error {
	var tables []tableMeta
	var err error
	if opts.table == "" {
		tables, err = loadTables(tableModels)
	} else {
		tables, err = loadGeneratedTableMetas(opts.modelPkgPath, opts.outPath)
	}
	if err != nil {
		return err
	}
	var modelDir string
	modelDir, err = resolveModelPath(opts.modelPkgPath)
	if err != nil {
		return err
	}
	return writeTemplateFile(filepath.Join(modelDir, "table_name.gen.go"), tableNameFileTemplate, struct {
		PackageName string
		Tables      []tableMeta
	}{
		PackageName: filepath.Base(modelDir),
		Tables:      tables,
	})
}
