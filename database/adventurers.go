package database

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/twodragon/kore-server/utils"
	"gopkg.in/guregu/null.v3"
)

var (
	Adventurers = make(map[int]*Adventurer)
)

type Adventurer struct {
	ID              int       `db:"id"`
	CharID          int       `db:"charid"`
	Index           int       `db:"index"`
	FinishAt        null.Time `db:"finish_at"`
	TotalAdventures int       `db:"total_adventures"`
	Level           int       `db:"level"`
	Status          int       `db:"status"`
}

func (b *Adventurer) Delete() error {
	_, err := pgsql_DbMap.Delete(b)
	return err
}

func (e *Adventurer) Create() error {
	return pgsql_DbMap.Insert(e)
}

func (b *Adventurer) Update() error {
	_, err := pgsql_DbMap.Update(b)
	if err != nil {
		fmt.Printf("Error: %s", err.Error())
	}
	return err
}
func GetAdventurers() error {
	var checks []*Adventurer
	query := `select * from hops.adventurers`

	if _, err := pgsql_DbMap.Select(&checks, query); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("GetAdventurers: %s", err.Error())
	}

	for _, cr := range checks {
		exp := cr.FinishAt.Time.Local()
		cr.FinishAt = null.NewTime(exp, true)
		Adventurers[cr.ID] = cr
		Adventurers[cr.ID].Init()
	}
	return nil
}

func (adv *Adventurer) GetData() []byte {

	resp := utils.Packet{0xaa, 0x55, 0x1f, 0x00, 0xd7, 0x02, 0x55, 0xaa}
	index := 6

	resp.Insert(utils.IntToBytes(uint64(adv.Index), 4, true), index)
	index += 4
	resp.Insert(utils.IntToBytes(uint64(adv.TotalAdventures), 4, true), index) //level
	index += 4
	resp.Insert(utils.IntToBytes(uint64(adv.Status), 1, true), index) //hired
	index += 1

	formatdate := adv.FinishAt.Time.Format("2006-01-02 15:04:05") // expires at

	resp.Insert(utils.IntToBytes(uint64(len(formatdate)), 1, false), index)
	index += 1

	resp.Insert([]byte(formatdate), index)
	index += len(formatdate)

	resp.SetLength(int16(binary.Size(resp) - 6))

	return resp
}

func (adv *Adventurer) Init() {
	if adv == nil {
		return
	}
	if adv.Status == 1 {
		duration := time.Until(adv.FinishAt.Time)
		if duration <= 0 {
			duration = time.Duration(0)
		}
		time.AfterFunc(duration, func() {
			adv.Status = 2
			adv.Update()
		})
	}
}
