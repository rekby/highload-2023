package examples

import (
	"context"
	"fmt"
	"github.com/rekby/fixenv"
	"github.com/rekby/fixenv/sf"
	"github.com/ydb-platform/ydb-go-sdk/v3"
	"github.com/ydb-platform/ydb-go-sdk/v3/table"
	"log"
	"math/rand"
	"strconv"
	"testing"
	"time"
)

type Env interface {
	fixenv.Env
}

func New(t *testing.T) *fixenv.EnvT {
	return fixenv.New(t)
}

func DB(e Env) *ydb.Driver {
	f := func() (*fixenv.GenericResult[*ydb.Driver], error) {
		connectionString := fmt.Sprintf("grpc://%s/local", YDBDocker(e))
		e.T().Logf("Connecting to %s", connectionString)
		db, err := ydb.Open(sf.Context(e), connectionString)
		clean := func() {
			if db == nil {
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			_ = db.Close(ctx)
			cancel()
		}

		return fixenv.NewGenericResultWithCleanup(db, clean), err
	}

	return fixenv.CacheResult(e, f, fixenv.CacheOptions{Scope: fixenv.ScopePackage})
}

func Accounts(e Env) *AccountsStorage {
	f := func() (*fixenv.GenericResult[*AccountsStorage], error) {
		db := DB(e)
		e.T().Logf("Creating table accounts...")
		err := db.Table().Do(sf.Context(e), func(ctx context.Context, s table.Session) error {
			return s.ExecuteSchemeQuery(ctx, `
CREATE TABLE accounts (id Text NOT NULL, balance Int64, PRIMARY KEY (id));
`)
		})
		if err != nil {
			return nil, err
		}

		clean := func() {
			log.Printf("Removing table accounts")
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			_ = db.Table().Do(ctx, func(ctx context.Context, s table.Session) error {
				return s.ExecuteSchemeQuery(ctx, "DROP TABLE accounts")
			})
			cancel()
		}

		storage := newAccountsStorage(db)
		return fixenv.NewGenericResultWithCleanup(storage, clean), nil
	}
	return fixenv.CacheResult(e, f, fixenv.CacheOptions{Scope: fixenv.ScopePackage})
}

func AccountID(e Env) string {
	return NamedAccountID(e, "default")
}

func NamedAccountID(e Env, name string) string {
	f := func() (*fixenv.GenericResult[string], error) {
		id := strconv.Itoa(rand.Int())
		e.T().Logf("Creating account %q: %v", name, id)
		err := Accounts(e).CreateAccount(sf.Context(e), id)
		if err != nil {
			return nil, err
		}

		clean := func() {
			e.T().Logf("Removing account %q: %v", name, id)
			err := Accounts(e).DropAccount(sf.Context(e), id)
			if err != nil {
				e.T().Fatalf("failed to remove account %q: %+v", err)
			}
		}
		return fixenv.NewGenericResultWithCleanup(id, clean), nil
	}

	return fixenv.CacheResult(e,
		f, fixenv.CacheOptions{
			CacheKey: name,
		},
	)
}
