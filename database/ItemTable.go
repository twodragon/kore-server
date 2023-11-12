package database

import (
	"fmt"
	"log"
	"sync"

	"github.com/twodragon/kore-server/utils"
	"github.com/xuri/excelize/v2"
	gorp "gopkg.in/gorp.v1"
)

var (
	ItemsList        = make(map[int64]*Item)
	ItemsMutex       = sync.RWMutex{}
	STRRates         = []int{350, 350, 325, 225, 175, 150, 80, 110, 90, 75, 50, 40, 30, 20, 15}
	BEASTstrRates    = []int{350, 350, 325, 225, 175, 150, 80, 110, 90, 75, 50, 40, 30, 20, 15}
	STRHappyHourRate = 0.00
	haxBoxes         = []int64{92000002, 92000003, 92000004, 92000005, 92000006, 92000007, 92000008, 92000009, 92000010,
		92000055, 92000056, 92000057, 92000058, 92000059, 92000060}
	AidTonics = []int64{13000037, 13000011, 13000012, 13000013, 13000014, 13000015, 13000060, 13000074}
)

func GetItemInfo(id int64) (*Item, bool) {
	ItemsMutex.RLock()
	defer ItemsMutex.RUnlock()
	item, ok := ItemsList[id]
	return item, ok
}
func SetItem(item *Item) {
	ItemsMutex.Lock()
	defer ItemsMutex.Unlock()
	ItemsList[item.ID] = item
}

const (
	WEAPON_TYPE = iota
	ARMOR_TYPE
	HT_ARMOR_TYPE
	HT_ARMOR_HELM
	HT_ARMOR_MASK
	HT_ARMOR_BOOTS
	HT_ARMOR_ARMOR
	DEATHKING_CASTLE_TICKET
	PANDEMONIUM_ENTRY_TICKET
	HT_ARMOR_HOUSE_UNIFORM
	ACC_TYPE
	PENDENT_TYPE
	QUEST_TYPE
	PET_ITEM_TYPE
	PET_ITEM_HEAD
	PET_ITEM_ARMOR
	PET_ITEM_CLAW
	SKILL_BOOK_TYPE
	PASSIVE_SKILL_BOOK_TYPE
	POTION_TYPE
	PET_TYPE
	PET_POTION_TYPE
	CHARM_OF_RETURN_TYPE
	FORTUNE_BOX_TYPE
	CHARM_TYPE
	MARBLE_TYPE
	MARBLE_TREE
	MARBLE_FIRE
	MARBLE_EARTH
	MARBLE_STEEL
	MARBLE_WATER
	MAP_BOOK_TYPE
	WRAPPER_BOX_TYPE
	NPC_SUMMONER_TYPE
	FIRE_SPIRIT
	WATER_SPIRIT
	HOLY_WATER_TYPE
	ESOTERIC_POTION_TYPE
	FILLER_POTION_TYPE
	SCALE_TYPE
	BAG_EXPANSION_TYPE
	MOVEMENT_SCROLL_TYPE
	SOCKET_TYPE
	FOOD_TYPE
	INGREDIENTS_TYPE
	DEAD_SPIRIT_INCENSE_TYPE
	AFFLICTION_TYPE
	RESET_ART_TYPE
	RESET_ARTS_TYPE
	FORM_TYPE
	MASTER_HT_ACC
	MASTER_HT_ORB
	MASTER_HT_STONE
	MASTER_HT_TABLET
	MASTER_HT_CHARM
	UPGRADING_STONE_TYPE
	UNKNOWN_TYPE
	MAP_BOOK
	MOB_SUMMONING_SCROLL
	CHARM_OF_LUCK
	BAGUA
	TOTAL_TRANSFORMATION
	BOW
	ARROW
	CHARM_OF_FACTION_TYPE
	SPECIAL_USAGE
	SPECIAL_USAGE2
	EXPANSION
	BOOK_OF_PET_TYPE
	SOCKET_REVISION_TYPE
	SOCKET_STABILIZER_TYPE
	SOCKET_INITIALIZATION
	SOCKET_MILED_STONE
	SHOUT_ARTS_TYPE
	ILLUSION_WATER_TYPE
	TRANSFORMATION_PAPER_TYPE
	ARTS_EXCHANGE_TYPE
	HT_POTIONS
)

