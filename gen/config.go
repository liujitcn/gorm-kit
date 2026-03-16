package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const defaultConfigFile = "config.yaml"

// yamlConfig 定义生成器的 YAML 配置结构，字段与 options 保持一致。
type yamlConfig struct {
	Driver       string            `yaml:"driver"`
	Source       string            `yaml:"source"`
	OutPath      string            `yaml:"out_path"`
	ModelPkgPath string            `yaml:"model_pkg_path"`
	DataPath     string            `yaml:"data_path"`
	Acronyms     map[string]string `yaml:"acronyms"`
}

// loadOptionsFromYAML 从 YAML 文件中加载生成器配置，并与默认选项合并。
func loadOptionsFromYAML(filePath string) (options, error) {
	opts := defaultOptions()
	cfg, err := loadYAMLConfig(filePath)
	if err != nil {
		return options{}, err
	}

	// 通过复用 Option 逻辑，确保 YAML 与代码配置行为一致。
	for _, opt := range buildOptionsFromYAML(cfg) {
		opt(&opts)
	}
	return opts, nil
}

// loadYAMLConfig 从 YAML 文件中解析原始配置结构。
func loadYAMLConfig(filePath string) (yamlConfig, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return yamlConfig{}, nil
		}
		return yamlConfig{}, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg yamlConfig
	if err = yaml.Unmarshal(content, &cfg); err != nil {
		return yamlConfig{}, fmt.Errorf("解析配置文件失败: %w", err)
	}
	return cfg, nil
}

// applyYAMLOverrides 将命令行覆盖项写回 YAML 配置结构。
func applyYAMLOverrides(cfg *yamlConfig, overrides map[string]string) error {
	for key, value := range overrides {
		switch key {
		case "driver":
			cfg.Driver = value
		case "source":
			cfg.Source = value
		case "out_path":
			cfg.OutPath = value
		case "model_pkg_path":
			cfg.ModelPkgPath = value
		case "data_path":
			cfg.DataPath = value
		default:
			if strings.HasPrefix(key, "acronyms.") {
				acronymKey := strings.TrimSpace(strings.TrimPrefix(key, "acronyms."))
				if acronymKey == "" {
					return fmt.Errorf("无效覆盖项: %s", key)
				}
				if cfg.Acronyms == nil {
					cfg.Acronyms = make(map[string]string)
				}
				cfg.Acronyms[acronymKey] = value
				continue
			}
			return fmt.Errorf("不支持的覆盖项: %s", key)
		}
	}
	return nil
}

// loadOptionsFromConfigFileAndOverrides 先加载 YAML，再应用命令行覆盖项。
func loadOptionsFromConfigFileAndOverrides(filePath string, overrides map[string]string) (options, error) {
	opts := defaultOptions()
	cfg, err := loadYAMLConfig(filePath)
	if err != nil {
		return options{}, err
	}
	if err = applyYAMLOverrides(&cfg, overrides); err != nil {
		return options{}, err
	}
	for _, opt := range buildOptionsFromYAML(cfg) {
		opt(&opts)
	}
	return opts, nil
}

// buildOptionsFromYAML 将 YAML 配置转换为 Option 列表。
func buildOptionsFromYAML(cfg yamlConfig) []Option {
	opts := make([]Option, 0, 6)
	if strings.TrimSpace(cfg.Driver) != "" {
		opts = append(opts, WithDriver(cfg.Driver))
	}
	if strings.TrimSpace(cfg.Source) != "" {
		opts = append(opts, WithSource(cfg.Source))
	}
	if strings.TrimSpace(cfg.OutPath) != "" {
		opts = append(opts, WithOutPath(cfg.OutPath))
	}
	if strings.TrimSpace(cfg.ModelPkgPath) != "" {
		opts = append(opts, WithModelPkgPath(cfg.ModelPkgPath))
	}
	if strings.TrimSpace(cfg.DataPath) != "" {
		opts = append(opts, WithDataPath(cfg.DataPath))
	}
	if len(cfg.Acronyms) > 0 {
		opts = append(opts, WithAcronyms(cfg.Acronyms))
	}
	return opts
}
