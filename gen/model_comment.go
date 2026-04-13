package main

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

type modelCommentMeta struct {
	ModelName    string
	TableComment string
}

type modelCommentTemplateContext struct {
	PackageName string
	Comments    []modelCommentMeta
}

// generateModelCommentFile 为模型额外生成表注释方法，便于迁移阶段恢复表注释。
func generateModelCommentFile(opts options, tableModels []interface{}) error {
	var (
		modelDir string
		comments []modelCommentMeta
		ctx      modelCommentTemplateContext
		err      error
	)

	modelDir, err = resolveModelPath(opts.modelPkgPath)
	if err != nil {
		return err
	}

	comments, err = loadModelComments(tableModels)
	if err != nil {
		return err
	}

	ctx = modelCommentTemplateContext{
		PackageName: filepath.Base(modelDir),
		Comments:    comments,
	}
	err = writeModelCommentFile(filepath.Join(modelDir, "table_comment.gen.go"), ctx)
	if err != nil {
		return err
	}
	return nil
}

// loadModelComments 从 gorm/gen 导出的表结果中提取可生成的表注释信息。
func loadModelComments(tableModels []interface{}) ([]modelCommentMeta, error) {
	comments := make([]modelCommentMeta, 0, len(tableModels))
	for _, tableModel := range tableModels {
		comment, ok := extractModelCommentMeta(tableModel)
		if !ok {
			return nil, fmt.Errorf("解析模型表注释失败，类型=%T", tableModel)
		}
		if strings.TrimSpace(comment.TableComment) == "" {
			continue
		}
		comments = append(comments, comment)
	}
	sort.Slice(comments, func(i, j int) bool {
		return comments[i].ModelName < comments[j].ModelName
	})
	return comments, nil
}

// extractModelCommentMeta 从 gorm/gen 返回对象中提取模型名和表注释。
func extractModelCommentMeta(tableModel interface{}) (modelCommentMeta, bool) {
	value, ok := indirectStructValue(tableModel)
	if !ok {
		return modelCommentMeta{}, false
	}

	modelName, ok := readStringField(value, "ModelStructName")
	if !ok || modelName == "" {
		return modelCommentMeta{}, false
	}
	tableComment, ok := readStringField(value, "TableComment")
	if !ok {
		return modelCommentMeta{}, false
	}

	return modelCommentMeta{
		ModelName:    modelName,
		TableComment: tableComment,
	}, true
}

// writeModelCommentFile 根据模板渲染模型表注释文件，并自动格式化。
func writeModelCommentFile(filename string, data modelCommentTemplateContext) error {
	t, err := template.New(filepath.Base(filename)).Parse(modelCommentFileTemplate)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	err = t.Execute(&buf, data)
	if err != nil {
		return err
	}

	formatted, err := format.Source(append([]byte(generatedFileHeader), buf.Bytes()...))
	if err != nil {
		return fmt.Errorf("格式化文件%s失败: %w", filename, err)
	}
	return os.WriteFile(filename, formatted, 0o644)
}

// resolveModelPath 解析模型输出目录，兼容相对路径配置。
func resolveModelPath(modelPath string) (string, error) {
	if modelPath == "" {
		return "", fmt.Errorf("model 输出目录不能为空")
	}
	return filepath.Abs(modelPath)
}
