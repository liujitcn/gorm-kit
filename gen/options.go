package gen

import (
	"strings"
)

const (
	// 默认连接与输出参数，供 NewGen 未传 Option 时兜底使用。
	defaultSource       = "root:112233@tcp(127.0.0.1:3306)/shop?charset=utf8&parseTime=True&loc=Local&timeout=1000ms"
	defaultOutPath      = "query"
	defaultModelPkgPath = "models"
	defaultDriver       = "mysql"
)

type Option func(o *options)

// options 为生成器内部配置，统一由 Option 写入。
type options struct {
	driver       string
	source       string
	outPath      string
	modelPkgPath string
}

// WithDriver 设置数据库驱动。
func WithDriver(driver string) Option {
	return func(o *options) {
		// 仅在非空时覆盖，避免误清空默认值。
		if strings.TrimSpace(driver) != "" {
			o.driver = driver
		}
	}
}

// WithSource 设置数据库连接串。
func WithSource(source string) Option {
	return func(o *options) {
		// 仅在非空时覆盖，避免误清空默认值。
		if strings.TrimSpace(source) != "" {
			o.source = source
		}
	}
}

// WithOutPath 设置 query 输出目录。
func WithOutPath(path string) Option {
	return func(o *options) {
		// 仅在非空时覆盖，避免误清空默认值。
		if strings.TrimSpace(path) != "" {
			o.outPath = path
		}
	}
}

// WithModelPkgPath 设置 model 包路径。
func WithModelPkgPath(path string) Option {
	return func(o *options) {
		// 仅在非空时覆盖，避免误清空默认值。
		if strings.TrimSpace(path) != "" {
			o.modelPkgPath = path
		}
	}
}

// defaultOptions 提供最小可运行配置。
func defaultOptions() options {
	return options{
		driver:       defaultDriver,
		source:       defaultSource,
		outPath:      defaultOutPath,
		modelPkgPath: defaultModelPkgPath,
	}
}
