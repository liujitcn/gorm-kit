package generator

import (
	"path/filepath"
)

const (
	// 默认连接与输出参数，供生成器配置缺省时使用。
	defaultDriver       = "mysql"
	defaultSourceName   = "default"
	defaultOutPath      = "query"
	defaultModelPkgPath = "models"
	defaultDataPath     = "data"
	defaultRepoPath     = "repo"
)

// Config 描述单数据源生成器的内部配置。
type Config struct {
	Driver      string
	Source      string
	SourceName  string
	NamedSource bool
	Table       string
	BasePath    string
}

// options 保存生成器使用的连接信息和派生输出路径。
type options struct {
	driver       string
	source       string
	sourceName   string
	namedSource  bool
	table        string
	outPath      string
	modelPkgPath string
	dataPath     string
	repoPath     string
}

// buildOptions 补齐默认值并派生固定的模型、查询和 data 输出目录。
func buildOptions(config Config) options {
	if config.Driver == "" {
		config.Driver = defaultDriver
	}
	if config.SourceName == "" {
		config.SourceName = defaultSourceName
	}
	return options{
		driver:       config.Driver,
		source:       config.Source,
		sourceName:   config.SourceName,
		namedSource:  config.NamedSource,
		table:        config.Table,
		outPath:      filepath.Join(config.BasePath, defaultOutPath),
		modelPkgPath: filepath.Join(config.BasePath, defaultModelPkgPath),
		dataPath:     filepath.Join(config.BasePath, defaultDataPath),
		repoPath:     filepath.Join(config.BasePath, defaultRepoPath),
	}
}
