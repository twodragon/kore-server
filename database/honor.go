package database

import (
	"database/sql"
	"fmt"
	"log"

	gorp "gopkg.in/gorp.v1"
)

type Rank struct {
	ID       int `db:"id" json:"id"`
	HonorID  int `db:"honor_id" json:"honor_id"`
	PlusSTR  int `db:"str" json:"str"`
	PlusDEX  int `db:"dex" json:"dex"`
	PlusINT  int `db:"int" json:"int"`
	PlusStat int `db:"plus_stat" json:"plus_stat"`
	PlusSP   int `db:"plus_skillpoint" json:"plus_skillpoint"`
}

func (b *Rank) Create() error {
	return pgsql_DbMap.Insert(b)
}

func (b *Rank) CreateWithTransaction(tr *gorp.Transaction) error {
	return tr.Insert(b)
}

func (b *Rank) Delete() error {
	_, err := pgsql_DbMap.Delete(b)
	return err
}
func (b *Rank) Update() error {
	_, err := pgsql_DbMap.Update(b)
	if err != nil {
		log.Print(fmt.Sprintf("Error: %s", err.Error()))
	}
	return err
}

func FindRankByHonorID(rankID int64) (*Rank, error) {

	var g Rank
	query := `select * from honor where honor_id = $1`

	if err := pgsql_DbMap.SelectOne(&g, query, rankID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindGuildByName: %s", err.Error())
	}

	return &g, nil
}
