package examples

import (
	"github.com/rekby/fixenv"
	"github.com/ydb-platform/ydb-go-sdk/v3"
	"testing"
)

type Env interface {
	fixenv.Env
}

func New(t *testing.T) *fixenv.EnvT {
	return fixenv.New(t)
}

func DB(e Env) *ydb.Driver {
	panic("not implemented yet")
}

func Accounts(e Env) *AccountsStorage {
	panic("not implemented yet")
}

func AccountID(e Env) string {
	panic("not implemented yet")
}

func NamedAccountID(e Env, name string) string {
	panic("not implemented yet")
}
