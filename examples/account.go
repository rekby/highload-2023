package examples

import (
	"context"
	"errors"
	"fmt"
	"github.com/ydb-platform/ydb-go-sdk/v3"
	"github.com/ydb-platform/ydb-go-sdk/v3/table"
	"github.com/ydb-platform/ydb-go-sdk/v3/table/types"
	"log"
)

var (
	ErrNoMoney = errors.New("no money on account")
)

type AccountsStorage struct {
	db *ydb.Driver
}

func newAccountsStorage(db *ydb.Driver) *AccountsStorage {
	return &AccountsStorage{db}
}

func (store *AccountsStorage) CreateAccount(ctx context.Context, id string) error {
	return store.db.Table().Do(ctx, func(ctx context.Context, s table.Session) error {
		query := `
DECLARE $id AS Text;

INSERT INTO accounts (id, balance) VALUES ($id, 0) 
`
		_, _, err := s.Execute(ctx, table.DefaultTxControl(), query, table.NewQueryParameters(
			table.ValueParam("$id", types.TextValue(id)),
		))
		return err
	})
}

func (store *AccountsStorage) DropAccount(ctx context.Context, id string) error {
	return store.db.Table().Do(ctx, func(ctx context.Context, s table.Session) error {
		query := `
DECLARE $id AS Text;

SELECT COUNT(*) AS cnt
FROM accounts
WHERE id=$id;

DELETE FROM accounts
WHERE id=$id;
`
		_, res, err := s.Execute(ctx, table.DefaultTxControl(), query, table.NewQueryParameters(
			table.ValueParam("$id", types.TextValue(id)),
		))
		if err != nil {
			return fmt.Errorf("failed to check account: %w", err)
		}

		res.NextResultSet(ctx, "cnt")
		res.NextRow()

		var cnt uint64
		err = res.ScanWithDefaults(&cnt)
		if err != nil {
			return fmt.Errorf("failed to scan account count: %w", err)
		}

		if cnt == 0 {
			return fmt.Errorf("has no account id %q", id)
		}
		return nil
	})
}

func (store *AccountsStorage) AddMoney(ctx context.Context, id string, money int64) error {
	return store.db.Table().Do(ctx, func(ctx context.Context, s table.Session) error {
		query := `
DECLARE $id AS Text;
DECLARE $money AS Int64;

SELECT COUNT(*) AS cnt
FROM accounts
WHERE id=$id;

UPDATE accounts
SET balance = balance + $money
WHERE id=$id
;
`
		_, res, err := s.Execute(ctx, table.DefaultTxControl(), query, table.NewQueryParameters(
			table.ValueParam("$id", types.TextValue(id)),
			table.ValueParam("$money", types.Int64Value(money)),
		))
		if err != nil {
			return fmt.Errorf("failed to update account: %w", err)
		}

		res.NextResultSet(ctx, "cnt")
		res.NextRow()

		var cnt uint64
		err = res.ScanWithDefaults(&cnt)
		if err != nil {
			return fmt.Errorf("failed to scan accounts count: %w", err)
		}

		if cnt == 0 {
			return fmt.Errorf("account does not exists: %q", id)
		}
		return nil
	})
}

func (store *AccountsStorage) DebitingMoney(ctx context.Context, id string, money int64) error {
	return store.db.Table().DoTx(ctx, func(ctx context.Context, tx table.TransactionActor) error {
		query := `
DECLARE $id AS Text;

SELECT balance FROM accounts WHERE id=$id;
`
		res, err := tx.Execute(ctx, query, table.NewQueryParameters(
			table.ValueParam("$id", types.TextValue(id)),
		))
		if err != nil {
			return fmt.Errorf("failed to get account balance %q: %w", id, err)
		}

		if !res.NextResultSet(ctx, "balance") {
			return fmt.Errorf("has no result set: %w", err)
		}
		if !res.NextRow() {
			return fmt.Errorf("account does not exists: %q", id)
		}

		var balance int64
		err = res.ScanWithDefaults(&balance)
		if err != nil {
			return fmt.Errorf("failed to scan balance: %w", err)
		}

		balance -= money
		if balance < 0 {
			return fmt.Errorf("faile to debit money: %w", ErrNoMoney)
		}

		query = `
DECLARE $id AS Text;
DECLARE $balance AS Int64;

UPSERT INTO accounts (id, balance) VALUES ($id, $balance);
`
		_, err = tx.Execute(ctx, query, table.NewQueryParameters(
			table.ValueParam("$id", types.TextValue(id)),
			table.ValueParam("$balance", types.Int64Value(balance)),
		))

		if err != nil {
			return fmt.Errorf("failed to upsert new balance: %w", err)
		}
		return nil
	})
}

