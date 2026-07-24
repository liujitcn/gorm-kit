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

// CleanOutputPath 清空生成输出目录中的全部内容并保留目录本身。
func CleanOutputPath(path string) error {
	dir, err := resolveGeneratedPath("output", path)
	if err != nil {
		return err
	}
	var entries []os.DirEntry
	entries, err = os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("读取输出目录%s失败: %w", dir, err)
	}
	for _, entry := range entries {
		entryPath := filepath.Join(dir, entry.Name())
		if err = os.RemoveAll(entryPath); err != nil {
			return fmt.Errorf("清理输出目录%s失败: %w", entryPath, err)
		}
	}
	return nil
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

// cleanupGeneratedDirs 清理 gorm/gen 产物目录，避免已删除表的历史文件残留。
func (g *Gen) cleanupGeneratedDirs() error {
	if g.opts.table != "" {
		// 单表模式必须保留其他表产物，只允许覆盖当前表对应文件和聚合入口。
		return nil
	}
	return cleanupTargets(
		cleanupTarget{label: "query", path: g.opts.outPath},
		cleanupTarget{label: "model", path: g.opts.modelPkgPath},
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
