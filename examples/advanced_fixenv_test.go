package examples

import (
	"github.com/rekby/fixenv"
	"os"
	"path/filepath"
	"testing"
)

func TestConsistentContext(t *testing.T) {
	e := fixenv.New(t)
	t.Log("first", Folder(e))
	t.Log("second", Folder(e)) // Same as first time
	t.Log("test finish")
}

func TestInbox(t *testing.T) {
	e := fixenv.New(t)
	t.Log("inbox", Inbox(e))   // /tmp/123/inbox
	t.Log("first", Folder(e))  // /tmp/123
	t.Log("second", Folder(e)) // /tmp/123
}

func Inbox(e fixenv.Env) string {
	return fixenv.Cache(e, nil, nil, func() (string, error) {
		path := filepath.Join(Folder(e), "inbox")
		e.T().Logf("Creating inbox directory: %v", path)
		return path, os.Mkdir(path, 0666)
	})
}
