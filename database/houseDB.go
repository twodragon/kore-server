package database

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	//	"github.com/go-co-op/gocron"
	"github.com/twodragon/kore-server/utils"
	null "gopkg.in/guregu/null.v3"
)

var (
	HousingItems      = make(map[int]*HousingItem)
	HousingItemsMutex sync.RWMutex

	MAX_CROPS     = 100
	MAX_FURNITURE = 10
)

type HousingItem struct {
	ID             int       `db:"id"`
	HouseID        int       `db:"house_id"`
	PosX           float64   `db:"pos_x"`
	PosY           float64   `db:"pos_y"`
	PosZ           float64   `db:"pos_z"`
	ItemID         int64     `db:"item_id"`
	OwnerID        int       `db:"owner"`
	ExpirationDate null.Time `db:"expires_at" json:"expires_at"`
	MapID          int16     `db:"map_id"`
	Server         int16     `db:"server"`
	MaxRelaxation  int       `db:"max_relaxation"`
	IsPublic       int       `db:"is_public"`

	PseudoID uint16 `db:"-"`

	PlayersMutex   sync.RWMutex        `db:"-"`
	OnSightPlayers map[int]interface{} `db:"-" json:"players"`
}

func FindBuiltHouses() error {

	var arr []*HousingItem
	query := `select * from hops.houses`
	if _, err := pgsql_DbMap.Select(&arr, query); err != nil {
		if err == sql.ErrNoRows {
			return err
		}
		return fmt.Errorf("FindBuiltHouses: %s", err.Error())
	}
	for _, house := range arr {
		exp := house.ExpirationDate.Time.Local()
		house.ExpirationDate = null.NewTime(exp, true)
		HousingItems[house.ID] = house
		HousingItems[house.ID].InitCrop()

	}

	return nil
}

func (g *HousingItem) Create() error {
	return pgsql_DbMap.Insert(g)
}

func (g *HousingItem) Update() error {
	_, err := pgsql_DbMap.Update(g)
	return err
}

func (g *HousingItem) Delete() error {
	gMutex.Lock()
	defer gMutex.Unlock()
	delete(Guilds, g.ID)

	_, err := pgsql_DbMap.Delete(g)
	return err
}

func (h *HousingItem) GetNearbyCharacters() ([]*Character, error) {

	var (
		distance = 50.0
		chars    []*Character
	)

	candidates := FindCharactersInMap(h.MapID)

	for _, char := range candidates {

		characterCoordinate := ConvertPointToLocation(char.Coordinate)

		itemLocation := ConvertPointToCoordinate(h.PosX, h.PosY)
		itemCoordinate := ConvertPointToLocation(itemLocation)
		if char.Socket.User.ConnectedServer == int(h.Server) && char.IsActive && char.IsOnline && utils.CalculateDistance(characterCoordinate, itemCoordinate) <= distance {
			chars = append(chars, char)
		}

	}

	return chars, nil
}
func FindHouseByCharId(c int) *HousingItem {
	for _, house := range HousingItems {
		if house.HouseID >= 10000 {
			continue
		}
		if house.OwnerID == c {
			return house
		}
	}
	return nil
}
func FindHouseByPseudoId(p int) *HousingItem {
	for _, house := range HousingItems {

		if house.PseudoID == uint16(p) {
			return house
		}
	}
	return nil
}
func (h *HousingItem) InitCrop() {
	info, ok := HouseItemsInfos[h.HouseID]

	if !ok {
		return
	}
	if info.Category == 1 {
		char, err := FindCharacterByID(h.OwnerID)
		if err != nil || char == nil {
			return
		}
		//	go RecoverRelaxPointsHandler(char, h)

	}
	if info.Category != 2 {
		return
	}
	if info.CanCollect == 1 {
		duration := time.Until(h.ExpirationDate.Time.Local())
		if duration <= 0 {
			h.Remove()
			return
		}
		time.AfterFunc(time.Until(h.ExpirationDate.Time.Local()), func() {
			if h != nil {
				h.Remove()
			}
		})
	} else if info.NextStage != 0 {
		duration := time.Until(h.ExpirationDate.Time.Local())

		if duration <= 0 {
			duration = time.Duration(0)
		}
		time.AfterFunc(duration, func() {

			h.HouseID = info.NextStage
			h.ItemID++

			info, ok := HouseItemsInfos[h.HouseID]
			if !ok {
				return
			}

			expiration := time.Now().Add(time.Second * time.Duration(int64(info.Timer)))
			h.ExpirationDate = null.NewTime(expiration, true)
			h.Update()

			chars, _ := h.GetNearbyCharacters()

			for _, char := range chars {

				char.OnSight.HousingitemsMutex.Lock()
				delete(char.OnSight.Housingitems, h.ID)
				char.OnSight.HousingitemsMutex.Unlock()
			}
			h.InitCrop()
		})
	}
}

func (h *HousingItem) Remove() {

	r := utils.Packet{0xaa, 0x55, 0x08, 0x00, 0xac, 0x04, 0x0a, 0x00, 0x3e, 0x07, 0x01, 0x00, 0x55, 0xaa}
	r.Overwrite(utils.IntToBytes(uint64(h.PseudoID), 4, true), 8)

	chars, _ := h.GetNearbyCharacters()

	for _, char := range chars {

		char.OnSight.HousingitemsMutex.Lock()
		delete(char.OnSight.Housingitems, h.ID)
		char.OnSight.HousingitemsMutex.Unlock()
		char.Socket.Write(r)
	}

	h.Delete()
	delete(HousingItems, h.ID)
}
func (c *Character) CountHouseItemsByCategory(cat int) int {
	count := 0
	for _, item := range HousingItems {
		if item.OwnerID == c.ID && HouseItemsInfos[item.HouseID].Category == cat {
			count++
		}
	}

	return count

}

/*func RecoverRelaxPointsHandler(char *Character, h *HousingItem) {
	var err error
	s := gocron.NewScheduler(time.Local)
	task1 := func() {
		if !char.IsActive && !char.IsOnline && char.Relaxation < h.MaxRelaxation {
			char.Relaxation++
			char.Update()
		}
	}
	_, err = s.Every("1m").Do(task1)

	s.StartAsync()
	if err != nil {
		fmt.Print(err)
	}
}*/
