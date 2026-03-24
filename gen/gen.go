package main

import (
	"fmt"
	"strings"
	"unicode"

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
	generator, err := g.newGenerator()
	if err != nil {
		return err
	}

	// 4. 基于当前库全部表生成模型与查询代码。
	generator.ApplyBasic(generator.GenerateAllTable()...)
	generator.Execute()
	return nil
}

// GenerateAllTable 导出当前配置下数据库全部表的生成结果。
func (g *Gen) GenerateAllTable() ([]interface{}, error) {
	generator, err := g.newGenerator()
	if err != nil {
		return nil, err
	}
	return generator.GenerateAllTable(), nil
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
		parts := strings.Split(tableName, "_")
		for i, part := range parts {
			parts[i] = upperFirst(strings.ToLower(part))
		}
		return strings.Join(parts, "")
	}
}

func upperFirst(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