type Item struct {
	ID              int64   `db:"id"`
	Name            string  `db:"name"`
	UIF             string  `db:"uif"`
	Type            int16   `db:"type"`
	ItemPair        int64   `db:"itempair"`
	HtType          int16   `db:"ht_type"`
	TimerType       int16   `db:"timer_type"`
	Timer           int     `db:"timer"`
	MinUpgradeLevel uint64  `db:"min_upgrade_level"`
	SpecialItem     int64   `db:"special_item"`
	BuyPrice        int64   `db:"buy_price"`
	SellPrice       int64   `db:"sell_price"`
	Slot            int     `db:"slot"`
	CharacterType   int     `db:"character_type"`
	MinLevel        int     `db:"min_level"`
	MaxLevel        int     `db:"max_level"`
	BaseDef1        int     `db:"base_def1"`
	BaseDef2        int     `db:"base_def2"`
	BaseDef3        int     `db:"base_def3"`
	BaseMinAtk      int     `db:"base_min_atk"`
	BaseMaxAtk      int     `db:"base_max_atk"`
	STR             int     `db:"str"`
	DEX             int     `db:"dex"`
	INT             int     `db:"int"`
	Wind            int     `db:"wind"`
	Water           int     `db:"water"`
	Fire            int     `db:"fire"`
	MaxHp           int     `db:"max_hp"`
	HPRecoveryRate  int     `db:"hp_recovery_rate"`
	MaxChi          int     `db:"max_chi"`
	CHIRecoveryRate int     `db:"chi_recovery_rate"`
	RunningSpeed    float64 `db:"running_speed"`
	MinAtk          int     `db:"min_atk"`
	MaxAtk          int     `db:"max_atk"`
	AtkRate         int     `db:"atk_rate"`
	MinArtsAtk      int     `db:"min_arts_atk"`
	MaxArtsAtk      int     `db:"max_arts_atk"`
	ArtsAtkRate     int     `db:"arts_atk_rate"`
	Def             int     `db:"def"`
	DefRate         int     `db:"def_rate"`
	ArtsDef         int     `db:"arts_def"`
	ArtsDefRate     int     `db:"arts_def_rate"`
	Accuracy        int     `db:"accuracy"`
	Dodge           int     `db:"dodge"`
	HpRecovery      int     `db:"hp_recovery"`
	ChiRecovery     int     `db:"chi_recovery"`
	ExpRate         float64 `db:"exp_rate"`
	DropRate        float64 `db:"drop_rate"`
	Tradable        int     `db:"tradable"`
	HolyWaterPlus   int     `db:"holy_water_plus"`
	HolyWaterUpg1   int     `db:"holy_water_upg1"`
	HolyWaterUpg2   int     `db:"holy_water_upg2"`
	HolyWaterUpg3   int     `db:"holy_water_upg3"`
	HolyWaterRate1  int     `db:"holy_water_rate1"`
	HolyWaterRate2  int     `db:"holy_water_rate2"`
	HolyWaterRate3  int     `db:"holy_water_rate3"`
	PoisonATK       int     `db:"poison_attack"`
	PoisonDEF       int     `db:"poison_defense"`
	ParaATK         int     `db:"para_attack"`
	ParaDEF         int     `db:"para_defense"`
	ConfusionATK    int     `db:"confusion_attack"`
	ConfusionDEF    int     `db:"confusion_defense"`
	PoisonTime      int     `db:"poison_time"`
	ParaTime        int     `db:"para_time"`
	ConfusionTime   int     `db:"confusion_time"`
	PVPdmg          int     `db:"pvp_dmg"`
	PVPsdmg         int     `db:"pvp_sdmg"`
	PVPdefRate      int     `db:"pvp_def_rate"`
	PVPsdefRate     int     `db:"pvp_sdef_rate"`
	Buff            int     `db:"buff"`
	BeastType       int     `db:"beast_item"`
	Range           int     `db:"shooting_distance"`

	SpecialEffectID          int `db:"special_effect_id"`
	SpecialEffectProbability int `db:"special_effect_probability"`
	SpecialEffectValue       int `db:"special_effect_value"`

	PvpDampReduction float64 `db:"pvp_damp_reduction"`

	CanCreateSocket int `db:"can_create_socket"`
	NPCID           int

	Stackable bool
}

func (item *Item) Create() error {
	return pgsql_DbMap.Insert(item)
}

func (item *Item) CreateWithTransaction(tr *gorp.Transaction) error {
	return tr.Insert(item)
}

func (item *Item) Delete() error {
	_, err := pgsql_DbMap.Delete(item)
	return err
}

func (item *Item) Update() error {
	_, err := pgsql_DbMap.Update(item)
	return err
}

