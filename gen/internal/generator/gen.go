package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/liujitcn/kratos-kit/database/gorm/driver"
	gormgen "gorm.io/gen"
	"gorm.io/gorm"
)

const (
	// softDeleteColumnName 软删除列名。
	softDeleteColumnName = "deleted_at"
	// softDeleteFieldType 软删除字段生成类型，配合联合唯一索引支持编号复用。
	softDeleteFieldType = "soft_delete.DeletedAt"
	// softDeleteImportPkgPath 软删除字段类型所在包路径。
	softDeleteImportPkgPath = "gorm.io/plugin/soft_delete"
)

// Gen 封装 gorm/gen 生成能力。
type Gen struct {
	opts options
}

// NewGen 创建生成器实例。
func NewGen(opts ...Option) *Gen {
	o := defaultOptions()
	// 按传入顺序应用 Option，后面的配置可以覆盖前面的配置。
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}
	o.ApplyBasePath()
	return &Gen{opts: o}
}

// Execute 执行代码生成。
func (g *Gen) Execute() error {
	_, err := g.Generate()
	return err
}

// Generate 按配置执行全部表或单表代码生成，并返回本次生成的表结果。
func (g *Gen) Generate() ([]interface{}, error) {
	generator, err := g.newGenerator()
	if err != nil {
		return nil, err
	}

	var tableModels []interface{}
	tableModels, err = g.generateTableModels(generator)
	if err != nil {
		return nil, err
	}
	// 只有成功读取到当前库表结构后才清理旧目录，避免连接异常时误删上次生成结果。
	if err = g.cleanupGeneratedDirs(); err != nil {
		return nil, err
	}

	// 4. 基于本次选中的表生成模型与查询代码。
	generator.ApplyBasic(tableModels...)
	generator.Execute()
	if g.opts.table != "" {
		var tables []tableMeta
		tables, err = loadGeneratedTableMetas(g.opts.modelPkgPath, g.opts.outPath)
		if err != nil {
			return nil, err
		}
		if err = writeGeneratedQueryFile(g.opts.outPath, tables); err != nil {
			return nil, err
		}
	}
	if err = generateModelCommentFile(g.opts, tableModels); err != nil {
		return nil, err
	}
	if err = generateDataFiles(g.opts, tableModels); err != nil {
		return nil, err
	}
	return tableModels, nil
}

// loadGeneratedTableMetas 从已有模型与查询文件中恢复聚合入口需要的全部表信息。
func loadGeneratedTableMetas(modelPath string, queryPath string) ([]tableMeta, error) {
	modelDir, err := resolveModelPath(modelPath)
	if err != nil {
		return nil, err
	}
	var queryDir string
	queryDir, err = resolveGeneratedPath("query", queryPath)
	if err != nil {
		return nil, err
	}
	var entries []os.DirEntry
	entries, err = os.ReadDir(modelDir)
	if err != nil {
		return nil, fmt.Errorf("读取模型目录失败: %w", err)
	}
	var comments []modelCommentMeta
	comments, err = loadGeneratedModelComments(filepath.Join(modelDir, "table_comment.gen.go"))
	if err != nil {
		return nil, err
	}
	commentMap := make(map[string]string, len(comments))
	for _, comment := range comments {
		commentMap[comment.ModelName] = comment.TableComment
	}
	tables := make([]tableMeta, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".gen.go") || entry.Name() == "table_comment.gen.go" {
			continue
		}
		tableName := strings.TrimSuffix(entry.Name(), ".gen.go")
		queryFile := filepath.Join(queryDir, entry.Name())
		if _, err = os.Stat(queryFile); err != nil {
			return nil, fmt.Errorf("模型%s缺少对应查询文件: %w", tableName, err)
		}
		tables = append(tables, tableMeta{
			TableName:      tableName,
			ModelName:      buildModelName(tableName),
			TableComment:   commentMap[buildModelName(tableName)],
			RepositoryName: buildRepositoryName(tableName),
		})
	}
	if len(tables) == 0 {
		return nil, fmt.Errorf("生成目录中未发现模型文件")
	}
	sort.Slice(tables, func(i, j int) bool {
		return tables[i].TableName < tables[j].TableName
	})
	return tables, nil
}

// writeGeneratedQueryFile 根据当前目录中的全部表重建 query 聚合入口。
func writeGeneratedQueryFile(queryPath string, tables []tableMeta) error {
	queryDir, err := resolveGeneratedPath("query", queryPath)
	if err != nil {
		return err
	}
	return writeTemplateFile(filepath.Join(queryDir, "gen.go"), queryFileTemplate, struct {
		PackageName string
		Tables      []tableMeta
	}{
		PackageName: buildPackageName(queryDir),
		Tables:      tables,
	})
}

// generateTableModels 根据 table 配置生成单张表或当前数据库全部表的模型元数据。
func (g *Gen) generateTableModels(generator *gormgen.Generator) ([]interface{}, error) {
	if g.opts.table != "" {
		tableNames, err := parseTableNames(g.opts.table)
		if err != nil {
			return nil, err
		}
		tableModels := make([]interface{}, 0, len(tableNames))
		for _, tableName := range tableNames {
			// 指定多张表时先全部读取成功，再进入文件写入阶段。
			tableModel, generateErr := generateTableModel(tableName, func() interface{} {
				return generator.GenerateModel(tableName, g.buildModelOpts()...)
			})
			if generateErr != nil {
				return nil, generateErr
			}
			tableModels = append(tableModels, tableModel)
		}
		return tableModels, nil
	}
	tableModels := g.generateAllTable(generator)
	if len(tableModels) == 0 {
		return nil, fmt.Errorf("数据源%s未发现可生成的表", g.sourceName())
	}
	return tableModels, nil
}

