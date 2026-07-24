package generator

import (
	"path/filepath"
	"strings"
)

const (
	// 默认连接与输出参数，供 NewGen 未传 Option 时兜底使用。
	defaultDriver       = "mysql"
	defaultOutPath      = "query"
	defaultModelPkgPath = "models"
	defaultDataPath     = "data"
)

// Option 配置生成器的一项可选参数。
type Option func(o *options)

// options 为生成器内部配置，统一由 Option 写入。
type options struct {
	driver       string
	source       string
	sourceName   string
	databaseKey  string
	table        string
	basePath     string
	outPath      string
	modelPkgPath string
	dataPath     string
}

// WithDriver 设置数据库驱动。
func WithDriver(driver string) Option {
	return func(o *options) {
		// 命令行配置仅用 TrimSpace 判定是否为空，保留原始输入值。
		if strings.TrimSpace(driver) != "" {
			o.driver = driver
		}
	}
}

// WithSource 设置数据库连接串。
func WithSource(source string) Option {
	return func(o *options) {
		// 命令行配置仅用 TrimSpace 判定是否为空，保留原始输入值。
		if strings.TrimSpace(source) != "" {
			o.source = source
		}
	}
}

// WithName 设置数据源名称，用于生成路径、日志和生成代码中的客户端标识。
func WithName(name string) Option {
	return func(o *options) {
		if name != "" {
			o.sourceName = name
		}
	}
}

// WithDatabaseKey 设置多数据库配置中的数据源 key，使生成客户端从 databases map 中取值。
func WithDatabaseKey(key string) Option {
	return func(o *options) {
		if strings.TrimSpace(key) != "" {
			o.databaseKey = key
		}
	}
}

// WithTable 设置需要生成的单张数据库表；未设置时生成当前数据库全部表。
func WithTable(table string) Option {
	return func(o *options) {
		// 表名按数据库原始标识符使用，仅用 TrimSpace 判定是否为空。
		if strings.TrimSpace(table) != "" {
			o.table = table
		}
	}
}

// WithBasePath 设置统一基础路径，用于批量拼接 models、query、data 输出目录。
func WithBasePath(path string) Option {
	return func(o *options) {
		trimmedPath := strings.TrimSpace(path)
		if trimmedPath != "" {
			o.basePath = trimmedPath
		}
	}
}

// WithOutPath 设置 query 输出目录。
func WithOutPath(path string) Option {
	return func(o *options) {
		// 命令行配置仅用 TrimSpace 判定是否为空，保留原始输入值。
		if strings.TrimSpace(path) != "" {
			o.outPath = path
		}
	}
}

// WithModelPkgPath 设置 model 包路径。
func WithModelPkgPath(path string) Option {
	return func(o *options) {
		// 命令行配置仅用 TrimSpace 判定是否为空，保留原始输入值。
		if strings.TrimSpace(path) != "" {
			o.modelPkgPath = path
		}
	}
}

// WithDataPath 设置 data 输出目录。
func WithDataPath(path string) Option {
	return func(o *options) {
		// 命令行配置仅用 TrimSpace 判定是否为空，保留原始输入值。
		if strings.TrimSpace(path) != "" {
			o.dataPath = path
		}
	}
}

// ApplyBasePath 将基础路径应用到默认输出目录，减少重复配置。
func (o *options) ApplyBasePath() {
	if strings.TrimSpace(o.basePath) == "" {
		return
	}
	// 统一以基础路径作为前缀，未单独覆盖时自动生成 base/models、base/query、base/data。
	o.outPath = filepath.Join(o.basePath, o.outPath)
	o.modelPkgPath = filepath.Join(o.basePath, o.modelPkgPath)
	o.dataPath = filepath.Join(o.basePath, o.dataPath)
}

// defaultOptions 提供最小可运行配置。
func defaultOptions() options {
	return options{
		driver:       defaultDriver,
		outPath:      defaultOutPath,
		modelPkgPath: defaultModelPkgPath,
		dataPath:     defaultDataPath,
	}
}
