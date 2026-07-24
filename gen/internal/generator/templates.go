package generator

import (
	"embed"
	"fmt"
	"text/template"
)

const (
	queryFileTemplate        = "query.go.tmpl"
	dataFileTemplate         = "data.go.tmpl"
	initFileTemplate         = "init.go.tmpl"
	repositoryFileTemplate   = "repository.go.tmpl"
	modelCommentFileTemplate = "model_comment.go.tmpl"
)

// templateFS 嵌入生成器使用的模板文件，避免运行时依赖当前工作目录。
//
//go:embed templates/*.tmpl
var templateFS embed.FS

// parseTemplateFile 从嵌入文件系统读取并解析指定模板。
func parseTemplateFile(filename string) (*template.Template, error) {
	content, err := templateFS.ReadFile("templates/" + filename)
	if err != nil {
		return nil, fmt.Errorf("读取模板文件%s失败: %w", filename, err)
	}
	return template.New(filename).Funcs(template.FuncMap{
		"lowerFirst": lowerFirst,
	}).Parse(string(content))
}
