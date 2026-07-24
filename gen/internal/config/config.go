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
	if opts.Table != "" && opts.DatabaseName == "" && (!hasLegacySource(sources) || len(sources) > 1) {
		return errors.New("使用 table 参数时必须指定 database")
	}
	if err = validateSourceDirectories(sources); err != nil {
		return err
	}
	if opts.Table == "" {
		cleanPath := basePath
		if opts.DatabaseName != "" && !sources[0].legacy {
			cleanPath = filepath.Join(basePath, sources[0].directory)
		}
		if err = generator.CleanOutputPath(cleanPath); err != nil {
			return err
		}
	}

	var generationErrors []error
	for _, source := range sources {
		generatedPath := basePath
		if !source.legacy {
			generatedPath = filepath.Join(basePath, source.directory)
		}
		options := []generator.Option{
			generator.WithDriver(source.driver),
			generator.WithSource(source.source),
			generator.WithName(source.name),
			generator.WithBasePath(generatedPath),
		}
		if !source.legacy {
			options = append(options, generator.WithDatabaseKey(source.name))
		}
		if opts.Table != "" {
			options = append(options, generator.WithTable(opts.Table))
		}
		if _, generateErr := generator.NewGen(options...).Generate(); generateErr != nil {
			generationErrors = append(generationErrors, fmt.Errorf("数据源%s生成失败: %w", source.name, generateErr))
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

	if selectedName != "" {
		source, exists := sources[selectedName]
		if !exists {
			return nil, fmt.Errorf("数据库数据源不存在: %s", selectedName)
		}
		source.directory, err = normalizeSourceDirectory(source.name)
		if err != nil {
			return nil, err
		}
		return []configSource{source}, nil
	}
	result := make([]configSource, 0, len(sources))
	for _, source := range sources {
		source.directory, err = normalizeSourceDirectory(source.name)
		if err != nil {
			return nil, err
		}
		result = append(result, source)
	}
	return result, nil
}

// hasLegacySource 判断当前生成目标是否包含旧的单数据库配置。
func hasLegacySource(sources []configSource) bool {
	for _, source := range sources {
		if source.legacy {
			return true
		}
	}
	return false
}

// validateSourceDirectories 校验数据源目录规范化后的唯一性。
func validateSourceDirectories(sources []configSource) error {
	directories := make(map[string]string, len(sources))
	for _, source := range sources {
		if source.legacy {
			continue
		}
		if previous, exists := directories[source.directory]; exists {
			return fmt.Errorf("数据源名称规范化后冲突: %s 与 %s", previous, source.name)
		}
		directories[source.directory] = source.name
	}
	return nil
}

// normalizeSourceDirectory 将数据源名称转为全小写、去连接符的生成目录名。
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
