package database

import (
	"database/sql"
	"encoding/json"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/twodragon/kore-server/gold"
	"github.com/twodragon/kore-server/utils"

	"github.com/thoas/go-funk"
	null "gopkg.in/guregu/null.v3"
)

var (
	categories = map[int][]int{
		1: {44, 49, 70, 71, 99, 100, 101, 102, 103, 104, 105, 107, 108}, 101: {102, 103}, 102: {105}, 103: {108}, 104: {104}, 105: {107}, 106: {101}, 107: {100}, 108: {99}, 109: {70, 71}, 110: {44, 49},
		2: {121, 122, 123, 124, 175}, 201: {121}, 202: {122}, 203: {123}, 204: {124}, 205: {}, 206: {}, 207: {}, 208: {}, 209: {HT_ARMOR_HELM}, 210: {HT_ARMOR_MASK}, 211: {HT_ARMOR_BOOTS}, 212: {HT_ARMOR_ARMOR}, 213: {HT_ARMOR_HOUSE_UNIFORM},
		3: {131, 132, 133, 134, 90}, 301: {131}, 302: {132}, 303: {133}, 304: {134}, 305: {MASTER_HT_STONE}, 306: {MASTER_HT_CHARM}, 307: {MASTER_HT_ORB}, 308: {MASTER_HT_STONE},
		4: {MARBLE_TYPE}, 401: {MARBLE_TREE}, 402: {MARBLE_FIRE}, 403: {MARBLE_EARTH}, 404: {MARBLE_STEEL}, 405: {MARBLE_WATER},
		5: {135, 136, 137, 221, 222, 223}, 501: {PET_TYPE}, 502: {PET_ITEM_HEAD}, 503: {PET_ITEM_ARMOR}, 504: {PET_ITEM_CLAW}, 505: {PET_POTION_TYPE, DEAD_SPIRIT_INCENSE_TYPE}, 506: {},
		6: {147, 148, 149, 150, 151, 152, 153, 154, 156, 159, 171, 168, 217}, 601: {147, 149, 150, 151}, 602: {171, 254, FORM_TYPE, 168, 217, 159}, 603: {FOOD_TYPE, INGREDIENTS_TYPE}, 604: {BAG_EXPANSION_TYPE}, 605: {CHARM_OF_RETURN_TYPE, MOVEMENT_SCROLL_TYPE}, 606: {MOB_SUMMONING_SCROLL}, 607: {},
		7: {80, 81, 190, 191, 192, 194, 195}, 701: {UPGRADING_STONE_TYPE}, 702: {UPGRADING_STONE_TYPE}, 703: {UPGRADING_STONE_TYPE}, 704: {UPGRADING_STONE_TYPE}, 705: {UPGRADING_STONE_TYPE}, 706: {UPGRADING_STONE_TYPE}, 707: {HOLY_WATER_TYPE}, 708: {SOCKET_TYPE},
		8: {164, 166, 167, 219}, 801: {CHARM_OF_LUCK}, 802: {CHARM_OF_LUCK}, 803: {CHARM_OF_LUCK}, 804: {SCALE_TYPE, BAGUA},
		9: {164, 166, 167, 187, 189, 219}, 901: {164, 166, 167, 187, 189, 219}, 902: {164, 166, 167, 187, 189, 219}, 903: {164, 166, 167, 187, 189, 219}, 904: {164, 166, 167, 187, 189, 219},
		10: {161}, 1001: {161}, 1002: {161},
		11:  {1},
		100: {1},
	}

	IsHideBannedUserItems = false
)

var (
	ConsignmentItemsList    = make(map[int]*ConsignmentItem)
	ConsignmentMutex        sync.RWMutex
	ConsignmentActionsMutex sync.Mutex
)

type ConsignmentItem struct {
	ID             int             `db:"id"`
	UserID         null.String     `db:"user_id"`
	SellerID       null.Int        `db:"character_id"`
	ItemID         int64           `db:"item_id"`
	SlotID         int16           `db:"slot_id"`
	Quantity       uint            `db:"quantity"`
	Plus           uint8           `db:"plus"`
	UpgradeArr     string          `db:"upgrades"`
	SocketCount    int8            `db:"socket_count"`
	SocketArr      string          `db:"sockets"`
	Activated      bool            `db:"activated"`
	InUse          bool            `db:"in_use"`
	PetInfo        json.RawMessage `db:"pet_info"`
	Consignment    bool            `db:"consignment"`
	Appearance     int64           `db:"appearance"`
	ItemType       int16           `db:"item_type"`
	JudgementStat  int64           `db:"judgement_stat"`
	Buff           int             `db:"buff"`
	IsServerEpoch  bool            `db:"server_epoch"`
	ActivationTime null.Time       `db:"activated_at"`
	Price          uint64          `db:"price" json:"price"`
	IsSold         bool            `db:"is_sold" json:"is_sold"`
	ExpiresAt      null.Time       `db:"expires_at" json:"expires_at"`

	Pet       *PetSlot `db:"-" json:"-"`
	IsExpired bool     `db:"-"`
}

func GetConsignmentData() []*ConsignmentItem {
	ConsignmentMutex.RLock()
	defer ConsignmentMutex.RUnlock()

	arr := make([]*ConsignmentItem, 0, len(ConsignmentItemsList))
	for _, consitem := range ConsignmentItemsList {
		arr = append(arr, consitem)
	}
	return arr
}
func GetConsignmentDataById(id int) (*ConsignmentItem, bool) {
	ConsignmentMutex.RLock()
	defer ConsignmentMutex.RUnlock()
	consignItem, ok := ConsignmentItemsList[id]
	if !ok {
		return nil, ok
	}

	return consignItem, ok
}