func (item *Item) GetType() int {
	if item == nil {
		return -1
	}
	if item.ID == 80006067 {
		return CHARM_OF_FACTION_TYPE
	} else if item.Type == 50 {
		return SOCKET_REVISION_TYPE
	} else if item.Type == 51 {
		return FIRE_SPIRIT
	} else if item.Type == 52 {
		return WATER_SPIRIT
	} else if item.Type == 59 {
		return BAG_EXPANSION_TYPE
	} else if item.Type == 64 {
		return MARBLE_TYPE
	} else if item.Type == 64 && item.Slot == 397 {
		return MARBLE_TREE
	} else if item.Type == 64 && item.Slot == 398 {
		return MARBLE_FIRE
	} else if item.Type == 64 && item.Slot == 399 {
		return MARBLE_EARTH
	} else if item.Type == 64 && item.Slot == 400 {
		return MARBLE_STEEL
	} else if item.Type == 64 && item.Slot == 401 {
		return MARBLE_WATER
	} else if item.Type == 66 {
		return MAP_BOOK_TYPE
	} else if item.Type == 69 {
		return SOCKET_MILED_STONE
	} else if item.Type == 73 {
		return SOCKET_INITIALIZATION
	} else if item.Type == 76 {
		return BOOK_OF_PET_TYPE
	} else if (item.Type >= 70 && item.Type <= 71) || (item.Type >= 99 && item.Type <= 108) || item.Type == 44 || item.Type == 49 {
		return WEAPON_TYPE
	} else if item.Type == 80 {
		return SOCKET_TYPE
	} else if item.Type == 81 {
		return HOLY_WATER_TYPE
	} else if item.Type == 83 {
		return SOCKET_STABILIZER_TYPE
	} else if item.Type == 90 {
		return MASTER_HT_ACC
	} else if item.Type == 90 && item.Slot == 312 {
		return MASTER_HT_STONE
	} else if item.Type == 90 && item.Slot == 313 {
		return MASTER_HT_CHARM
	} else if item.Type == 90 && item.Slot == 314 {
		return MASTER_HT_ORB
	} else if item.Type == 90 && item.Slot == 315 {
		return MASTER_HT_STONE
	} else if item.Type == 93 {
		return MOB_SUMMONING_SCROLL
	} else if item.Type == 95 {
		return HT_POTIONS
	} else if item.Type == 110 {
		return AFFLICTION_TYPE
	} else if item.Type == 111 {
		return RESET_ART_TYPE
	} else if item.Type == 112 {
		return RESET_ARTS_TYPE
	} else if item.Type == 113 {
		return TRANSFORMATION_PAPER_TYPE
	} else if item.Type == 115 {
		return INGREDIENTS_TYPE
	} else if item.Type == 116 {
		return FOOD_TYPE
	} else if item.Type >= 121 && item.Type <= 124 && item.Slot < 307 {
		return ARMOR_TYPE
	} else if (item.Type >= 121 && item.Type <= 124) || item.Type == 175 {
		return HT_ARMOR_TYPE
	} else if item.Type == 121 && item.HtType > 0 && item.Slot == 307 {
		return HT_ARMOR_HELM
	} else if item.Type == 122 && item.HtType > 0 && item.Slot == 308 {
		return HT_ARMOR_MASK
	} else if item.Type == 122 && item.HtType > 0 && item.Slot == 310 {
		return HT_ARMOR_BOOTS
	} else if item.Type == 122 && item.HtType > 0 && item.Slot == 309 {
		return HT_ARMOR_ARMOR
	} else if item.Type == 122 && item.HtType > 0 {
		return HT_ARMOR_HOUSE_UNIFORM
	} else if item.Type >= 131 && item.Type <= 134 {
		return ACC_TYPE
	} else if item.Type >= 135 && item.Type <= 137 {
		return PET_ITEM_TYPE
	} else if item.Type == 135 {
		return PET_ITEM_HEAD
	} else if item.Type == 136 {
		return PET_ITEM_ARMOR
	} else if item.Type == 137 {
		return PET_ITEM_CLAW
	} else if item.Type == 147 {
		return FILLER_POTION_TYPE
	} else if item.Type == 150 {
		return ESOTERIC_POTION_TYPE
	} else if item.Type == 151 {
		return POTION_TYPE
	} else if item.Type == 152 {
		return CHARM_OF_RETURN_TYPE
	} else if item.Type == 153 {
		return MOVEMENT_SCROLL_TYPE
	} else if item.Type == 161 {
		return SKILL_BOOK_TYPE
	} else if item.Type == 162 {
		return PASSIVE_SKILL_BOOK_TYPE
	} else if item.Type == 165 {
		return SHOUT_ARTS_TYPE
	} else if item.Type == 166 {
		return SCALE_TYPE
	} else if item.Type == 168 || item.Type == 213 {
		return WRAPPER_BOX_TYPE
	} else if item.Type == 174 {
		return FORM_TYPE
	} else if item.Type >= 191 && item.Type <= 195 {
		return UPGRADING_STONE_TYPE
	} else if item.Type == 191 {
		return PENDENT_TYPE
	} else if item.Type == 202 {
		return QUEST_TYPE
	} else if item.Type == 203 {
		return FORTUNE_BOX_TYPE
	} else if item.Type == 221 {
		return PET_TYPE
	} else if item.Type == 222 {
		return PET_POTION_TYPE
	} else if item.Type == 223 {
		return DEAD_SPIRIT_INCENSE_TYPE
	} else if item.Type == 233 {
		return NPC_SUMMONER_TYPE
	} else if item.Type == 226 {
		return CHARM_TYPE
	} else if item.Type == 243 {
		return TOTAL_TRANSFORMATION
	} else if item.Type == 164 {
		return CHARM_OF_LUCK
	} else if item.Type == 219 {
		return BAGUA
	} else if item.Type == 105 {
		return BOW
	} else if item.Type == 106 {
		return ARROW
	} else if item.Type == 171 || item.Type == 228 {
		return SPECIAL_USAGE
	} else if item.Type == 237 {
		return EXPANSION
	} else if item.Type == 248 {
		return ILLUSION_WATER_TYPE
	} else if item.Type == 254 {
		return SPECIAL_USAGE2
	}
	return UNKNOWN_TYPE
}
func GetAllItems() error {
	log.Print("Reading Items table...")

	f, err := excelize.OpenFile("data/tb_ItemTable_Normal.xlsx")
	if err != nil {
		return err
	}
	defer f.Close()
	// Get all the rows in the Sheet1.
	rows, err := f.GetRows("Sheet1")
	if err != nil {
		return err
	}
	for index, row := range rows {
		if index == 0 {
			continue
		}
		item := &Item{
			ID:                       int64(utils.StringToInt(row[1])), //B
			Name:                     row[2],                           //C
			UIF:                      row[4],                           //E
			ItemPair:                 int64(utils.StringToInt(row[19])),
			Type:                     int16(utils.StringToInt(row[22])), //W
			HtType:                   int16(utils.StringToInt(row[23])), //X
			TimerType:                int16(utils.StringToInt(row[27])),
			Timer:                    utils.StringToInt(row[28]),
			MinUpgradeLevel:          uint64(utils.StringToInt(row[30])),
			SpecialItem:              int64(utils.StringToInt(row[38])), //ok
			BuyPrice:                 int64(utils.StringToInt(row[39])), //AN
			SellPrice:                int64(utils.StringToInt(row[40])), //AO
			Slot:                     utils.StringToInt(row[41]),        //AP
			CharacterType:            utils.StringToInt(row[43]),
			MinLevel:                 utils.StringToInt(row[47]), //
			MaxLevel:                 utils.StringToInt(row[48]), //
			BaseDef1:                 utils.StringToInt(row[55]),
			BaseDef2:                 utils.StringToInt(row[56]),
			BaseDef3:                 utils.StringToInt(row[57]),
			BaseMinAtk:               utils.StringToInt(row[58]),
			BaseMaxAtk:               utils.StringToInt(row[59]),
			Range:                    utils.StringToInt(row[61]),
			STR:                      utils.StringToInt(row[64]), //BM
			DEX:                      utils.StringToInt(row[65]), //BN
			INT:                      utils.StringToInt(row[66]), //BO
			Wind:                     utils.StringToInt(row[67]),
			Water:                    utils.StringToInt(row[68]),
			Fire:                     utils.StringToInt(row[69]),
			PoisonATK:                utils.StringToInt(row[70]),
			PoisonDEF:                utils.StringToInt(row[71]),
			PoisonTime:               utils.StringToInt(row[72]),
			ConfusionATK:             utils.StringToInt(row[73]),
			ConfusionDEF:             utils.StringToInt(row[74]),
			ConfusionTime:            utils.StringToInt(row[76]),
			ParaATK:                  utils.StringToInt(row[77]),
			ParaDEF:                  utils.StringToInt(row[78]),
			ParaTime:                 utils.StringToInt(row[80]),
			MaxHp:                    utils.StringToInt(row[83]),
			HPRecoveryRate:           utils.StringToInt(row[84]),
			MaxChi:                   utils.StringToInt(row[85]),
			CHIRecoveryRate:          utils.StringToInt(row[86]),
			RunningSpeed:             utils.StringToFloat64(row[87]),
			MinAtk:                   utils.StringToInt(row[90]),
			MaxAtk:                   utils.StringToInt(row[91]),
			AtkRate:                  utils.StringToInt(row[92]),
			MinArtsAtk:               utils.StringToInt(row[93]),
			MaxArtsAtk:               utils.StringToInt(row[94]),
			ArtsAtkRate:              utils.StringToInt(row[95]),
			Def:                      utils.StringToInt(row[97]),  //CT
			DefRate:                  utils.StringToInt(row[98]),  //
			ArtsDef:                  utils.StringToInt(row[100]), //CW
			ArtsDefRate:              utils.StringToInt(row[101]), //
			Accuracy:                 utils.StringToInt(row[103]),
			Dodge:                    utils.StringToInt(row[104]),
			HpRecovery:               utils.StringToInt(row[105]),
			ChiRecovery:              utils.StringToInt(row[106]),
			ExpRate:                  utils.StringToFloat64(row[113]),
			DropRate:                 utils.StringToFloat64(row[114]),
			Tradable:                 utils.StringToInt(row[119]),
			HolyWaterPlus:            utils.StringToInt(row[124]),
			HolyWaterUpg1:            utils.StringToInt(row[125]),
			HolyWaterUpg2:            utils.StringToInt(row[126]),
			HolyWaterUpg3:            utils.StringToInt(row[127]),
			HolyWaterRate1:           utils.StringToInt(row[128]),
			HolyWaterRate2:           utils.StringToInt(row[129]),
			HolyWaterRate3:           utils.StringToInt(row[130]),
			PVPdmg:                   utils.StringToInt(row[139]),
			PVPsdmg:                  utils.StringToInt(row[140]),
			PVPdefRate:               utils.StringToInt(row[145]),
			PVPsdefRate:              utils.StringToInt(row[146]),
			Buff:                     utils.StringToInt(row[147]),
			BeastType:                utils.StringToInt(row[148]),
			SpecialEffectID:          utils.StringToInt(row[109]),
			SpecialEffectProbability: utils.StringToInt(row[110]),
			SpecialEffectValue:       utils.StringToInt(row[111]),
			CanCreateSocket:          utils.StringToInt(row[54]),
			PvpDampReduction:         0,
			NPCID:                    utils.StringToInt(row[29]),
			Stackable:                false,
		}
		if item.Slot == 11 || item.Slot == 14 || item.Slot == 17 {
			item.Stackable = true
		}
		SetItem(item)
	}
	return nil
}
func GetItemInfoByName(name string) *Item {
	for _, item := range ItemsList {
		if item.Name == name {
			return item
		}
	}
	return nil
}

