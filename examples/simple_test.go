package examples

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateDir_Go(t *testing.T) {
	dir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(dir)
	}()

	fpath := filepath.Join(dir, "tmp")
	err = os.Mkdir(fpath, 0666)
	if err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
}

func TestCreateFile_Go(t *testing.T) {
	dir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(dir)
	}()

	fpath := filepath.Join(dir, "tmp")
	f, err := os.Create(fpath)
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	_ = f.Close()
}
