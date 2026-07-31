package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type cleanupTarget struct {
	label string
	path  string
}

// CleanOutputPath 只清理输出根目录中的 query、data、repo 目录。
func CleanOutputPath(path string) error {
	dir, err := resolveGeneratedPath("output", path)
	if err != nil {
		return err
	}
	return cleanupTargets(
		cleanupTarget{label: "query", path: filepath.Join(dir, defaultOutPath)},
		cleanupTarget{label: "data", path: filepath.Join(dir, defaultDataPath)},
		cleanupTarget{label: "repo", path: filepath.Join(dir, defaultRepoPath)},
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

// cleanupGeneratedDirs 只清理生成器负责重建的 query、data、repo 目录。
func (g *Gen) cleanupGeneratedDirs() error {
	if g.opts.table != "" {
		// 单表模式必须保留其他表产物，只允许覆盖当前表对应文件和聚合入口。
		return nil
	}
	return cleanupTargets(
		cleanupTarget{label: "query", path: g.opts.outPath},
		cleanupTarget{label: "data", path: g.opts.dataPath},
		cleanupTarget{label: "repo", path: g.opts.repoPath},
	)
}

// collectCleanupDirs 汇总并去重需要清理的生成目录。
func collectCleanupDirs(targets ...cleanupTarget) ([]string, error) {
	dirs := make([]string, 0, len(targets))
	seen := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		dir, err := resolveGeneratedPath(target.label, target.path)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[dir]; ok {
			continue
		}
		seen[dir] = struct{}{}
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)
	return dirs, nil
}

// resolveGeneratedPath 解析生成目录为绝对路径，并校验必要参数。
func resolveGeneratedPath(label, path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("%s 输出目录不能为空", label)
	}
	return filepath.Abs(path)
}