// Determines if a weapon item can use an action with specified type
func (item *Item) CanUse(t byte) bool {
	if item.Type == int16(t) || t == 0 || t == 1 {
		return true
	} else if (item.Type == 70 || item.Type == 71) && (t == 70 || t == 71) {
		return true
	} else if (item.Type == 102 || item.Type == 103) && (t == 102 || t == 103) {
		return true
	} else if (item.Type == 44 || item.Type == 49) && (t >= 1 && t <= 8) { //BEAST
		return true
	}

	return false
}

func GetBoxContentInfo(boxid int64) (string, string, string) {
	description := ""
	info, ok := GetItemInfo(boxid)
	if info == nil || !ok {
		return "", "", ""
	}
	gambling, ok := GamblingItems[int(boxid)]
	if gambling == nil || !ok {
		return "", "", ""
	}

	drop, ok := GetDropInfo(gambling.DropID)
	if drop == nil || !ok {
		return "", "", ""
	}

	items := drop.Items
	probabilities := drop.Probabilities

	title := fmt.Sprintf("**[%s]** content:\n", info.Name)

	for i := 0; i < len(items)-1; i++ {
		prob := probabilities[i]
		if i > 0 {
			prob = probabilities[i] - probabilities[i-1]
		}
		iteminfo, ok := GetItemInfo(int64(items[i]))
		if !ok || iteminfo == nil {
			continue
		}
		description += fmt.Sprintf("**[%s]** ``(%0.2f%%)``\n", iteminfo.Name, float64(prob)/10)
	}

	return title, description, info.UIF
}
