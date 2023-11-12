package database

import (
	"database/sql"
	"fmt"
	"time"
)

var (
	FiveClans   = make(map[int]*FiveClan)
	TempleDatas = make(map[int]*TempleData)
)

type TempleData struct {
	ID          int    `db:"id" json:"id"`
	TempleName  string `db:"temple_name" `
	GateNpcID   int    `db:"gate_npcid"`
	GuardNpcID  int    `db:"guard_npcid" `
	StatueNpcID int    `db:"statue_npcid" `
	Cooldown    int    `db:"cooldown" `
	BuffID      int    `db:"buff_id" `
}

func getTempleDatas() error {

	query := `select * from data.five_clan_war`

	temple := []*TempleData{}

	if _, err := pgsql_DbMap.Select(&temple, query); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("getAllItems: %s", err.Error())
	}

	for _, Temple := range temple {
		TempleDatas[int(Temple.ID)] = Temple
	}

	return nil
}
func GetTempleDataByStatue(statueNpcId int) *TempleData {
	for _, temple := range TempleDatas {
		if temple.StatueNpcID == statueNpcId {
			return temple
		}
	}
	return nil
}
func GetTempleDataByGuard(guardNpcId int) *TempleData {
	for _, temple := range TempleDatas {
		if temple.GuardNpcID == guardNpcId {
			return temple
		}
	}
	return nil
}
func GetTempleDataByGate(gateNpcId int) *TempleData {
	for _, temple := range TempleDatas {
		if temple.GuardNpcID == gateNpcId {
			return temple
		}
	}
	return nil
}

type FiveClan struct {
	AreaID   int   `db:"id"`
	ClanID   int   `db:"clanid"`
	Buff     int64 `db:"buff"`
	Duration int64 `db:"duration"`
}

func (b *FiveClan) Update() error {
	_, err := pgsql_DbMap.Update(b)
	return err
}

func getFiveAreas() error {
	var areas []*FiveClan
	query := `select * from hops.fiveclan_war`

	if _, err := pgsql_DbMap.Select(&areas, query); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("getFiveAreas: %s", err.Error())
	}

	for _, cr := range areas {
		FiveClans[cr.AreaID] = cr
	}
	return nil
}

func HandleClanBuffs() {
	getFiveAreas()
	for _, temple := range FiveClans {
		if temple.ClanID != 0 {
			if temple.Duration > 0 {
				temple.Duration -= 1
				temple.Update()
			} else {
				temple.Buff = 0
				temple.ClanID = 0
				temple.Duration = 0
				temple.Update()
			}
		}
	}
	time.AfterFunc(time.Minute, func() { HandleClanBuffs() })
}
