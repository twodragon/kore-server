package database

import (
	"database/sql"
	"fmt"

	null "gopkg.in/guregu/null.v3"
)

var (
	CheckIns = make(map[int]*CheckIn)
)

type CheckIn struct {
	CharID      int       `db:"charid" `
	TotalChecks int       `db:"total" `
	LastCheckIn null.Time `db:"last_check"`
	LastClaimed int       `db:"last_claimed" `
}

func (c *CheckIn) Update() error {
	_, err := pgsql_DbMap.Update(c)
	return err
}

func (c *CheckIn) Create() error {
	return pgsql_DbMap.Insert(c)
}

func GetCheckIns() error {
	var checks []*CheckIn
	query := `select * from hops.checkin`

	if _, err := pgsql_DbMap.Select(&checks, query); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("GetCheckIns: %s", err.Error())
	}

	for _, cr := range checks {
		CheckIns[cr.CharID] = cr
	}
	return nil
}
