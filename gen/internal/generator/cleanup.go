package generator

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type cleanupTarget struct {
	label string
	path  string
}

// CleanOutputPath 清理输出根目录中由生成器完整重建的目录。
func CleanOutputPath(path string) error {
	dir, err := resolveCleanupBasePath(path)
	if err != nil {
		return err
	}
	return cleanupTargets(
		cleanupTarget{label: "query", path: filepath.Join(dir, defaultOutPath)},
		cleanupTarget{label: "models", path: filepath.Join(dir, defaultModelPkgPath)},
		cleanupTarget{label: "data", path: filepath.Join(dir, defaultDataPath)},
	)
}

// cleanupTargets 按传入顺序清理生成目录，并自动跳过重复目录。
func cleanupTargets(targets ...cleanupTarget) error {
	dirs, err := collectCleanupDirs(targets...)
	if err != nil {
		return err
	}
	for _, dir := range dirs {
		if err = os.RemoveAll(dir); err != nil {
			return fmt.Errorf("清理目录%s失败: %w", dir, err)
		}
	}
	return nil
}

// cleanupGeneratedDirs 清理生成器负责完整重建的输出目录。
func (g *Gen) cleanupGeneratedDirs() error {
	if g.opts.table != "" {
		// 单表模式必须保留其他表产物，只允许覆盖当前表对应文件和聚合入口。
		return nil
	}
	dir, err := resolveCleanupBasePath(g.opts.basePath)
	if err != nil {
		return err
	}
	return cleanupTargets(
		cleanupTarget{label: "query", path: filepath.Join(dir, defaultOutPath)},
		cleanupTarget{label: "models", path: filepath.Join(dir, defaultModelPkgPath)},
		cleanupTarget{label: "data", path: filepath.Join(dir, defaultDataPath)},
	)
}

// collectCleanupDirs 汇总并去重需要清理的生成目录。
func collectCleanupDirs(targets ...cleanupTarget) ([]string, error) {
	dirs := make([]string, 0, len(targets))
	seen := make(map[string]struct{}, len(targets))
	var err error
	for _, target := range targets {
		var dir string
		dir, err = resolveGeneratedPath(target.label, target.path)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[dir]; ok {
			continue
		}
		seen[dir] = struct{}{}
		dirs = append(dirs, dir)
	}
	slices.Sort(dirs)
	return dirs, nil
}

// resolveGeneratedPath 解析生成目录为绝对路径，并校验必要参数。
func resolveGeneratedPath(label, path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("%s 输出目录不能为空", label)
	}
	return filepath.Abs(path)
}

// resolveCleanupBasePath 解析清理根目录并拒绝文件系统根目录。
func resolveCleanupBasePath(path string) (string, error) {
	basePath, err := resolveGeneratedPath("output", path)
	if err != nil {
		return "", err
	}
	var missingParts []string
	for {
		var resolvedPath string
		resolvedPath, err = filepath.EvalSymlinks(basePath)
		if err == nil {
			for i := len(missingParts) - 1; i >= 0; i-- {
				resolvedPath = filepath.Join(resolvedPath, missingParts[i])
			}
			basePath = filepath.Clean(resolvedPath)
			break
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("解析 output 输出目录%s失败: %w", path, err)
		}
		parentPath := filepath.Dir(basePath)
		if parentPath == basePath {
			break
		}
		missingParts = append(missingParts, filepath.Base(basePath))
		basePath = parentPath
	}
	rootPath := filepath.VolumeName(basePath) + string(filepath.Separator)
	if basePath == filepath.Clean(rootPath) {
		return "", fmt.Errorf("output 输出目录不能是文件系统根目录: %s", path)
	}
	return basePath, nil
}
