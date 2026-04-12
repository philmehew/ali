package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/philmehew/ali/internal/models"
)

func TestPath_EnvOverride(t *testing.T) {
	custom := "/tmp/ali-test/functions.yaml"
	t.Setenv("ALI_CONFIG", custom)

	path, err := Path()
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	if path != custom {
		t.Errorf("expected %q, got %q", custom, path)
	}
}

func TestPath_Default(t *testing.T) {
	t.Setenv("ALI_CONFIG", "")

	dir, _ := os.UserConfigDir()
	expected := filepath.Join(dir, "ali", "functions.yaml")

	path, err := Path()
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}
}

func TestLoad_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("ALI_CONFIG", filepath.Join(tmpDir, "nonexistent.yaml"))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Functions) != 0 {
		t.Errorf("expected empty functions, got %d", len(cfg.Functions))
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("ALI_CONFIG", filepath.Join(tmpDir, "functions.yaml"))

	original := &models.AliConfig{
		Functions: []models.AliFunction{
			{
				Name:        "glog",
				Description: "Pretty git log",
				Body:        "git log --oneline -n $1",
				Defaults:    []string{"10"},
			},
		},
	}

	if err := Save(original); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded.Functions) != 1 {
		t.Fatalf("expected 1 function, got %d", len(loaded.Functions))
	}
	if loaded.Functions[0].Name != "glog" {
		t.Errorf("expected name %q, got %q", "glog", loaded.Functions[0].Name)
	}
	if loaded.Functions[0].Body != "git log --oneline -n $1" {
		t.Errorf("expected body %q, got %q", "git log --oneline -n $1", loaded.Functions[0].Body)
	}
}

func TestFindFunction(t *testing.T) {
	cfg := &models.AliConfig{
		Functions: []models.AliFunction{
			{Name: "glog", Body: "git log"},
			{Name: "mygrep", Body: "grep -r"},
		},
	}

	fn := FindFunction(cfg, "glog")
	if fn == nil {
		t.Fatal("expected to find glog, got nil")
	}
	if fn.Name != "glog" {
		t.Errorf("expected name %q, got %q", "glog", fn.Name)
	}

	fn = FindFunction(cfg, "missing")
	if fn != nil {
		t.Errorf("expected nil for missing function, got %v", fn)
	}
}

func TestFindFunctionIndex(t *testing.T) {
	cfg := &models.AliConfig{
		Functions: []models.AliFunction{
			{Name: "glog"},
			{Name: "mygrep"},
		},
	}

	if idx := FindFunctionIndex(cfg, "glog"); idx != 0 {
		t.Errorf("expected index 0, got %d", idx)
	}
	if idx := FindFunctionIndex(cfg, "mygrep"); idx != 1 {
		t.Errorf("expected index 1, got %d", idx)
	}
	if idx := FindFunctionIndex(cfg, "missing"); idx != -1 {
		t.Errorf("expected -1, got %d", idx)
	}
}

func TestSave_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("ALI_CONFIG", filepath.Join(tmpDir, "nested", "dir", "functions.yaml"))

	cfg := &models.AliConfig{Functions: []models.AliFunction{}}
	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "nested", "dir", "functions.yaml")); os.IsNotExist(err) {
		t.Error("expected config file to be created in nested directory")
	}
}
