// Package gen 提供 GORM 数据模型、查询和 data 代码生成能力。
package gen

import (
	"github.com/liujitcn/gorm-kit/gen/internal/config"
	"github.com/liujitcn/gorm-kit/gen/internal/generator"
)

// Gen 封装单数据源代码生成器。
type Gen = generator.Gen

// Option 配置单数据源生成器的一项可选参数。
type Option = generator.Option

// ConfigOptions 描述从服务 data.yaml 批量生成数据库代码的参数。
type ConfigOptions = config.ConfigOptions

// NewGen 创建单数据源生成器实例。
func NewGen(opts ...Option) *Gen {
	return generator.NewGen(opts...)
}

// WithDriver 设置数据库驱动。
func WithDriver(driverName string) Option {
	return generator.WithDriver(driverName)
}

// WithSource 设置数据库连接串。
func WithSource(source string) Option {
	return generator.WithSource(source)
}

// WithName 设置数据源名称。
func WithName(name string) Option {
	return generator.WithName(name)
}

// WithDatabaseKey 设置多数据库配置中的数据源 key。
func WithDatabaseKey(key string) Option {
	return generator.WithDatabaseKey(key)
}

// WithTable 设置需要生成的表名，支持逗号分隔的多表。
func WithTable(table string) Option {
	return generator.WithTable(table)
}

// WithBasePath 设置模型、查询和 data 输出目录的统一前缀。
func WithBasePath(path string) Option {
	return generator.WithBasePath(path)
}

// WithOutPath 设置 query 输出目录。
func WithOutPath(path string) Option {
	return generator.WithOutPath(path)
}

// WithModelPkgPath 设置 model 输出目录。
func WithModelPkgPath(path string) Option {
	return generator.WithModelPkgPath(path)
}

// WithDataPath 设置 data 输出目录。
func WithDataPath(path string) Option {
	return generator.WithDataPath(path)
}

// GenerateConfig 读取服务配置并生成一个或多个数据源的代码。
func GenerateConfig(opts ConfigOptions) error {
	return config.GenerateConfig(opts)
}
