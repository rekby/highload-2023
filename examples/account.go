package examples

import (
	"database/sql"
	"errors"
)

var (
	ErrNoMoney = errors.New("no money on account")
)

type AccountsStorage struct {
	db        *sql.DB
	tableName string
}

func (s *AccountsStorage) CreateAccount(id string) error {
	panic("not implemented yet")
}

func (s *AccountsStorage) DropAccount(id string) error {
	panic("not implemented yet")
}

func (s *AccountsStorage) AddMoney(id string, money int) error {
	panic("not implemented yet")
}

func (s *AccountsStorage) DebitingMoney(id string, money int) error {
	panic("not implemented yet")
}

func (s *AccountsStorage) TransferMoney(from, to string, money int) error {
	panic("not implemented yet")
}

func (s *AccountsStorage) GetMoney(id string) (int, error) {
	panic("not implemented yet")
}
