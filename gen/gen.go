package main

import (
	"fmt"

	"github.com/liujitcn/kratos-kit/database/gorm/driver"
	gormgen "gorm.io/gen"
	"gorm.io/gorm"
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
	return &Gen{opts: o}
}

// Execute 执行代码生成。
func (g *Gen) Execute() error {
	_, err := g.Generate()
	return err
}

// Generate 执行代码生成并返回当前数据库的全部表结果。
func (g *Gen) Generate() ([]interface{}, error) {
	generator, err := g.newGenerator()
	if err != nil {
		return nil, err
	}

	tableModels := g.generateAllTable(generator)
	// 只有成功读取到当前库表结构后才清理旧目录，避免连接异常时误删上次生成结果。
	if err = g.cleanupGeneratedDirs(); err != nil {
		return nil, err
	}

	// 4. 基于当前库全部表生成模型与查询代码。
	generator.ApplyBasic(tableModels...)
	generator.Execute()
	return tableModels, nil
}

// GenerateAllTable 导出当前配置下数据库全部表的生成结果。
func (g *Gen) GenerateAllTable() ([]interface{}, error) {
	generator, err := g.newGenerator()
	if err != nil {
		return nil, err
	}
	return g.generateAllTable(generator), nil
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
	return generator.GenerateAllTable(g.buildFieldNameStrategy())
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
