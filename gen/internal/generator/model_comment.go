package generator

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
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
	if opts.table != "" {
		var existingComments []modelCommentMeta
		existingComments, err = loadGeneratedModelComments(filepath.Join(modelDir, "table_comment.gen.go"))
		if err != nil {
			return err
		}
		comments, err = mergeModelComments(existingComments, tableModels)
		if err != nil {
			return err
		}
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

// loadGeneratedModelComments 从已有生成文件中读取模型表注释。
func loadGeneratedModelComments(filename string) ([]modelCommentMeta, error) {
	file, err := parser.ParseFile(token.NewFileSet(), filename, nil, 0)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("解析模型表注释文件失败: %w", err)
	}
	comments := make([]modelCommentMeta, 0)
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name.Name != "TableComment" || function.Recv == nil || len(function.Recv.List) != 1 || function.Body == nil || len(function.Body.List) != 1 {
			continue
		}
		modelName := receiverModelName(function.Recv.List[0].Type)
		returnStatement, ok := function.Body.List[0].(*ast.ReturnStmt)
		if !ok || modelName == "" || len(returnStatement.Results) != 1 {
			continue
		}
		literal, ok := returnStatement.Results[0].(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			continue
		}
		tableComment, unquoteErr := strconv.Unquote(literal.Value)
		if unquoteErr != nil {
			return nil, fmt.Errorf("解析模型%s表注释失败: %w", modelName, unquoteErr)
		}
		comments = append(comments, modelCommentMeta{ModelName: modelName, TableComment: tableComment})
	}
	return comments, nil
}

// receiverModelName 返回表注释方法接收者的模型名称。
func receiverModelName(expression ast.Expr) string {
	if star, ok := expression.(*ast.StarExpr); ok {
		expression = star.X
	}
	identifier, ok := expression.(*ast.Ident)
	if !ok {
		return ""
	}
	return identifier.Name
}

// mergeModelComments 使用本次生成结果替换同名模型注释并保留其他模型。
func mergeModelComments(existingComments []modelCommentMeta, tableModels []interface{}) ([]modelCommentMeta, error) {
	commentMap := make(map[string]string, len(existingComments)+len(tableModels))
	for _, comment := range existingComments {
		commentMap[comment.ModelName] = comment.TableComment
	}
	for _, tableModel := range tableModels {
		comment, ok := extractModelCommentMeta(tableModel)
		if !ok {
			return nil, fmt.Errorf("解析模型表注释失败，类型=%T", tableModel)
		}
		delete(commentMap, comment.ModelName)
		if strings.TrimSpace(comment.TableComment) != "" {
			commentMap[comment.ModelName] = comment.TableComment
		}
	}
	comments := make([]modelCommentMeta, 0, len(commentMap))
	for modelName, tableComment := range commentMap {
		comments = append(comments, modelCommentMeta{ModelName: modelName, TableComment: tableComment})
	}
	sort.Slice(comments, func(i, j int) bool {
		return comments[i].ModelName < comments[j].ModelName
	})
	return comments, nil
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
	t, err := parseTemplateFile(modelCommentFileTemplate)
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