// generateTableModel 将 gorm/gen 的单表生成 panic 转换为可由调用链处理的错误。
func generateTableModel(tableName string, generate func() interface{}) (tableModel interface{}, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("生成表%q失败: %v", tableName, recovered)
		}
	}()
	return generate(), nil
}

// GenerateAllTable 导出当前配置下数据库全部表的生成结果。
func (g *Gen) GenerateAllTable() ([]interface{}, error) {
	generator, err := g.newGenerator()
	if err != nil {
		return nil, err
	}
	tableModels := g.generateAllTable(generator)
	if len(tableModels) == 0 {
		return nil, fmt.Errorf("数据源%s未发现可生成的表", g.sourceName())
	}
	return tableModels, nil
}

// sourceName 返回错误和生成模板使用的数据源名称。
func (g *Gen) sourceName() string {
	if g.opts.sourceName == "" {
		return "default"
	}
	return g.opts.sourceName
}

// parseTableNames 解析逗号分隔的表名，并保留数据库原始标识符。
func parseTableNames(value string) ([]string, error) {
	names := make([]string, 0)
	for _, name := range strings.Split(value, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		names = append(names, name)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("table 参数不能为空")
	}
	return names, nil
}

// newGenerator 按当前配置初始化 gorm/gen 生成器。
func (g *Gen) newGenerator() (*gormgen.Generator, error) {
	opts := g.opts
	// 1. 根据 driver 名称加载 gorm dialector 构造器。
	gormDriver, ok := driver.Opens[opts.driver]
	if !ok {
		return nil, fmt.Errorf("gorm驱动加载失败【%s】", opts.driver)
	}

	// 2. 建立数据库连接。
	db, err := gorm.Open(gormDriver(opts.source), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// 3. 初始化生成器并写入基础配置。
	generator := gormgen.NewGenerator(gormgen.Config{
		OutPath:           opts.outPath,
		ModelPkgPath:      opts.modelPkgPath,
		FieldNullable:     false,
		FieldCoverable:    false,
		FieldSignable:     false,
		FieldWithIndexTag: true,
		FieldWithTypeTag:  true,
		WithUnitTest:      false,
	})
	generator.UseDB(db)
	// 使用固定的下划线转驼峰策略生成模型名。
	generator.WithModelNameStrategy(g.buildTableToModelNameStrategy())
	// 预注册软删除插件包路径；未用到该类型的模型文件会由 imports.Process 自动清理多余 import。
	generator.WithImportPkgPath(softDeleteImportPkgPath)
	return generator, nil
}

// buildTableToModelNameStrategy 构建“表名 -> 模型名”转换策略。
func (g *Gen) buildTableToModelNameStrategy() func(tableName string) string {
	return func(tableName string) string {
		return buildModelName(tableName)
	}
}

// generateAllTable 使用统一命名选项导出当前数据库全部表。
func (g *Gen) generateAllTable(generator *gormgen.Generator) []interface{} {
	return generator.GenerateAllTable(g.buildModelOpts()...)
}

// buildModelOpts 汇总模型生成选项，包含字段命名策略与软删除字段映射。
func (g *Gen) buildModelOpts() []gormgen.ModelOpt {
	return []gormgen.ModelOpt{
		g.buildFieldNameStrategy(),
		g.buildSoftDeleteStrategy(),
	}
}

// buildFieldNameStrategy 构建“字段列名 -> 模型字段名”转换策略。
func (g *Gen) buildFieldNameStrategy() gormgen.ModelOpt {
	return gormgen.FieldModify(func(field gormgen.Field) gormgen.Field {
		if field == nil || field.ColumnName == "" {
			return field
		}
		// 在 gorm/gen 调用 GORM SchemaName 前预先补齐扩展缩写，避免 SKU、LLM 等被降级为普通驼峰。
		field.Name = buildModelName(field.ColumnName)
		return field
	})
}

// buildSoftDeleteStrategy 将整数型 deleted_at 列映射为 soft_delete.DeletedAt，保留软删除语义并支持联合唯一索引复用编号。
func (g *Gen) buildSoftDeleteStrategy() gormgen.ModelOpt {
	return gormgen.FieldModify(func(field gormgen.Field) gormgen.Field {
		if field == nil || field.ColumnName != softDeleteColumnName {
			return field
		}
		// 仅处理 BIGINT 软删除列；INT 列放不下毫秒时间戳，datetime 等历史类型保持原状，避免生成类型与列类型不匹配。
		switch strings.TrimPrefix(field.Type, "*") {
		case "int64", "uint64":
			field.Type = softDeleteFieldType
			// softDelete:milli 表示删除时写入毫秒时间戳，0 表示未删除。
			field.GORMTag = field.GORMTag.Set("softDelete", "milli")
		}
		return field
	})
}
