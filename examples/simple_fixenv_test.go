package examples

import (
	"github.com/rekby/fixenv"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateDir_Fixenv(t *testing.T) {
	e := fixenv.New(t)
	fpath := filepath.Join(Folder(e), "dir")
	err := os.Mkdir(fpath, 0666)
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
}

func TestCreateFile_Fixenv(t *testing.T) {
	e := fixenv.New(t)
	fpath := filepath.Join(Folder(e), "file")
	f, err := os.Create(fpath)
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	_ = f.Close()
}

func Folder(e fixenv.Env) string {
	f := func() (*fixenv.GenericResult[string], error) {
		dir, err := os.MkdirTemp("", "")
		if err != nil {
			return nil, err
		}
		clean := func() {
			_ = os.RemoveAll(dir)
		}
		return fixenv.NewGenericResultWithCleanup(dir, clean), nil
	}

	return fixenv.CacheResult(e, f)
}
