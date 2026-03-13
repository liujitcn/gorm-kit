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
	acronyms     map[string]string
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

// WithOutputPath 设置 query 输出目录。
func WithOutputPath(path string) Option {
	return func(o *options) {
		// 仅在非空时覆盖，避免误清空默认值。
		if strings.TrimSpace(path) != "" {
			o.outPath = path
		}
	}
}

// WithModelPackagePath 设置 model 包路径。
func WithModelPackagePath(path string) Option {
	return func(o *options) {
		// 仅在非空时覆盖，避免误清空默认值。
		if strings.TrimSpace(path) != "" {
			o.modelPkgPath = path
		}
	}
}

// WithAcronym 追加单个缩写映射（key 不区分大小写）。
func WithAcronym(key, value string) Option {
	return func(o *options) {
		k := strings.ToLower(strings.TrimSpace(key))
		v := strings.TrimSpace(value)
		if k == "" || v == "" {
			return
		}
		if o.acronyms == nil {
			o.acronyms = make(map[string]string)
		}
		o.acronyms[k] = v
	}
}

// WithAcronyms 批量追加缩写映射（key 不区分大小写）。
func WithAcronyms(m map[string]string) Option {
	return func(o *options) {
		if len(m) == 0 {
			return
		}
		if o.acronyms == nil {
			o.acronyms = make(map[string]string, len(m))
		}
		for key, value := range m {
			k := strings.ToLower(strings.TrimSpace(key))
			v := strings.TrimSpace(value)
			if k == "" || v == "" {
				continue
			}
			o.acronyms[k] = v
		}
	}
}

// WithOutPath 兼容旧命名，内部转发到 WithOutputPath。
func WithOutPath(path string) Option {
	return WithOutputPath(path)
}

// WithModelPkgPath 兼容旧命名，内部转发到 WithModelPackagePath。
func WithModelPkgPath(path string) Option {
	return WithModelPackagePath(path)
}

// defaultOptions 提供最小可运行配置。
func defaultOptions() options {
	return options{
		driver:       defaultDriver,
		source:       defaultSource,
		outPath:      defaultOutPath,
		modelPkgPath: defaultModelPkgPath,
		// 默认缩写在 options 初始化时直接注入，后续仅做增量覆盖。
		acronyms: map[string]string{
			"api":   "API",
			"id":    "ID",
			"ip":    "IP",
			"url":   "URL",
			"uri":   "URI",
			"http":  "HTTP",
			"https": "HTTPS",
			"tcp":   "TCP",
			"udp":   "UDP",
			"rpc":   "RPC",
			"sql":   "SQL",
			"db":    "DB",
			"uid":   "UID",
			"uuid":  "UUID",
			"sku":   "SKU",
			"sn":    "SN",
		},
	}
}
