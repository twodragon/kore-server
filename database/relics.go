package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	gorp "gopkg.in/gorp.v1"
)

var (
	Relics = make(map[int]*Relic)
)

type Relic struct {
	ID            int    `db:"id"`
	RequiredItems string `db:"required_items"`

	requiredItems []int64
}

func (e *Relic) Create() error {
	return pgsql_DbMap.Insert(e)
}

func (e *Relic) CreateWithTransaction(tr *gorp.Transaction) error {
	return tr.Insert(e)
}

func (e *Relic) Delete() error {
	_, err := pgsql_DbMap.Delete(e)
	return err
}

func GetRelics() error {
	var relics []*Relic
	query := `select * from data.relics`

	if _, err := pgsql_DbMap.Select(&relics, query); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("GetRelics: %s", err.Error())
	}

	for _, r := range relics {
		Relics[r.ID] = r
		r.GetReqItemsList()
	}

	return nil
}

func (e *Relic) GetReqItemsList() ([]int64, error) {

	if len(e.requiredItems) > 0 {
		return e.requiredItems, nil
	}

	data := fmt.Sprintf("[%s]", strings.Trim(e.RequiredItems, "{}"))
	err := json.Unmarshal([]byte(data), &e.requiredItems)
	if err != nil {
		return nil, err
	}

	return e.requiredItems, nil
}
