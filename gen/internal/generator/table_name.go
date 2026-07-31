package generator

import (
	"os"
	"path/filepath"
)

// normalizeLegacyTableNameFile 清空历史辅助文件中的重复方法，表名映射由 GORM 模型文件提供。
func normalizeLegacyTableNameFile(opts options) error {
	modelDir, err := resolveModelPath(opts.modelPkgPath)
	if err != nil {
		return err
	}
	filename := filepath.Join(modelDir, "table_name.gen.go")
	_, err = os.Stat(filename)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return writeTemplateFile(filename, tableNameFileTemplate, struct {
		PackageName string
	}{
		PackageName: filepath.Base(modelDir),
	})
}
