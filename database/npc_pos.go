package database

import (
	"database/sql"
	"fmt"
	"sync"

	"github.com/twodragon/kore-server/utils"

	gorp "gopkg.in/gorp.v1"
)

var (
	NPCPostions = make(map[int]*NpcPosition)
	NPCPosMutex sync.RWMutex
)

type NpcPosition struct {
	ID          int     `db:"id"`
	NPCID       int     `db:"npc_id"`
	MapID       int16   `db:"map"`
	Min_X       float64 `db:"min_location_x"`
	Max_X       float64 `db:"min_location_Y"`
	Min_Y       float64 `db:"max_location_x"`
	Max_Y       float64 `db:"max_location_y"`
	Count       int16   `db:"count"`
	RespawnTime int     `db:"respawn_time"`
	IsNPC       bool    `db:"is_npc"`
	Attackable  bool    `db:"attackable"`
	Rotation    int     `db:"rotation"`

	PseudoID    uint16 `db:"-"`
	MinLocation string `db:"-"`
	MaxLocation string `db:"-"`
}

func GetNPCPostions() []*NpcPosition {
	NPCPosMutex.RLock()
	defer NPCPosMutex.RUnlock()
	var arr []*NpcPosition
	for _, v := range NPCPostions {
		arr = append(arr, v)
	}
	return arr
}
func GetNPCPosByID(id int) *NpcPosition {
	NPCPosMutex.RLock()
	defer NPCPosMutex.RUnlock()
	nocpos, ok := NPCPostions[id]
	if !ok {
		return nil
	}
	return nocpos
}

func SetNPCPos(id int, pos *NpcPosition) {
	NPCPosMutex.Lock()
	defer NPCPosMutex.Unlock()
	NPCPostions[id] = pos
}

func (e *NpcPosition) SetLocations(min, max *utils.Location) {
	e.MinLocation = fmt.Sprintf("%.2f,%.2f", min.X, min.Y)
	e.MaxLocation = fmt.Sprintf("%.2f,%.2f", max.X, max.Y)

}

func (e *NpcPosition) Create() error {
	return pgsql_DbMap.Insert(e)
}

func (e *NpcPosition) CreateWithTransaction(tr *gorp.Transaction) error {
	return tr.Insert(e)
}

func (e *NpcPosition) Update() error {
	_, err := pgsql_DbMap.Update(e)
	return err
}

func (e *NpcPosition) Delete() error {
	_, err := pgsql_DbMap.Delete(e)
	return err
}

func GetAllNPCPos() error {

	var arr []*NpcPosition
	query := `select * from data.npc_pos order by id`

	if _, err := pgsql_DbMap.Select(&arr, query); err != nil {
		if err == sql.ErrNoRows {
			return err
		}
		return fmt.Errorf("GetAllNpcPos: %s", err.Error())
	}
	for _, pos := range arr {
		pos.MinLocation = fmt.Sprintf("%.2f,%.2f", pos.Min_X, pos.Min_Y)
		pos.MaxLocation = fmt.Sprintf("%.2f,%.2f", pos.Max_X, pos.Max_Y)
		SetNPCPos(pos.ID, pos)

	}

	return nil
}

func GetAllAIPos() ([]*NpcPosition, error) {

	var arr []*NpcPosition
	query := `select * from data.npc_pos WHERE is_npc = '0' order by id `

	if _, err := pgsql_DbMap.Select(&arr, query); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("GetAllNpcPos: %s", err.Error())
	}
	for _, pos := range arr {
		pos.MinLocation = fmt.Sprintf("%.2f,%.2f", pos.Min_X, pos.Min_Y)
		pos.MaxLocation = fmt.Sprintf("%.2f,%.2f", pos.Max_X, pos.Max_Y)
	}

	return arr, nil
}

func FindNPCPosByID(id int) (*NpcPosition, error) {

	var pos *NpcPosition
	query := `select * from data.npc_pos where id = $1`

	if err := pgsql_DbMap.SelectOne(&pos, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindNPCPosByID: %s", err.Error())
	}
	pos.MinLocation = fmt.Sprintf("%.2f,%.2f", pos.Min_X, pos.Min_Y)
	pos.MaxLocation = fmt.Sprintf("%.2f,%.2f", pos.Max_X, pos.Max_Y)

	return pos, nil
}

func FindNPCPosInMap(mapID int16) ([]*NpcPosition, error) {

	var arr []*NpcPosition
	query := `select * from data.npc_pos where "map" = $1`

	if _, err := pgsql_DbMap.Select(&arr, query, mapID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindNPCPosInMap: %s", err.Error())
	}
	for _, pos := range arr {
		pos.MinLocation = fmt.Sprintf("%.2f,%.2f", pos.Min_X, pos.Min_Y)
		pos.MaxLocation = fmt.Sprintf("%.2f,%.2f", pos.Max_X, pos.Max_Y)
	}

	return arr, nil
}

func RefreshAllNPCPos() ([]*NpcPosition, error) {

	var arr []*NpcPosition
	query := `select * from data.npc_pos order by id`

	if _, err := pgsql_DbMap.Select(&arr, query); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("GetAllNpcPos: %s", err.Error())
	}
	for _, pos := range arr {
		pos.MinLocation = fmt.Sprintf("%.2f,%.2f", pos.Min_X, pos.Min_Y)
		pos.MaxLocation = fmt.Sprintf("%.2f,%.2f", pos.Max_X, pos.Max_Y)
	}

	return arr, nil
}
