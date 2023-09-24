package examples

import (
	"errors"
	"testing"
)

func TestAddMoney(t *testing.T) {
	e := New(t)

	accounts := Accounts(e)
	requireNoErr(t, accounts.AddMoney(AccountID(e), 100))
	requireNoErr(t, accounts.AddMoney(AccountID(e), 25))

	money, err := accounts.GetMoney(AccountID(e))
	requireNoErr(t, err)
	requireEquals(t, 125, money)
}

func TestDebitMoney(t *testing.T) {
	e := New(t)

	accounts := Accounts(e)
	id := AccountID(e)
	requireErrorIs(t, accounts.DebitingMoney(id, 10), ErrNoMoney)
	requireNoErr(t, accounts.AddMoney(id, 100))
	requireNoErr(t, accounts.DebitingMoney(id, 10))

	money, err := accounts.GetMoney(id)
	requireNoErr(t, err)
	requireEquals(t, 90, money)
}

func TestTransferMoney(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		t.Parallel()
		e := New(t)
		accounts := Accounts(e)

		requireNoErr(t, accounts.AddMoney(NamedAccountID(e, "alice"), 100))
		requireNoErr(t,
			accounts.TransferMoney(
				NamedAccountID(e, "alice"),
				NamedAccountID(e, "bob"),
				10,
			),
		)

		money, err := accounts.GetMoney(NamedAccountID(e, "alice"))
		requireNoErr(t, err)
		requireEquals(t, 90, money)

		money, err = accounts.GetMoney(NamedAccountID(e, "bob"))
		requireNoErr(t, err)
		requireEquals(t, 10, money)
	})
	t.Run("NoMoney", func(t *testing.T) {
		t.Parallel()
		e := New(t)
		requireErrorIs(t,
			Accounts(e).TransferMoney(
				NamedAccountID(e, "alice"),
				NamedAccountID(e, "bob"),
				100,
			),
			ErrNoMoney,
		)
	})
}

func requireNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func requireErrorIs(t *testing.T, err, target error) {
	t.Helper()
	if !errors.Is(err, target) {
		t.Fatal(err)
	}
}

func requireEquals(t *testing.T, expected, actual int) {
	t.Helper()
	if expected != actual {
		t.Fatalf("%v != %v", expected, actual)
	}
}