func RemoveConsignmentData(id int) {
	ConsignmentMutex.Lock()
	defer ConsignmentMutex.Unlock()

	delete(ConsignmentItemsList, id)
}

func SetConsignmentData(consitem *ConsignmentItem) {
	ConsignmentMutex.Lock()
	defer ConsignmentMutex.Unlock()

	ConsignmentItemsList[consitem.ID] = consitem
}

func ReadConsignmentData() error {
	var arr []*ConsignmentItem
	query := `select * from hops.consign`
	if _, err := pgsql_DbMap.Select(&arr, query); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}
	ConsignmentMutex.Lock()
	ConsignmentItemsList = make(map[int]*ConsignmentItem)
	ConsignmentMutex.Unlock()
	for _, consitem := range arr { //
		consitem.IsExpired = false
		if time.Since(consitem.ExpiresAt.Time) >= 0 {
			consitem.IsExpired = true
		}

		SetConsignmentData(consitem)

		if consitem.PetInfo != nil {
			json.Unmarshal(consitem.PetInfo, &consitem.Pet)
		}
		InventoryItems.Add(consitem.ID, consitem)

	}

	return nil
}

func (slot *ConsignmentItem) Insert() error {

	if slot.UpgradeArr == "" {
		slot.UpgradeArr = "{0,0,0,0,0,0,0,0,0,0,0,0,0,0,0}"
	}
	if slot.SocketArr == "" {
		slot.SocketArr = "{0,0,0,0,0,0,0,0,0,0,0,0,0,0,0}"
	}

	if slot.Pet != nil {
		slot.PetInfo, _ = json.Marshal(slot.Pet)
	}

	if slot.PetInfo == nil {
		slot.PetInfo = json.RawMessage("{}")
	}

	err := pgsql_DbMap.Insert(slot)
	if err != nil {
		return err
	}

	return nil
}

func (e *ConsignmentItem) Delete() error {
	_, err := pgsql_DbMap.Delete(e)
	return err
}

func (e *ConsignmentItem) Update() error {
	_, err := pgsql_DbMap.Update(e)
	return err
}

func GetConsignmentItems(page, category, subcategory, minUpgLevel, maxUpgLevel, orderBy int, minPrice, maxPrice uint64, itemName string) ([]*ConsignmentItem, int64, error) {

	if maxPrice == 50*gold.B {
		maxPrice = math.MaxInt64
	}

	cat := category
	if subcategory > 0 {
		cat = category*100 + subcategory
	}

	cats, ok := categories[cat]

	consitems := GetConsignmentData()

	items := funk.Filter(consitems, func(item *ConsignmentItem) bool {
		iteminfo, kk := GetItemInfo(item.ItemID)
		if !kk {
			return false
		}
		if iteminfo == nil { //
			return false
		}
		seller, err := FindCharacterByID(int(item.SellerID.Int64))
		if err != nil || seller == nil {
			return false
		}
		if item.IsExpired {
			return false
		}
		if !item.IsSold && strings.Contains(strings.ToLower(iteminfo.Name), strings.ToLower(itemName)) {
			if !ok {
				return true
			} else {
				return funk.Contains(cats, int(iteminfo.Type))
			}
		}

		return false
	}).([]*ConsignmentItem)

	sort.Slice(items, func(i, j int) bool {
		iteminfo_i, ok := GetItemInfo(items[i].ItemID)
		iteminfo_j, ok := GetItemInfo(items[j].ItemID)
		if !ok {
			return false
		}
		if int8(orderBy) == 4 {
			return items[i].Price > items[j].Price
		} else if int8(orderBy) == -4 {

			return items[i].Price < items[j].Price
		} else if int8(orderBy) == 2 {
			return items[i].Quantity > items[j].Quantity
		} else if int8(orderBy) == -2 {
			return items[i].Quantity < items[j].Quantity
		} else if int8(orderBy) == 1 {
			return iteminfo_i.Name > iteminfo_j.Name
		} else if int8(orderBy) == -3 {
			return utils.ParseDate(items[i].ExpiresAt) < utils.ParseDate(items[j].ExpiresAt)
		} else if int8(orderBy) == -3 {
			return utils.ParseDate(items[i].ExpiresAt) > utils.ParseDate(items[j].ExpiresAt)
		}
		return iteminfo_i.Name > iteminfo_j.Name
	})
	return items, int64(len(items)), nil
}

func (e *ConsignmentItem) GetUpgrades() []byte {

	upgs := strings.Split(strings.Trim(string(e.UpgradeArr), "{}"), ",")
	return funk.Map(upgs, func(upg string) byte {
		u, _ := strconv.ParseUint(upg, 10, 8)
		return byte(u)
	}).([]byte)
}

func (e *ConsignmentItem) GetSockets() []byte {
	upgs := strings.Split(strings.Trim(string(e.SocketArr), "{}"), ",")
	return funk.Map(upgs, func(upg string) byte {
		u, _ := strconv.ParseUint(upg, 10, 8)
		return byte(u)
	}).([]byte)
}

func FindConsignmentItemsBySellerID(sellerID int) ([]*ConsignmentItem, error) {
	items := GetConsignmentData()
	filtered := funk.Filter(items, func(item *ConsignmentItem) bool {
		return item.SellerID == null.IntFrom(int64(sellerID))
	}).([]*ConsignmentItem)
	return filtered, nil
}
