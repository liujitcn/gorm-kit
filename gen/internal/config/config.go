package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/liujitcn/gorm-kit/gen/internal/generator"
	"gopkg.in/yaml.v3"
)

const (
	defaultConfigPath = "./configs/data.yaml"
	defaultBasePath   = "gen"
)

// ConfigOptions 描述从服务 data.yaml 批量生成数据库代码的参数。
type ConfigOptions struct {
	ConfigPath   string
	DatabaseName string
	Table        string
	BasePath     string
}

type generatorConfig struct {
	Data generatorDataConfig `yaml:"data"`
}

type generatorDataConfig struct {
	Database  *generatorDatabaseConfig            `yaml:"database"`
	Databases map[string]*generatorDatabaseConfig `yaml:"databases"`
}

type generatorDatabaseConfig struct {
	Driver string `yaml:"driver"`
	Source string `yaml:"source"`
}

type configSource struct {
	name      string
	driver    string
	source    string
	legacy    bool
	directory string
}

// GenerateConfig 读取服务配置并生成一个或多个数据源的代码。
func GenerateConfig(opts ConfigOptions) error {
	configPath := opts.ConfigPath
	if configPath == "" {
		configPath = defaultConfigPath
	}
	basePath := opts.BasePath
	if basePath == "" {
		basePath = defaultBasePath
	}
	sources, err := loadConfigSources(configPath, opts.DatabaseName)
	if err != nil {
		return err
	}
	var generationErrors []error
	for _, source := range sources {
		generatedPath := basePath
		if source.name != "default" {
			generatedPath = filepath.Join(basePath, source.directory)
		}
		if opts.Table == "" {
			if err = generator.CleanOutputPath(generatedPath); err != nil {
				return err
			}
		}
		_, err = generator.NewGen(generator.Config{
			Driver:      source.driver,
			Source:      source.source,
			SourceName:  source.name,
			NamedSource: !source.legacy,
			Table:       opts.Table,
			BasePath:    generatedPath,
		}).Generate()
		if err != nil {
			generationErrors = append(generationErrors, fmt.Errorf("数据源%s生成失败: %w", source.name, err))
		}
	}
	return errors.Join(generationErrors...)
}

// loadConfigSources 读取配置并按约定解析需要生成的目标数据源。
func loadConfigSources(filename string, selectedName string) ([]configSource, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件%s失败: %w", filename, err)
	}
	var fileConfig generatorConfig
	if err = yaml.Unmarshal(content, &fileConfig); err != nil {
		return nil, fmt.Errorf("解析配置文件%s失败: %w", filename, err)
	}
	sources := make(map[string]configSource, len(fileConfig.Data.Databases)+1)
	for name, database := range fileConfig.Data.Databases {
		if name == "" {
			return nil, errors.New("数据库名称不能为空")
		}
		if database == nil {
			return nil, fmt.Errorf("数据库配置不能为空: %s", name)
		}
		sources[name] = configSource{
			name:   name,
			driver: database.Driver,
			source: database.Source,
		}
	}
	if fileConfig.Data.Database != nil {
		if _, exists := sources["default"]; exists {
			return nil, errors.New("database config conflict: default")
		}
		sources["default"] = configSource{
			name:   "default",
			driver: fileConfig.Data.Database.Driver,
			source: fileConfig.Data.Database.Source,
			legacy: true,
		}
	}
	if len(sources) == 0 {
		return nil, errors.New("未配置任何数据库数据源")
	}

	if selectedName == "" {
		selectedName = "default"
	}
	source, exists := sources[selectedName]
	if !exists {
		// 传入的数据源名称不存在时回退到默认数据源，兼容不同环境下可选的数据源配置。
		source, exists = sources["default"]
		if !exists {
			if selectedName == "default" {
				return nil, errors.New("未配置默认数据库数据源，请通过 database 参数指定数据源")
			}
			return nil, fmt.Errorf("数据库数据源不存在: %s", selectedName)
		}
		// 连接配置借用 default，但生成目录和运行时查找名称仍使用传入的数据源名称。
		source.name = selectedName
		source.legacy = false
	}
	source.directory, err = normalizeSourceDirectory(source.name)
	if err != nil {
		return nil, err
	}
	return []configSource{source}, nil
}

// normalizeSourceDirectory 将数据源名称转换为安全且稳定的输出目录名。
func normalizeSourceDirectory(name string) (string, error) {
	var builder strings.Builder
	for _, character := range strings.ToLower(name) {
		if unicode.IsLetter(character) || unicode.IsDigit(character) {
			builder.WriteRune(character)
		}
	}
	if builder.Len() == 0 {
		return "", fmt.Errorf("数据源名称无法生成目录: %s", name)
	}
	return builder.String(), nil
}
