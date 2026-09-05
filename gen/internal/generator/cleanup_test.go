package generator

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCleanOutputPath 验证全量生成前会清理全部生成目录。
func TestCleanOutputPath(t *testing.T) {
	directory := t.TempDir()
	var err error
	for _, name := range []string{defaultOutPath, defaultModelPkgPath, defaultDataPath} {
		path := filepath.Join(directory, name)
		err = os.MkdirAll(path, 0o755)
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(filepath.Join(path, "stale.go"), []byte("package stale"), 0o600)
		if err != nil {
			t.Fatal(err)
		}
	}
	err = CleanOutputPath(directory)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{defaultOutPath, defaultModelPkgPath, defaultDataPath} {
		_, err = os.Stat(filepath.Join(directory, name))
		if !os.IsNotExist(err) {
			t.Fatalf("生成目录未清理: %s", name)
		}
	}
}

// TestCleanupGeneratedDirsPreservesSingleTableOutputs 验证单表生成不会清理其他表产物。
func TestCleanupGeneratedDirsPreservesSingleTableOutputs(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, defaultModelPkgPath)
	err := os.MkdirAll(path, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	stalePath := filepath.Join(path, "existing.go")
	err = os.WriteFile(stalePath, []byte("package models"), 0o600)
	if err != nil {
		t.Fatal(err)
	}
	generator := &Gen{opts: buildOptions(Config{BasePath: directory, Table: "base_user"})}
	err = generator.cleanupGeneratedDirs()
	if err != nil {
		t.Fatal(err)
	}
	_, err = os.Stat(stalePath)
	if err != nil {
		t.Fatalf("单表生成不应清理现有模型: %v", err)
	}
}
