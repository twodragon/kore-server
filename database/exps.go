package database

import (
	"database/sql"
	"fmt"

	gorp "gopkg.in/gorp.v1"
)

type ExpInfo struct {
	Level       int16 `db:"level"`
	Exp         int64 `db:"exp"`
	SkillPoints int   `db:"skill_points"`
}

var (
	EXPs = make(map[int16]*ExpInfo)
)

func (e *ExpInfo) Create() error {
	return pgsql_DbMap.Insert(e)
}

func (e *ExpInfo) CreateWithTransaction(tr *gorp.Transaction) error {
	return tr.Insert(e)
}

func (e *ExpInfo) Update() error {
	_, err := pgsql_DbMap.Update(e)
	return err
}

func (e *ExpInfo) Delete() error {
	_, err := pgsql_DbMap.Delete(e)
	return err
}

func GetExps() error {

	var arr []*ExpInfo
	query := "select * from data.exp_table"

	if _, err := pgsql_DbMap.Select(&arr, query); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("GetExps: %s", err.Error())
	}

	for _, e := range arr {
		EXPs[e.Level] = e
	}
	return nil
}
