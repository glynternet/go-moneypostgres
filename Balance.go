package GOHMoneyDB

import (
	"database/sql"
	"github.com/GlynOwenHanmer/GOHMoney"
	"bytes"
	"fmt"
	"time"
	"errors"
)

const (
	balanceInsertFields string = "account_id, date, balance"
	balanceSelectFields string = "id, date, balance"
)

// Balance holds logic for an Account item that is held within a GOHMoney database.
type Balance struct {
	GOHMoney.Balance
	Id uint `json:"id"`
}

// Balanes holds multiple Balance items
type Balances []Balance

// Balances returns all Balances for a given Account and any errors that occur whilst attempting to retrieve the Balances.
func (account Account) Balances(db *sql.DB) (Balances, error) {
	return selectBalancesForAccount(db, account.Id)
}

// selectBalancesForAccount returns all Balance items, as a single Balances item, for a given account Id number in the given database, along with any errors that occur whilst attempting to retrieve the Balances.
func selectBalancesForAccount(db *sql.DB, accountId uint) (Balances, error) {
	var queryBuffer bytes.Buffer
	queryBuffer.WriteString("SELECT ")
	queryBuffer.WriteString(balanceSelectFields)
	queryBuffer.WriteString(" FROM balances WHERE account_id = ")
	queryBuffer.WriteString(fmt.Sprintf("%d", accountId))
	queryBuffer.WriteString(" ORDER BY date ASC, Id ASC")
	rows, err := db.Query(queryBuffer.String())
	if err != nil {
		return Balances{}, err
	}
	defer rows.Close()
	balances := Balances{}
	for rows.Next() {
		balance := Balance{}
		err := rows.Scan(&balance.Id, &balance.Date, &balance.Amount)
		if err != nil {
			return nil, err
		}
		balances = append(balances, balance)
	}
	return balances, rows.Err()
}

// InsertBalance adds a Balance entry to the given DB for the given account and returns the inserted Balance item with any errors that occured while attempting to insert the Balance.
func (account Account) InsertBalance(db *sql.DB, balance GOHMoney.Balance) (Balance, error) {
	err := account.Account.ValidateBalance(balance)
	if err != nil {
		return Balance{}, err
	}
	var query bytes.Buffer
	fmt.Fprintf(&query, `INSERT INTO balances (%s) VALUES ($1, $2, $3) `, balanceInsertFields)
	fmt.Fprintf(&query, `RETURNING %s;`, balanceSelectFields)
	row := db.QueryRow(query.String(), account.Id, balance.Date, balance.Amount)
	var insertedBalance Balance
	return insertedBalance, row.Scan(&insertedBalance.Id, &insertedBalance.Date, &insertedBalance.Amount)
}

// UpdateBalance updates a Balance entry in a given db for a given account and original Balance, returning any errors that are present with the validitiy of the Account, original Balance or update Balance.
func (account Account) UpdateBalance(db *sql.DB, original Balance, update GOHMoney.Balance) (Balance, error) {
	if err := account.ValidateBalance(db,original); err != nil {
		return Balance{}, err
	}
	if err := update.Validate(); err != nil {
		return Balance{}, errors.New(`Update Balance is not valid: ` + err.Error())
	}
	if err := account.Account.ValidateBalance(update); err != nil {
		return Balance{}, errors.New(`Update is not valid for account: ` + err.Error())
	}
	row := db.QueryRow(`UPDATE balances SET balance = $1, date = $2 WHERE id = $3 returning ` + balanceSelectFields, update.Amount, update.Date, original.Id)
	balance, err := scanRowForBalance(row)
	return *balance, err
}

// BalanceAtDate returns a Balance item representing the Balance of an account at the given time for the given account with the given DB.
func (account Account) BalanceAtDate(db *sql.DB, time time.Time) (Balance, error) {
	var query bytes.Buffer
	fmt.Fprintf(&query, `SELECT %s`, balanceSelectFields)
	fmt.Fprint(&query, ` FROM balances `)
	fmt.Fprintf(&query, `WHERE account_id = $1`)
	fmt.Fprintf(&query, ` AND date <= $2 `)
	fmt.Fprintf(&query, `ORDER BY date DESC, id DESC LIMIT 1;`, )
	row := db.QueryRow(query.String(), account.Id, time)
	balance, err := scanRowForBalance(row)
	return *balance, err
}

// scanRowForBalance scans a single sql.Row for a Balance object and returns any error occurring along the way.
func scanRowForBalance(row *sql.Row) (*Balance, error) {
	var balance Balance
	err := row.Scan(&balance.Id, &balance.Date, &balance.Amount)
	if err == sql.ErrNoRows {
		err = NoBalances
	}
	return &balance, err
}