package main

import (
	"strings"
)

const (
	// 默认连接与输出参数，供 NewGen 未传 Option 时兜底使用。
	defaultOutPath      = "query"
	defaultModelPkgPath = "models"
	defaultDataPath     = "data"
)

type Option func(o *options)

// options 为生成器内部配置，统一由 Option 写入。
type options struct {
	driver       string
	source       string
	outPath      string
	modelPkgPath string
	dataPath     string
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

// WithDataPath 设置 data 输出目录。
func WithDataPath(path string) Option {
	return func(o *options) {
		// 仅在非空时覆盖，避免误清空默认值。
		if strings.TrimSpace(path) != "" {
			o.dataPath = path
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

// defaultOptions 提供最小可运行配置。
func defaultOptions() options {
	return options{
		outPath:      defaultOutPath,
		modelPkgPath: defaultModelPkgPath,
		dataPath:     defaultDataPath,
	}
}
