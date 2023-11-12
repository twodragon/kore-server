package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	gorp "gopkg.in/gorp.v1"
)

var (
	allTeleports = make(map[int]*Teleports)
)

type Teleports struct {
	ID        int             `db:"id" json:"id"`
	Teleports json.RawMessage `db:"teleport_info" json:"teleports"`
}
type TeleportsSlots struct {
	Slots []*TeleportSet `json:"teleports"`
}

type TeleportSet struct {
	Teleportslots []*SlotsTuple `json:"slots"`
}

type SlotsTuple struct {
	SlotID int `json:"slotid"`
	MapID  int `json:"mapid"`
	Coordx int `json:"coordx"`
	Coordy int `json:"coordy"`
}

func (e *Teleports) Create(c *Character) error {
	e.ID = c.ID
	e.Teleports = json.RawMessage(`{"teleports": [{}, {}, {}, {}, {}, {}]}`)
	return pgsql_DbMap.Insert(e)
}

func (e *Teleports) CreateWithTransaction(tr *gorp.Transaction) error {
	return tr.Insert(e)
}

func (e *Teleports) Update() error {
	_, err := pgsql_DbMap.Update(e)
	return err
}

func (e *Teleports) Delete() error {
	skMutex.Lock()
	delete(allTeleports, e.ID)
	skMutex.Unlock()

	_, err := pgsql_DbMap.Delete(e)
	return err
}

func (e *Teleports) GetTeleports() (*TeleportsSlots, error) {
	if e == nil {
		return nil, fmt.Errorf("Null teleport list.")
	}
	slots := &TeleportsSlots{}
	err := json.Unmarshal([]byte(e.Teleports), &slots)
	if err != nil {
		log.Print("Error: ", err)
		return nil, err
	}
	return slots, nil
}

func (e *Teleports) SetTeleports(slots *TeleportsSlots) error {
	data, err := json.Marshal(slots)
	if err != nil {
		return err
	}

	e.Teleports = json.RawMessage(data)
	return nil
}

func FindTeleportsByID(id int) (*Teleports, error) {

	skMutex.RLock()
	s, ok := allTeleports[id]
	skMutex.RUnlock()

	if ok {
		return s, nil
	}

	query := `select * from hops.characters_teleports where id = $1`
	teleport := &Teleports{}
	if err := pgsql_DbMap.SelectOne(&teleport, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindTeleportsByID: %s", err.Error())
	}

	skMutex.Lock()
	allTeleports[id] = teleport
	skMutex.Unlock()

	return teleport, nil
}
