package main

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
