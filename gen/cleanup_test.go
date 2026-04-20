package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestCollectCleanupDirs 去重并返回绝对路径形式的生成目录列表。
func TestCollectCleanupDirs(t *testing.T) {
	tempDir := t.TempDir()
	dirs, err := collectCleanupDirs(
		cleanupTarget{label: "query", path: filepath.Join(tempDir, "query")},
		cleanupTarget{label: "model", path: filepath.Join(tempDir, "models")},
	)
	if err != nil {
		t.Fatalf("collectCleanupDirs() error = %v", err)
	}

	expected := []string{
		filepath.Join(tempDir, "models"),
		filepath.Join(tempDir, "query"),
	}
	if !reflect.DeepEqual(dirs, expected) {
		t.Fatalf("collectCleanupDirs() = %v, want %v", dirs, expected)
	}
}

// TestCollectCleanupDirsDedup 相同目录配置只应清理一次，避免重复处理。
func TestCollectCleanupDirsDedup(t *testing.T) {
	tempDir := t.TempDir()
	sharedDir := filepath.Join(tempDir, "generated")
	dirs, err := collectCleanupDirs(
		cleanupTarget{label: "query", path: sharedDir},
		cleanupTarget{label: "model", path: sharedDir},
	)
	if err != nil {
		t.Fatalf("collectCleanupDirs() error = %v", err)
	}
	if len(dirs) != 1 {
		t.Fatalf("collectCleanupDirs() len = %d, want 1", len(dirs))
	}
	if dirs[0] != sharedDir {
		t.Fatalf("collectCleanupDirs() first dir = %s, want %s", dirs[0], sharedDir)
	}
}

// TestCleanupGeneratedDirsRemovesLegacyFiles 重新生成前应删除旧表残留的历史文件。
func TestCleanupGeneratedDirsRemovesLegacyFiles(t *testing.T) {
	tempDir := t.TempDir()
	queryDir := filepath.Join(tempDir, "query")
	modelDir := filepath.Join(tempDir, "models")
	legacyFiles := []string{
		filepath.Join(queryDir, "old_table.gen.go"),
		filepath.Join(modelDir, "old_table.gen.go"),
	}
	for _, legacyFile := range legacyFiles {
		if err := os.MkdirAll(filepath.Dir(legacyFile), 0o755); err != nil {
			t.Fatalf("MkdirAll(%s) error = %v", legacyFile, err)
		}
		if err := os.WriteFile(legacyFile, []byte("legacy"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", legacyFile, err)
		}
	}

	g := &Gen{opts: options{
		outPath:      queryDir,
		modelPkgPath: modelDir,
	}}
	if err := g.cleanupGeneratedDirs(); err != nil {
		t.Fatalf("cleanupGeneratedDirs() error = %v", err)
	}

	for _, dir := range []string{queryDir, modelDir} {
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Fatalf("目录 %s 仍存在，want removed, err = %v", dir, err)
		}
	}
}

// TestCleanupTargetsRemovesDataDir 复用 cleanup 逻辑时也应支持清理 data 目录。
func TestCleanupTargetsRemovesDataDir(t *testing.T) {
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	legacyFile := filepath.Join(dataDir, "old_table.go")
	if err := os.MkdirAll(filepath.Dir(legacyFile), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", legacyFile, err)
	}
	if err := os.WriteFile(legacyFile, []byte("legacy"), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", legacyFile, err)
	}

	if err := cleanupTargets(cleanupTarget{label: "data", path: dataDir}); err != nil {
		t.Fatalf("cleanupTargets() error = %v", err)
	}

	if _, err := os.Stat(dataDir); !os.IsNotExist(err) {
		t.Fatalf("目录 %s 仍存在，want removed, err = %v", dataDir, err)
	}
}