func (store *AccountsStorage) TransferMoney(ctx context.Context, from, to string, money int64) error {
	return store.db.Table().DoTx(ctx, func(ctx context.Context, tx table.TransactionActor) error {
		query := `
DECLARE $from AS Text;
DECLARE $to AS Text;

SELECT balance 
FROM accounts
WHERE id=$from;

SELECT balance 
FROM accounts
WHERE id=$to;
`
		res, err := tx.Execute(ctx, query, table.NewQueryParameters(
			table.ValueParam("$from", types.TextValue(from)),
			table.ValueParam("$to", types.TextValue(to)),
		))
		if err != nil {
			return fmt.Errorf("failed to read account balance")
		}

		var fromBalance, toBalance int64
		if !res.NextResultSet(ctx, "balance") {
			return fmt.Errorf("has no from result set")
		}
		if !res.NextRow() {
			return fmt.Errorf("has no from account %q", from)
		}
		if err = res.ScanWithDefaults(&fromBalance); err != nil {
			return fmt.Errorf("failed to scan from account: %w", err)
		}

		if !res.NextResultSet(ctx, "balance") {
			return fmt.Errorf("has no to result set")
		}
		if !res.NextRow() {
			return fmt.Errorf("has no to account")
		}
		if err = res.ScanWithDefaults(&toBalance); err != nil {
			return fmt.Errorf("failed to scan to balance: %w", err)
		}

		log.Printf("rekby: transfer from balance for account %q: %v", from, fromBalance)

		if fromBalance < money {
			return fmt.Errorf("failed to transer money: %w", ErrNoMoney)
		}

		query = `
DECLARE $updatedList AS List<Struct<
	id: Text,
	balance: Int64,
>>;

UPSERT INTO accounts
SELECT * FROM AS_TABLE($updatedList)
`
		updatedList := types.ListValue(
			types.StructValue(
				types.StructFieldValue("id", types.TextValue(from)),
				types.StructFieldValue("balance", types.Int64Value(fromBalance-money)),
			),
			types.StructValue(
				types.StructFieldValue("id", types.TextValue(to)),
				types.StructFieldValue("balance", types.Int64Value(toBalance+money)),
			),
		)
		_, err = tx.Execute(ctx, query, table.NewQueryParameters(
			table.ValueParam("$updatedList", updatedList),
		))
		if err != nil {
			return fmt.Errorf("failed to update balance: %w", err)
		}

		return nil
	})
}

func (store *AccountsStorage) GetMoney(ctx context.Context, id string) (int64, error) {
	var money int64
	err := store.db.Table().Do(ctx, func(ctx context.Context, s table.Session) error {
		query := `
DECLARE $id AS Text;

SELECT balance FROM accounts WHERE id=$id
`
		_, res, err := s.Execute(ctx, table.DefaultTxControl(), query, table.NewQueryParameters(
			table.ValueParam("$id", types.TextValue(id)),
		))
		if err != nil {
			return fmt.Errorf("failed to get account balance %q: %w", id, err)
		}

		if !res.NextResultSet(ctx, "balance") {
			return fmt.Errorf("has no result set: %w", res.Err())
		}
		res.NextRow()

		err = res.ScanWithDefaults(&money)
		if err != nil {
			return fmt.Errorf("failed to scan balance: %w", err)
		}

		return nil
	})
	return money, err
}
