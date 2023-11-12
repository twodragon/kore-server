package database

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
	"regexp"
	dbg "runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/osamingo/boolconv"
	"github.com/thoas/go-funk"
	gorp "gopkg.in/gorp.v1"
	null "gopkg.in/guregu/null.v3"

	"github.com/twodragon/kore-server/logging"
	"github.com/twodragon/kore-server/messaging"
	"github.com/twodragon/kore-server/nats"

	"github.com/twodragon/kore-server/utils"
)

const (
	BEAST_KING            = 0x32
	EMPRESS               = 0x33
	MONK                  = 0x34
	MALE_BLADE            = 0x35
	FEMALE_BLADE          = 0x36
	AXE                   = 0x38
	FEMALE_ROD            = 0x39
	DUAL_BLADE            = 0x3B
	DIVINE_BEAST_KING     = 0x3C
	DIVINE_EMPRESS        = 0x3D
	DIVINE_MONK           = 0x3E
	DIVINE_MALE_BLADE     = 0x3F
	DIVINE_FEMALE_BLADE   = 0x40
	DIVINE_AXE            = 0x42
	DIVINE_FEMALE_ROD     = 0x43
	DIVINE_DUAL_BLADE     = 0x45
	DARKNESS_BEAST_KING   = 0x46
	DARKNESS_EMPRESS      = 0x47
	DARKNESS_MONK         = 0x48
	DARKNESS_MALE_BLADE   = 0x49
	DARKNESS_FEMALE_BLADE = 0x4A
	DARKNESS_AXE          = 0x4C
	DARKNESS_FEMALE_ROD   = 0x4D
	DARKNESS_DUAL_BLADE   = 0x4F
)

var (
	characters     = make(map[int]*Character)
	characterMutex sync.RWMutex
	GenerateID     func(*Character) error
	GeneratePetID  func(*Character, *PetSlot)

	REFLECTED                   = utils.Packet{0xAA, 0x55, 0x09, 0x00, 0x2F, 0xF1, 0x01, 0x00, 0x32, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}
	DEAL_DAMAGE                 = utils.Packet{0xAA, 0x55, 0x1c, 0x00, 0x16, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}
	DEAL_NORMAL_CRITICAL_DAMAGE = utils.Packet{0xAA, 0x55, 0x1C, 0x00, 0x16, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x55, 0xAA}
	DEAL_SKILL_DAMAGE           = utils.Packet{0xAA, 0x55, 0x1c, 0x00, 0x16, 0x42, 0x75, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}
	DEAL_SKILL_CRITICAL_DAMAGE  = utils.Packet{0xAA, 0x55, 0x1c, 0x00, 0x16, 0x42, 0x75, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x55, 0xAA}
	NOT_ENOUGH_GOLD             = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x59, 0x08, 0xF9, 0x03, 0x55, 0xAA} // not enough gold
	DEAL_POISON_DAMAGE          = utils.Packet{0xAA, 0x55, 0x2e, 0x00, 0x16, 0xFE, 0xFF, 0xFF, 0xFF, 0x01, 0x01, 0x01, 0x00, 0x00, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x1c, 0x57, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x27, 0x00, 0x00, 0x55, 0xAA}

	BAG_EXPANDED  = utils.Packet{0xAA, 0x55, 0x17, 0x00, 0xA3, 0x02, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x30, 0x31, 0x00, 0x55, 0xAA}
	BANK_ITEMS    = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x57, 0x05, 0x01, 0x02, 0x55, 0xAA}
	BANK_EXPANDED = utils.Packet{0xAA, 0x55, 0x17, 0x00, 0xA3, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x30, 0x31, 0x00, 0x55, 0xAA}

	CHARACTER_DIED    = utils.Packet{0xAA, 0x55, 0x02, 0x00, 0x12, 0x01, 0x55, 0xAA}
	CHARACTER_SPAWNED = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x21, 0x01, 0xD7, 0xEF, 0xE6, 0x00, 0x03, 0x01, 0x00, 0x00, 0x00, 0x00, 0xC9, 0x00, 0x00, 0x00,
		0x49, 0x2A, 0xFE, 0x00, 0x20, 0x1C, 0x00, 0x00, 0x02, 0xD2, 0x7E, 0x7F, 0xBF, 0xCD, 0x1A, 0x86, 0x3D, 0x33, 0x33, 0x6B, 0x41, 0xFF, 0xFF, 0x10, 0x27,
		0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0xC4, 0x0E, 0x00, 0x00, 0xC8, 0xBB, 0x30, 0x00, 0x00, 0x03, 0xF3, 0x03, 0x01, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00,
		0x00, 0x10, 0x27, 0x00, 0x00, 0x49, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x64, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x55, 0xAA}
	MEDITATION_MODE = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x82, 0x05, 0x00, 0x55, 0xAA}

	EXP_SKILL_PT_CHANGED = utils.Packet{0xAA, 0x55, 0x0D, 0x00, 0x13, 0x55, 0xAA}

	HP_CHI = utils.Packet{0xAA, 0x55, 0x28, 0x00, 0x16, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x27, 0x00, 0x00, 0x55, 0xAA}

	RESPAWN_COUNTER = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x12, 0x02, 0x01, 0x00, 0x00, 0x55, 0xAA}
	SHOW_ITEMS      = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x59, 0x05, 0x0A, 0x00, 0x00, 0x00, 0x55, 0xAA}

	TELEPORT_PLAYER  = utils.Packet{0xAA, 0x55, 0x09, 0x00, 0x24, 0x55, 0xAA}
	ITEM_COUNT       = utils.Packet{0xAA, 0x55, 0x0C, 0x00, 0x59, 0x04, 0x0A, 0x00, 0x55, 0xAA}
	GREEN_ITEM_COUNT = utils.Packet{0xAA, 0x55, 0x0A, 0x00, 0x59, 0x19, 0x0A, 0x00, 0x55, 0xAA}
	ITEM_EXPIRED     = utils.Packet{0xAA, 0x55, 0x06, 0x00, 0x69, 0x03, 0x55, 0xAA}
	ITEM_ADDED       = utils.Packet{0xaa, 0x55, 0x2e, 0x00, 0x57, 0x0a, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x83, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xaa}
	ITEM_LOOTED      = utils.Packet{0xAA, 0x55, 0x33, 0x00, 0x59, 0x01, 0x0A, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x21, 0x11, 0x55, 0xAA}

	OMOK_WIN    = utils.Packet{0xAA, 0x55, 0x08, 0x00, 0xa4, 0x04, 0x0A, 0x00, 0x00, 0x00, 0x55, 0xAA}
	PTS_CHANGED = utils.Packet{0xAA, 0x55, 0x06, 0x00, 0xA2, 0x04, 0x55, 0xAA}
	GOLD_LOOTED = utils.Packet{0xAA, 0x55, 0x0D, 0x00, 0x59, 0x01, 0x0A, 0x00, 0x02, 0x55, 0xAA}
	GET_GOLD    = utils.Packet{0xAA, 0x55, 0x12, 0x00, 0x63, 0x01, 0x55, 0xAA}

	MAP_CHANGED = utils.Packet{0xAA, 0x55, 0x02, 0x00, 0x2B, 0x01, 0x55, 0xAA, 0xAA, 0x55, 0x0E, 0x00, 0x73, 0x00, 0x00, 0x00, 0x7A, 0x44, 0x55, 0xAA,
		0xAA, 0x55, 0x07, 0x00, 0x01, 0xB9, 0x0A, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA, 0xAA, 0x55, 0x09, 0x00, 0x24, 0x55, 0xAA,
		0xAA, 0x55, 0x03, 0x00, 0xA6, 0x00, 0x00, 0x55, 0xAA, 0xAA, 0x55, 0x02, 0x00, 0xAD, 0x00, 0x55, 0xAA}

	DARK_MODE_ACTIVE = utils.Packet{0xaa, 0x55, 0x0a, 0x00, 0xad, 0x02, 0x9a, 0x99, 0x99, 0x3f, 0x66, 0x66, 0x66, 0x00, 0x55, 0xaa}

	ITEM_REMOVED = utils.Packet{0xAA, 0x55, 0x0B, 0x00, 0x59, 0x02, 0x0A, 0x00, 0x01, 0x55, 0xAA}
	SELL_ITEM    = utils.Packet{0xAA, 0x55, 0x16, 0x00, 0x58, 0x02, 0x0A, 0x00, 0x20, 0x1C, 0x00, 0x00, 0x55, 0xAA}

	GET_STATS = utils.Packet{0xAA, 0x55, 0xd2, 0x00, 0x14, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xA0, 0x40, 0x05,
		0x00, 0x00, 0x00, 0x00, 0x40, 0x05, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00,
		0x00, 0x64, 0x00, 0x74, 0x04, 0x00, 0x00, 0x88, 0x00, 0x00, 0x00, 0x74, 0x00, 0x00, 0x00, 0xB8,
		0x1E, 0x85, 0x3F, 0x00, 0x00, 0x80, 0x3f, 0x00, 0x00, 0x55, 0xAA}

	ITEM_REPLACEMENT = utils.Packet{0xAA, 0x55, 0x0C, 0x00, 0x59, 0x03, 0x0A, 0x00, 0x55, 0xAA}
	ITEM_SWAP        = utils.Packet{0xAA, 0x55, 0x15, 0x00, 0x59, 0x07, 0x0A, 0x00, 0x00, 0x55, 0xAA}
	HT_UPG_FAILED    = utils.Packet{0xAA, 0x55, 0x31, 0x00, 0x54, 0x02, 0xA7, 0x0F, 0x01, 0x00, 0xA3, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}
	UPG_FAILED       = utils.Packet{0xAA, 0x55, 0x09, 0x00, 0x54, 0x02, 0xA2, 0x0F, 0x00, 0x55, 0xAA}
	UPG_SUCCESS      = utils.Packet{0xAA, 0x55, 0x09, 0x00, 0x54, 0x02, 0xA2, 0x0F, 0x01, 0x55, 0xAA}

	PRODUCTION_SUCCESS = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x54, 0x04, 0x08, 0x10, 0x01, 0x55, 0xAA}
	PRODUCTION_FAILED  = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x54, 0x04, 0x09, 0x10, 0x00, 0x55, 0xAA}
	PRODUCTION_ERROR   = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x54, 0x04, 0x07, 0x10, 0x55, 0xAA}

	ENCHANT_SUCCESS = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x54, 0x07, 0x08, 0x10, 0x01, 0x55, 0xAA}
	ENCHANT_FAILED  = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x54, 0x07, 0x09, 0x10, 0x00, 0x55, 0xAA}
	ENCHANT_ERROR   = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x54, 0x07, 0x07, 0x10, 0x55, 0xAA}

	FUSION_SUCCESS = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x54, 0x09, 0x10, 0x10, 0x01, 0x55, 0xAA}
	FUSION_FAILED  = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x54, 0x09, 0x11, 0x10, 0x00, 0x55, 0xAA}

	DISMANTLE_SUCCESS  = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x54, 0x05, 0x68, 0x10, 0x01, 0x00, 0x55, 0xAA}
	EXTRACTION_SUCCESS = utils.Packet{0xAA, 0x55, 0xB7, 0x00, 0x54, 0x06, 0xCC, 0x10, 0x01, 0x00, 0xA2, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}

	HOLYWATER_FAILED  = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x54, 0x10, 0x32, 0x11, 0x00, 0x55, 0xAA}
	HOLYWATER_SUCCESS = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x54, 0x10, 0x31, 0x11, 0x01, 0x55, 0xAA}

	ITEM_REGISTERED = utils.Packet{0xAA, 0x55, 0x43, 0x00, 0x3D, 0x01, 0x0A, 0x00, 0x00, 0x80, 0x1A, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0D, 0x00,
		0x00, 0x00, 0x63, 0x99, 0xEA, 0x00, 0x00, 0xA1, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}
	CLAIM_MENU              = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x3D, 0x03, 0x0A, 0x00, 0x55, 0xAA}
	CONSIGMENT_ITEM_BOUGHT  = utils.Packet{0xAA, 0x55, 0x08, 0x00, 0x3D, 0x02, 0x0A, 0x00, 0x55, 0xAA}
	CONSIGMENT_ITEM_SOLD    = utils.Packet{0xAA, 0x55, 0x02, 0x00, 0x3F, 0x00, 0x55, 0xAA}
	CONSIGMENT_ITEM_CLAIMED = utils.Packet{0xAA, 0x55, 0x0A, 0x00, 0x3D, 0x04, 0x0A, 0x00, 0x01, 0x00, 0x55, 0xAA}
	SKILL_UPGRADED          = utils.Packet{0xAA, 0x55, 0x0B, 0x00, 0x81, 0x02, 0x0A, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}
	SKILL_DOWNGRADED        = utils.Packet{0xAA, 0x55, 0x0E, 0x00, 0x81, 0x03, 0x0A, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}
	SKILL_REMOVED           = utils.Packet{0xAA, 0x55, 0x09, 0x00, 0x81, 0x06, 0x0A, 0x00, 0x00, 0x55, 0xAA}
	PASSIVE_SKILL_UGRADED   = utils.Packet{0xAA, 0x55, 0x06, 0x00, 0x82, 0x02, 0x0A, 0x00, 0x00, 0x00, 0x55, 0xAA}
	PASSIVE_SKILL_REMOVED   = utils.Packet{0xAA, 0x55, 0x09, 0x00, 0x82, 0x04, 0x0A, 0x00, 0x00, 0x55, 0xAA}
	SKILL_CASTED            = utils.Packet{0xAA, 0x55, 0x1D, 0x00, 0x42, 0x0A, 0x00, 0x00, 0x00, 0x01, 0x01, 0x55, 0xAA}
	TRADE_CANCELLED         = utils.Packet{0xAA, 0x55, 0x06, 0x00, 0x53, 0x03, 0xD5, 0x07, 0x7E, 0x02, 0x55, 0xAA}
	SKILL_BOOK_EXISTS       = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x59, 0x04, 0xFA, 0x03, 0x55, 0xAA}
	INVALID_CHARACTER_TYPE  = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x59, 0x04, 0xF2, 0x03, 0x55, 0xAA}
	NO_SLOTS_FOR_SKILL_BOOK = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x59, 0x04, 0xF3, 0x03, 0x55, 0xAA}
	OPEN_SALE               = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x55, 0x01, 0x0A, 0x00, 0x55, 0xAA}
	GET_SALE_ITEMS          = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x55, 0x03, 0x0A, 0x00, 0x00, 0x55, 0xAA}
	CLOSE_SALE              = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x55, 0x02, 0x0A, 0x00, 0x55, 0xAA}
	BOUGHT_SALE_ITEM        = utils.Packet{0xAA, 0x55, 0x39, 0x00, 0x53, 0x10, 0x0A, 0x00, 0x01, 0x00, 0xA2, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}
	SOLD_SALE_ITEM          = utils.Packet{0xAA, 0x55, 0x10, 0x00, 0x55, 0x07, 0x0A, 0x00, 0x55, 0xAA}
	BUFF_INFECTION          = utils.Packet{0xAA, 0x55, 0x0C, 0x00, 0x4D, 0x02, 0x0A, 0x01, 0x55, 0xAA}
	BUFF_EXPIRED            = utils.Packet{0xAA, 0x55, 0x06, 0x00, 0x4D, 0x03, 0x55, 0xAA}

	COOKING_SUCCESS = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x54, 0x09, 0x12, 0x10, 0x01, 0x55, 0xAA}
	COOKING_ERROR   = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x54, 0x09, 0x13, 0x10, 0x01, 0x55, 0xAA}
	COOKING_FAILED  = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x54, 0x09, 0x14, 0x10, 0x01, 0x55, 0xAA}

	REPURCHASE_LIST = utils.Packet{0xAA, 0x55, 0x2F, 0x00, 0xAF, 0x02, 0x01, 0x55, 0xAA}
	TRASH_LIST      = utils.Packet{0xAA, 0x55, 0x2F, 0x00, 0xAF, 0x04, 0x01, 0x55, 0xAA}

	DEAL_BUFF_AI = utils.Packet{0xaa, 0x55, 0x1e, 0x00, 0x16, 0x00, 0x00, 0x00, 0x00, 0x01, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xaa}

	SPLIT_ITEM = utils.Packet{0xAA, 0x55, 0x5C, 0x00, 0x59, 0x09, 0x0A, 0x00, 0x00, 0xA1, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xA1, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}

	MAP_BOOK_SHOW = utils.Packet{0xAA, 0x55, 0x2E, 0x00, 0x57, 0x0A, 0x06, 0xEF, 0xE7, 0x00, 0x00, 0xAA, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA, 0xAA, 0x55, 0x04, 0x00, 0x59, 0x24, 0x0A, 0x00, 0x55, 0xAA}

	RELIC_DROP       = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x71, 0x10, 0x00, 0x55, 0xAA}
	PVP_FINISHED     = utils.Packet{0xAA, 0x55, 0x02, 0x00, 0x2A, 0x05, 0x55, 0xAA}
	FORM_ACTIVATED   = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x37, 0x55, 0xAA}
	FORM_DEACTIVATED = utils.Packet{0xAA, 0x55, 0x01, 0x00, 0x38, 0x55, 0xAA}
	CHANGE_RANK      = utils.Packet{0xAA, 0x55, 0x09, 0x00, 0x2F, 0xF1, 0x36, 0x55, 0xAA}

	QUEST_HANDLER = utils.Packet{0xaa, 0x55, 0x30, 0x00, 0x57, 0x40, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xaa}

	InvisibilitySkillIDs = []int{241, 244, 58, 72, 73, 70}

	Beast_King_Infections = []int16{277, 307, 368, 283, 319, 382, 291, 333, 398, 297, 351, 418}
	Empress_Infections    = []int16{280, 313, 375, 287, 326, 390, 294, 342, 408, 302, 359, 429}

	EFFECT_ALREADY_EXIST = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x59, 0x04, 0xFA, 0x03, 0x55, 0xAA}

	ARRANGE_ITEM = utils.Packet{0xAA, 0x55, 0x32, 0x00, 0x78, 0x02, 0x00, 0xA2, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}

	ARRANGE_BANK_ITEM = utils.Packet{0xAA, 0x55, 0x2F, 0x00, 0x80, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}

	RELAXATION_INFO = utils.Packet{0xaa, 0x55, 0x0c, 0x00, 0xac, 0x08, 0x0a, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xaa}
)

type Target struct {
	Damage  int `db:"-" json:"damage"`
	SkillId int `db:"-" json:"skillid"`
	AI      *AI `db:"-" json:"ai"`
}

type PlayerTarget struct {
	Damage  int        `db:"-" json:"damage"`
	Enemy   *Character `db:"-" json:"ai"`
	SkillId int        `db:"-" json:"skillid"`
}

type AidSettings struct {
	PetFood1ItemID  int64 `db:"-" json:"petfood1"`
	PetFood1Percent uint  `db:"-" json:"petfood1percent"`
	PetChiItemID    int64 `db:"-" json:"petchi"`
	PetChiPercent   uint  `db:"-" json:"petchipercent"`
}
type MessageItems struct {
	ID     int   `db:"-" json:"id"`
	SlotID int   `db:"-" json:"slotid"`
	ItemID int64 `db:"-" json:"itemid"`
}

type groupSettings struct {
	ExperienceSharingMethod int
	LootDistriburionMethod  int
}

type Character struct {
	ID           int     `db:"id" json:"id"`
	UserID       string  `db:"user_id" json:"user_id"`
	Name         string  `db:"name" json:"name"`
	Epoch        int64   `db:"epoch" json:"epoch"`
	Type         int     `db:"type" json:"type"`
	Faction      int     `db:"faction" json:"faction"`
	Height       int     `db:"height" json:"height"`
	Gold         uint64  `db:"gold" json:"gold"`
	Exp          int64   `db:"exp" json:"exp"`
	Level        int     `db:"level" json:"level"`
	Class        int     `db:"class" json:"class"`
	IsOnline     bool    `db:"is_online" json:"is_online"`
	IsActive     bool    `db:"is_active" json:"is_active"`
	Coordinate   string  `db:"coordinate" json:"coordinate"`
	Map          int16   `db:"map" json:"map"`
	HTVisibility int     `db:"ht_visibility" json:"ht_visibility"`
	WeaponSlot   int     `db:"weapon_slot" json:"weapon_slot"`
	RunningSpeed float64 `db:"running_speed" json:"running_speed"`
	GuildID      int     `db:"guild_id" json:"guild_id"`

	AntiExp bool `db:"-"`

	Slotbar             []byte    `db:"slotbar" json:"slotbar"`
	CreatedAt           null.Time `db:"created_at" json:"created_at"`
	AidMode             bool      `db:"aid_mode" json:"aid_mode"`
	AidTime             uint32    `db:"aid_time" json:"aid_time"`
	Injury              float64   `db:"injury" json:"injury"`
	HeadStyle           int64     `db:"headstyle" json:"headstyle"`
	FaceStyle           int64     `db:"facestyle" json:"facestyle"`
	Reborns             int32     `db:"reborns" json:"reborns"`
	HonorRank           int64     `db:"rank" json:"rank"`
	YingYangTicketsLeft int       `db:"ying_yang_tickets" json:"ying_yang_tickets"`
	Relaxation          int       `db:"relaxation"`
	OnlineHours         int       `db:"online_hours"`

	ShowUpgradingRate bool `db:"-" json:"-"`

	BoxOpenerBank []*InventorySlot `db:"-" json:"-"`

	CastSkillMutex  sync.RWMutex `db:"-" json:"-"`
	ConsumableMutex sync.RWMutex `db:"-" json:"-"`
	InvMutex        sync.Mutex   `db:"-"`
	GoldMutex       sync.Mutex   `db:"-" json:"-"`
	BoxOpenerMutex  sync.RWMutex `db:"-"`
	CheckInMutex    sync.Mutex   `db:"-"`
	ExpMutex        sync.Mutex   `db:"-"`
	Looting         sync.Mutex   `db:"-"`

	Socket        *Socket   `db:"-" json:"-"`
	ExploreWorld  func()    `db:"-" json:"-"`
	HasLot        bool      `db:"-" json:"-"`
	LastRoar      time.Time `db:"-" json:"-"`
	Meditating    bool      `db:"-"`
	MovementToken int64     `db:"-" json:"-"`
	PseudoID      uint16    `db:"-" json:"pseudo_id"`
	PTS           int       `db:"-" json:"pts"`
	OnSight       struct {
		Drops             map[int]interface{} `db:"-" json:"drops"`
		DropsMutex        sync.RWMutex
		Mobs              map[int]interface{} `db:"-" json:"mobs"`
		MobMutex          sync.RWMutex        `db:"-"`
		NPCs              map[int]interface{} `db:"-" json:"npcs"`
		NpcMutex          sync.RWMutex        `db:"-"`
		Pets              map[int]interface{} `db:"-" json:"pets"`
		PetsMutex         sync.RWMutex        `db:"-"`
		Players           map[int]interface{} `db:"-" json:"players"`
		PlayerMutex       sync.RWMutex        `db:"-"`
		BabyPets          map[int]interface{} `db:"-" json:"BabyPets"`
		BabyPetsMutex     sync.RWMutex        `db:"-"`
		Housingitems      map[int]interface{} `db:"-" `
		HousingitemsMutex sync.RWMutex        `db:"-"`
	} `db:"-" json:"on_sight"`

	AttackMutex sync.Mutex `db:"-" json:"-"`
	AttackDelay int64      `db:"-" json:"-"`

	PartyID       string          `db:"-"`
	Selection     int             `db:"-" json:"selection"`
	Targets       []*Target       `db:"-" json:"target"`
	TamingAI      *AI             `db:"-" json:"-"`
	PlayerTargets []*PlayerTarget `db:"-" json:"player_targets"`

	ClanGoldDonation uint64 `db:"clan_gold_donation" json:"-"`
	//MobsAttacking []*AI           `db:"-" json:"-"`

	TradeID       string `db:"-" json:"trade_id"`
	Invisible     bool   `db:"-" json:"-"`
	DetectionMode bool   `db:"-" json:"-"`
	VisitedSaleID uint16 `db:"-" json:"-"`
	DuelID        int    `db:"-" json:"-"`
	DuelStarted   bool   `db:"-" json:"-"`
	Respawning    bool   `db:"-" json:"-"`

	IsinWar bool `db:"-"`

	SkillHistory      utils.SMap   `db:"-"`
	Morphed           bool         `db:"-"`
	MorphedNPCID      int          `db:"-"`
	IsDungeon         bool         `db:"-"`
	DungeonLevel      int16        `db:"-"`
	CanTip            int16        `db:"-"`
	GeneratedNumber   int          `db:"-"`
	HandlerCB         func()       `db:"-"`
	PetHandlerCB      func()       `db:"-"`
	MapBook           string       `db:"map_book"`
	Poisoned          bool         `db:"-"`
	Paralised         bool         `db:"-"`
	Confused          bool         `db:"-"`
	PlayerAidSettings *AidSettings `db:"-"`
	KilledByCharacter *Character   `db:"-"`

	inventory      []*InventorySlot `db:"-"`
	RepurchaseList RepurchaseList   `db:"-"`

	WarKillCount    int  `db:"-"`
	WarContribution int  `db:"-"`
	PartyMode       int  `db:"-"`
	IsAcceptedWar   bool `db:"-"`
	//	IsQuestMenuOpened bool    `db:"-"`
	//	QuestActions      []int   `db:"-"`
	LastNPCAction int64   `db:"-"`
	InjuryCount   float64 `db:"-"`
	//	questMobsIDs  []int   `db:"-"`

	AidStartingPosition string          `db:"-"`
	MessageItems        []*MessageItems `db:"-"`

	IsMounting           bool `db:"-"`
	HaveItemsMovedForWar bool `db:"-"`

	CanRun         bool `db:"-"`
	DarkModeActive bool `db:"-"`

	HpRecoveryCooldown int `db:"-"`
	TypeOfBankOpened   int `db:"-"`

	DowngradingSkillWarningMessageShowed bool `db:"-"`

	GroupSettings groupSettings `db:"-"`

	Stunned          bool   `db:"-"`
	CanMove          bool   `db:"-"`
	BattleMode       int    `db:"-"`
	Opponent         uint16 `db:"-"`
	OmokID           int    `db:"-"`
	OmokRequestState int    `db:"-"` // 0: not requested, 1: requested, 2: accepted

	OpenerBoost int `db:"-"`

	AntiDupeMutex   sync.RWMutex `db:"-"`
	ArrangeCooldown uint16       `db:"-"`
	KilledMobs      int          `db:"-"`

	House *HousingItem `db:"-"`

	CountOnlineHours func() `db:"-"`
}

func ConvertPointToCoordinate(X float64, Y float64) string {

	str := fmt.Sprintf("%.1f,%.1f", X, Y)

	return str
}
func (t *Character) PreInsert(s gorp.SqlExecutor) error {
	now := time.Now().UTC()
	t.CreatedAt = null.TimeFrom(now)
	//t.CreatedAt = time.Now().Format("2006-01-02 15:04:05")
	return nil
}

func (t *Character) SetCoordinate(coordinate *utils.Location) {
	t.Coordinate = fmt.Sprintf("(%.1f,%.1f)", coordinate.X, coordinate.Y)
}

func (t *Character) Create() error {
	return pgsql_DbMap.Insert(t)
}

func (t *Character) CreateWithTransaction(tr *gorp.Transaction) error {
	return tr.Insert(t)
}

func (c *Character) Update() error {

	_, err := pgsql_DbMap.Update(c)
	if err != nil {
		log.Println(err)
	}
	return err
}

func (t *Character) Delete() error {
	characterMutex.Lock()
	defer characterMutex.Unlock()

	delete(characters, t.ID)
	_, err := pgsql_DbMap.Delete(t)
	return err
}

func (c *Character) BoxOpenerStorage() ([]*InventorySlot, error) {
	c.BoxOpenerBank = make([]*InventorySlot, 450)
	for i := range c.BoxOpenerBank {
		c.BoxOpenerBank[i] = NewSlot()
	}
	boxOpenerBank, err := FindBoxOpenerStorageByUserId(c.UserID)
	if err != nil {
		return nil, err
	}
	for _, s := range boxOpenerBank {
		c.BoxOpenerBank[s.SlotID] = s
	}

	return c.BoxOpenerBank, nil
}

func (t *Character) InventorySlots() ([]*InventorySlot, error) {

	if len(t.inventory) > 0 {
		return t.inventory, nil
	}

	inventory := make([]*InventorySlot, 450)

	for i := range inventory {
		inventory[i] = NewSlot()
	}

	slots, err := FindInventorySlotsByCharacterID(t.ID)
	if err != nil {
		return nil, err
	}

	bankSlots, err := FindBankSlotsByUserID(t.UserID)
	if err != nil {
		return nil, err
	}

	for _, s := range slots {
		inventory[s.SlotID] = s
	}

	for _, s := range bankSlots {
		inventory[s.SlotID] = s
	}

	t.inventory = inventory
	return inventory, nil
}

func (t *Character) SetInventorySlots(slots []*InventorySlot) { // FIX HERE
	t.inventory = slots
}

func (t *Character) CopyInventorySlots() []*InventorySlot {
	slots := []*InventorySlot{}
	for _, s := range t.inventory {
		copySlot := *s
		slots = append(slots, &copySlot)
	}

	return slots
}

func RefreshAIDs() error {
	query := `update characters SET aid_time = 7200 WHERE aid_time < 7200`
	_, err := pgsql_DbMap.Exec(query)
	if err != nil {
		return err
	}

	characterMutex.RLock()
	allChars := funk.Values(characters).([]*Character)
	characterMutex.RUnlock()
	for _, c := range allChars {
		if c.AidTime < 7200 {
			c.AidTime = 7200
		}
	}

	return err
}

func FindCharactersByUserID(userID string) ([]*Character, error) {
	characterMutex.RLock()

	charMap := make(map[int]*Character)
	for _, c := range characters {
		if c.UserID == userID {
			charMap[c.ID] = c
		}
	}
	characterMutex.RUnlock()
	var arr []*Character
	query := `SELECT * FROM hops.characters WHERE user_id = $1`
	_, err := pgsql_DbMap.Select(&arr, query, userID)
	if err != nil {
		log.Printf("Error retrieving characters: %v", err)
	}
	characterMutex.Lock()
	defer characterMutex.Unlock()

	var chars []*Character
	for _, c := range arr {
		char, ok := charMap[c.ID]
		if ok {
			chars = append(chars, char)
		} else {
			characters[c.ID] = c
			chars = append(chars, c)
		}
	}

	return chars, nil
}
func IsValidUsername(name string) (bool, error) {

	var (
		count int64
		err   error
		query string
	)

	re := regexp.MustCompile("^[a-zA-Z0-9]{4,18}$")
	if !re.MatchString(name) {
		return false, nil
	}

	query = `select count(*) from hops.characters where lower(name) = $1`

	if count, err = pgsql_DbMap.SelectInt(query, strings.ToLower(name)); err != nil {
		return false, fmt.Errorf("IsValidUsername: %s", err.Error())
	}

	return count == 0, nil
}

func FindCharacterByName(name string) (*Character, error) {

	for _, c := range characters {
		if c.Name == name {
			return c, nil
		}
	}

	character := &Character{}
	err := pgsql_DbMap.SelectOne(&character, "select * from hops.users where login = $1", name)
	if err != nil {
		return nil, err
	}

	characterMutex.Lock()
	defer characterMutex.Unlock()

	characterMutex.Lock()

	characters[character.ID] = character
	characterMutex.Unlock()
	return character, nil
}

func FindAllCharacter() ([]*Character, error) {

	charMap := make(map[int]*Character)

	var arr []*Character

	query := `select * from hops.characters`

	_, err := pgsql_DbMap.Select(&arr, query)
	if err != nil {
		return nil, err
	}
	characterMutex.Lock()
	defer characterMutex.Unlock()
	var chars []*Character
	for _, c := range arr {
		char, ok := charMap[c.ID]
		if ok {
			chars = append(chars, char)
		} else {
			characters[c.ID] = c
			chars = append(chars, c)
		}
	}

	return chars, nil
}

func FindCharacterByID(id int) (*Character, error) {

	characterMutex.RLock()
	c, ok := characters[id]
	characterMutex.RUnlock()

	if ok {
		return c, nil
	}

	character := &Character{}

	err := pgsql_DbMap.SelectOne(&character, "select * from hops.characters where id = $1", id)
	if err != nil {
		return nil, err
	}

	characterMutex.Lock()
	defer characterMutex.Unlock()
	characters[character.ID] = character

	return character, nil
}

func (c *Character) GetAppearingItemSlots() []int {

	helmSlot := 0
	if c.HTVisibility&0x01 != 0 {
		helmSlot = 0x0133
	}

	maskSlot := 1
	if c.HTVisibility&0x02 != 0 {
		maskSlot = 0x0134
	}

	armorSlot := 2
	if c.HTVisibility&0x04 != 0 {
		armorSlot = 309
	}

	bootsSlot := 9
	if c.HTVisibility&0x10 != 0 {
		bootsSlot = 0x0136
	}

	armorSlot2 := 2
	if c.HTVisibility&0x08 != 0 {
		armorSlot2 = 311
	}

	if armorSlot2 != 2 {
		armorSlot = armorSlot2
	}

	return []int{helmSlot, maskSlot, armorSlot, 3, 4, 5, 6, 7, 8, bootsSlot, 10}
}

func (c *Character) GetEquipedItemSlots() []int {
	return []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 307, 309, 310, 312, 313, 314, 315}
}

func (c *Character) Logout() {
	friends, _ := FindAllFriendsIDsByCharacterID(c.ID)
	for _, friend := range friends {
		char, err := FindCharacterByID(friend.CharacterID)
		if err != nil {
			continue
		}
		if char == nil {
			continue
		}
		index := 6
		resp := MODIFY_FRIEND
		resp.Insert(utils.IntToBytes(uint64(friend.ID), 4, true), index)
		index += 4
		online, err := boolconv.NewBoolByInterface(false)
		if err != nil {
			log.Println("error should not be nil")
		}
		resp.Overwrite(online.Bytes(), index)
		resp.SetLength(int16(binary.Size(resp) - 6))

		if char.Socket != nil {
			char.Socket.Write(resp)
		}
	}

	c.IsOnline = false
	c.IsActive = false
	c.IsDungeon = false
	c.OnSight.Drops = map[int]interface{}{}
	c.OnSight.Mobs = map[int]interface{}{}
	c.OnSight.NPCs = map[int]interface{}{}
	c.OnSight.Pets = map[int]interface{}{}
	c.OnSight.Players = map[int]interface{}{}
	c.OnSight.BabyPets = map[int]interface{}{}
	c.ExploreWorld = nil
	c.HandlerCB = nil
	c.PetHandlerCB = nil
	c.PTS = 0
	c.TradeID = ""
	c.LeaveParty()
	c.EndPvP()
	sale := FindSale(c.PseudoID)
	if sale != nil {
		sale.Delete()
	}

	if trade := FindTrade(c); trade != nil {
		c.CancelTrade()
	}

	if c.GuildID > 0 {
		guild, err := FindGuildByID(c.GuildID)
		if err == nil && guild != nil {
			guild.InformMembers(c)
		}
	}

	RemoveFromRegister(c)
	RemovePetFromRegister(c)
	DeleteCharacterFromCache(c.ID)
	//DeleteStatFromCache(c.ID)
	c.Socket.User.Update()
	c.Update()
}

func (c *Character) EndPvP() {
	if c.DuelID > 0 {
		op, _ := FindCharacterByID(c.DuelID)
		if op != nil {
			op.Socket.Write(PVP_FINISHED)
			op.DuelID = 0
			op.DuelStarted = false
		}
		c.DuelID = 0
		c.DuelStarted = false
		c.Socket.Write(PVP_FINISHED)
	}
}

func DeleteCharacterFromCache(id int) {

	characterMutex.Lock()
	delete(characters, id)
	characterMutex.Unlock()
}

func (c *Character) GetNearbyCharacters() ([]*Character, error) {

	var (
		distance = float64(50)
	)

	u, err := FindUserByID(c.UserID)
	if err != nil {
		return nil, err
	}

	myCoordinate := ConvertPointToLocation(c.Coordinate)
	characterMutex.RLock()
	allChars := funk.Values(characters)
	characterMutex.RUnlock()
	characters := funk.Filter(allChars, func(character *Character) bool {

		user, err := FindUserByID(character.UserID)
		if err != nil || user == nil {
			return false
		}
		characterCoordinate := ConvertPointToLocation(character.Coordinate)

		return character.IsOnline && user.ConnectedServer == u.ConnectedServer && character.Map == c.Map &&
			(!character.Invisible || c.DetectionMode) && utils.CalculateDistance(characterCoordinate, myCoordinate) <= distance
	}).([]*Character)

	return characters, nil
}

func (c *Character) GetNearbyAIIDs() ([]int, error) {

	var (
		distance = 50.0
		ids      []int
	)

	user, err := FindUserByID(c.UserID)
	if err != nil {
		return nil, err
	} else if user == nil {
		return nil, nil
	}

	if AIsByMap[user.ConnectedServer] == nil {

		return ids, nil
	}

	candidates, ok := AIsByMap[user.ConnectedServer][c.Map]
	if !ok {
		return ids, nil
	}
	filtered := funk.Filter(candidates, func(ai *AI) bool {

		characterCoordinate := ConvertPointToLocation(c.Coordinate)
		aiCoordinate := ConvertPointToLocation(ai.Coordinate)

		return utils.CalculateDistance(characterCoordinate, aiCoordinate) <= distance
	})

	for _, ai := range filtered.([]*AI) {
		ids = append(ids, ai.ID)
	}

	return ids, nil
}

func (c *Character) GetNearbyNPCIDs() ([]int, error) {

	var (
		distance = 50.0
		ids      []int
	)

	user, err := FindUserByID(c.UserID)
	if err != nil {
		return nil, err
	} else if user == nil {
		return nil, nil
	}

	filtered := make([]*NpcPosition, 0)
	for _, pos := range GetNPCPostions() {
		characterCoordinate := ConvertPointToLocation(c.Coordinate)
		minLocation := ConvertPointToLocation(pos.MinLocation)
		maxLocation := ConvertPointToLocation(pos.MaxLocation)

		npcCoordinate := &utils.Location{X: (minLocation.X + maxLocation.X) / 2, Y: (minLocation.Y + maxLocation.Y) / 2}

		if c.Map == pos.MapID && utils.CalculateDistance(characterCoordinate, npcCoordinate) <= distance && pos.IsNPC && !pos.Attackable {
			filtered = append(filtered, pos)
		}
	}

	for _, pos := range filtered {
		ids = append(ids, pos.ID)
	}

	return ids, nil
}

func (c *Character) GetNearbyDrops() ([]int, error) {

	var (
		distance = 50.0
		ids      []int
	)

	user, err := FindUserByID(c.UserID)
	if err != nil {
		return nil, err
	} else if user == nil {
		return nil, nil
	}

	allDrops := GetDropsInMap(user.ConnectedServer, c.Map)
	filtered := funk.Filter(allDrops, func(drop *Drop) bool {

		characterCoordinate := ConvertPointToLocation(c.Coordinate)

		return utils.CalculateDistance(characterCoordinate, &drop.Location) <= distance
	})

	for _, d := range filtered.([]*Drop) {
		ids = append(ids, d.ID)
	}

	return ids, nil
}

func (c *Character) SpawnCharacter() ([]byte, error) {

	if c == nil {
		return nil, nil
	}
	if c.Socket == nil || c.Socket.Stats == nil {
		return nil, nil
	}

	resp := CHARACTER_SPAWNED
	index := 6
	resp.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), index) // character pseudo id
	index += 2
	resp.Insert([]byte{0xee, 0x22, 0x00, 0x00}, index)
	index += 4
	if c.IsActive {
		resp.Insert([]byte{0x03, 0x00, 0x00, 0x00, 0x00}, index)
	} else {
		resp.Insert([]byte{0x04, 0x00, 0x00, 0x00, 0x00}, index)
	}
	index += 5

	if c.DuelID > 0 {
		resp.Overwrite(utils.IntToBytes(500, 2, true), 13) // duel state
	}

	resp.Insert(utils.IntToBytes(uint64(len(c.Name)), 1, true), index)
	index++
	resp.Insert([]byte(c.Name), index) // character name
	index += len(c.Name)
	resp.Insert(utils.IntToBytes(uint64(c.Level), 4, true), index)
	index += 4
	resp.Insert([]byte{byte(c.Type), byte(c.Class)}, index) // character type-class
	index += 2
	resp.Insert([]byte{0x01, 0x00, 0x20, 0x1c, 0x00, 0x00, 0x00}, index)
	index += 7

	coordinate := ConvertPointToLocation(c.Coordinate)
	resp.Insert(utils.FloatToBytes(coordinate.X, 4, true), index) // coordinate-x
	index += 4

	resp.Insert(utils.FloatToBytes(coordinate.Y, 4, true), index) // coordinate-y
	index += 4

	resp.Insert([]byte{0x00, 0x00, 0x60, 0x41}, index)
	index += 4

	resp.Insert(utils.FloatToBytes(coordinate.X, 4, true), index) // coordinate-x
	index += 4

	resp.Insert(utils.FloatToBytes(coordinate.Y, 4, true), index) // coordinate-y
	index += 4
	resp.Insert([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff}, index)
	index += 10
	resp.Insert(utils.IntToBytes(uint64(c.Socket.Stats.Honor), 4, true), index) // HONOR
	index += 4
	resp.Insert([]byte{0xc8, 0x00, 0x00, 0x00}, index)
	index += 4

	resp.Insert(utils.IntToBytes(uint64(c.Socket.Stats.HP), 4, true), index) // hp
	index += 4
	resp.Insert([]byte{0x00, 0x00, 0x00, 0x00, 0x00}, index)
	index += 5
	resp.Insert(utils.IntToBytes(uint64(c.WeaponSlot), 1, true), index)
	index++
	resp.Insert([]byte{0xf2, 0x03}, index)
	index += 2
	resp.Insert(utils.IntToBytes(uint64(c.BattleMode), 1, true), index) //battle mode
	index++
	resp.Insert([]byte{0x00, 0x00, 0x05}, index)
	index += 3
	if c.Morphed {
		resp.Insert(utils.IntToBytes(uint64(c.MorphedNPCID), 4, true), index)
		index += 4
	} else {
		resp.Insert([]byte{0x00, 0x00, 0x00, 0x00}, index)
		index += 4
	}
	//index += 5
	resp.Insert(utils.IntToBytes(uint64(c.HonorRank), 4, true), index) // TITLE
	index += 4
	resp.Insert(utils.IntToBytes(uint64(c.Type), 1, true), index)
	index++
	resp.Insert(utils.IntToBytes(uint64(c.GuildID), 4, true), index) // guild id
	index += 4
	resp.Insert([]byte{0x01, 0x00, 0x00, 0x00}, index)
	index += 4
	resp.Insert([]byte{byte(c.Faction)}, index) // character faction
	index++
	resp.Insert([]byte{0x00, 0x00, 0x00, 0x00, 0x64, 0xff, 0xff, 0xff, 0xff}, index)
	index += 9
	items, err := c.ShowItemsByCharacter()
	if err != nil {
		return nil, err
	}

	itemsData := items
	sale := FindSale(c.PseudoID)
	if sale != nil {
		itemsData = []byte{0x05, 0xAA, 0x45, 0xF1, 0x00, 0x00, 0x00, 0xA1, 0x00, 0x00, 0x00, 0x00, 0xB4, 0x6C, 0xF1, 0x00, 0x01, 0x00, 0xA1, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x09, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0A, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	}
	//myString := hex.EncodeToString(itemsData)
	//log.Print("String:", myString)
	resp.Insert(itemsData, index)
	index += len(itemsData)

	if sale != nil {
		resp.Insert([]byte{0x02}, index) // sale indicator
		index++

		resp.Insert([]byte{byte(len(sale.Name))}, index) // sale name length
		index++

		resp.Insert([]byte(sale.Name), index) // sale name
		index += len(sale.Name)

		resp.Insert([]byte{0x00}, index)
		index++
	}
	resp.SetLength(int16(binary.Size(resp) - 6))
	dataItems, _ := c.ShowItems()
	resp.Concat(dataItems) // FIX => workaround for weapon slot

	if c.GuildID > 0 {
		guild, err := FindGuildByID(c.GuildID)
		if err == nil && guild != nil {
			resp.Concat(guild.GetInfo())
		}
	}

	STYLE_MENU := utils.Packet{0xaa, 0x55, 0x0d, 0x00, 0x01, 0xb5, 0x0a, 0x00, 0x00, 0x55, 0xaa}
	styleresp := STYLE_MENU
	styleresp[8] = byte(0x02)
	index = 9
	headitem, ok := GetItemInfo(c.HeadStyle)
	if !ok || headitem == nil {
		c.HeadStyle = 0
	}
	faceitem, ok := GetItemInfo(c.FaceStyle)
	if !ok || faceitem == nil {
		c.FaceStyle = 0
	}

	styleresp.Insert(utils.IntToBytes(uint64(c.HeadStyle), 4, true), index)
	index += 4
	styleresp.Insert(utils.IntToBytes(uint64(c.FaceStyle), 4, true), index)
	index += 4
	resp.Concat(styleresp)

	c.DeleteAura()

	return resp, nil
}

func (c *Character) ShowItems() ([]byte, error) {

	if c == nil {
		return nil, nil
	}

	slots := c.GetAppearingItemSlots()
	inventory, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}

	helm := inventory[slots[0]]
	mask := inventory[slots[1]]
	armor := inventory[slots[2]]
	weapon1 := inventory[slots[3]]
	weapon2 := inventory[slots[4]]
	boots := inventory[slots[9]]
	pet := inventory[slots[10]].Pet

	count := byte(4)
	if weapon1.ItemID > 0 {
		count++
	}
	if weapon2.ItemID > 0 {
		count++
	}
	if pet != nil && pet.IsOnline {
		count++
	}
	weapon1ID := int64(0)
	if weapon1.Appearance != 0 {
		weapon1ID = weapon1.Appearance
	}
	weapon2ID := int64(0)
	if weapon2.Appearance != 0 {
		weapon2ID = weapon2.Appearance
	}
	helmID := int64(0)
	if slots[0] == 0 && helm.Appearance != 0 {
		helmID = helm.Appearance
	}
	if helmID == 0 {
		helmID = int64(c.HeadStyle)
	}
	maskID := int64(0)
	if slots[1] == 1 && mask.Appearance != 0 {
		maskID = mask.Appearance
	}
	if maskID == 0 {
		maskID = int64(c.FaceStyle)
	}
	armorID := int64(0)
	if slots[2] == 2 && armor.Appearance != 0 {
		armorID = armor.Appearance
	}
	bootsID := int64(0)
	if slots[9] == 9 && boots.Appearance != 0 {
		bootsID = boots.Appearance
	}

	resp := SHOW_ITEMS
	resp.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), 8) // character pseudo id
	resp[10] = byte(c.WeaponSlot)                                 // character weapon slot
	resp[11] = count

	index := 12
	resp.Insert(utils.IntToBytes(uint64(helm.ItemID), 4, true), index) // helm id
	index += 4

	resp.Insert(utils.IntToBytes(uint64(slots[0]), 2, true), index) // helm slot
	resp.Insert([]byte{0xA2}, index+2)
	index += 3

	resp.Insert(utils.IntToBytes(uint64(helm.Plus), 1, true), index) // helm plus
	resp.Insert(utils.IntToBytes(uint64(helmID), 4, true), index+1)  // Kinézet
	index += 5

	resp.Insert(utils.IntToBytes(uint64(mask.ItemID), 4, true), index) // mask id
	index += 4

	resp.Insert(utils.IntToBytes(uint64(slots[1]), 2, true), index) // mask slot
	resp.Insert([]byte{0xA2}, index+2)
	index += 3

	resp.Insert(utils.IntToBytes(uint64(mask.Plus), 1, true), index) // mask plus
	resp.Insert(utils.IntToBytes(uint64(maskID), 4, true), index+1)  // Kinézet
	index += 5

	resp.Insert(utils.IntToBytes(uint64(armor.ItemID), 4, true), index) // armor id
	index += 4

	resp.Insert(utils.IntToBytes(uint64(slots[2]), 2, true), index) // armor slot
	resp.Insert([]byte{0xA2}, index+2)
	index += 3

	resp.Insert(utils.IntToBytes(uint64(armor.Plus), 1, true), index) // armor plus
	resp.Insert(utils.IntToBytes(uint64(armorID), 4, true), index+1)  // Kinézet
	index += 5

	if weapon1.ItemID > 0 {
		resp.Insert(utils.IntToBytes(uint64(weapon1.ItemID), 4, true), index) // weapon1 id
		index += 4

		resp.Insert([]byte{0x03, 0x00}, index) // weapon1 slot
		resp.Insert([]byte{0xA2}, index+2)
		index += 3

		resp.Insert(utils.IntToBytes(uint64(weapon1.Plus), 1, true), index) // weapon1 plus
		resp.Insert(utils.IntToBytes(uint64(weapon1ID), 4, true), index+1)  // Kinézet
		index += 5
	}

	if weapon2.ItemID > 0 {
		resp.Insert(utils.IntToBytes(uint64(weapon2.ItemID), 4, true), index) // weapon2 id
		index += 4

		resp.Insert([]byte{0x04, 0x00}, index) // weapon2 slot
		resp.Insert([]byte{0xA2}, index+2)
		index += 3

		resp.Insert(utils.IntToBytes(uint64(weapon2.Plus), 1, true), index) // weapon2 plus
		resp.Insert(utils.IntToBytes(uint64(weapon2ID), 4, true), index+1)  // Kinézet
		index += 5
	}

	resp.Insert(utils.IntToBytes(uint64(boots.ItemID), 4, true), index) // boots id
	index += 4

	resp.Insert(utils.IntToBytes(uint64(slots[9]), 2, true), index) // boots slot
	resp.Insert([]byte{0xA2}, index+2)
	index += 3

	resp.Insert(utils.IntToBytes(uint64(boots.Plus), 1, true), index) // boots plus
	resp.Insert(utils.IntToBytes(uint64(bootsID), 4, true), index+1)  // Kinézet
	index += 5

	if pet != nil && pet.IsOnline {
		resp.Insert(utils.IntToBytes(uint64(inventory[10].ItemID), 4, true), index) // pet id
		index += 4

		resp.Insert(utils.IntToBytes(uint64(slots[10]), 2, true), index) // pet slot
		index += 2
		if pet != nil {
			resp.Insert(utils.IntToBytes(uint64(pet.Level), 1, true), index) // pet plus ?
			index++
		} else {
			resp.Insert(utils.IntToBytes(uint64(0), 1, true), index) // pet plus ?
			index++
		}
		resp.Insert([]byte{0x05, 0x00, 0x00, 0x00}, index)
		index += 4
	}

	resp.SetLength(int16(binary.Size(resp) - 6))
	return resp, nil
}

func FindOnlineCharacterByUserID(userID string) (*Character, error) {

	var id int
	query := `select id from hops.characters where user_id = $1 and is_online = true`

	if err := pgsql_DbMap.SelectOne(&id, query, userID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindOnlineCharacterByUserID: %s", err.Error())
	}

	return FindCharacterByID(id)
}
func FindCharactersInMap(mapid int16) map[int]*Character {

	characterMutex.RLock()
	allChars := funk.Values(characters).([]*Character)
	characterMutex.RUnlock()

	allChars = funk.Filter(allChars, func(c *Character) bool {
		if c.Socket == nil {
			return false
		}

		return c.Map == mapid && c.IsOnline
	}).([]*Character)

	candidates := make(map[int]*Character)
	for _, c := range allChars {
		candidates[c.ID] = c
	}

	return candidates
}

func FindCharactersInServer(server int) (map[int]*Character, error) {

	characterMutex.RLock()
	allChars := funk.Values(characters).([]*Character)
	characterMutex.RUnlock()

	allChars = funk.Filter(allChars, func(c *Character) bool {
		if c.Socket == nil {
			return false
		}
		user := c.Socket.User
		if user == nil {
			return false
		}

		return user.ConnectedServer == server && c.IsOnline
	}).([]*Character)

	candidates := make(map[int]*Character)
	for _, c := range allChars {
		candidates[c.ID] = c
	}

	return candidates, nil
}

func FindOnlineCharacters() (map[int]*Character, error) {

	characters := make(map[int]*Character)
	users := AllUsers()
	users = funk.Filter(users, func(u *User) bool {
		return u.ConnectedIP != "" && u.ConnectedServer > 0
	}).([]*User)

	for _, u := range users {
		c, _ := FindOnlineCharacterByUserID(u.ID)
		if c == nil {
			continue
		}

		characters[c.ID] = c
	}

	return characters, nil
}

func (c *Character) FindItemInInventory(callback func(*InventorySlot) bool, itemIDs ...int64) (int16, *InventorySlot, error) {

	slots, err := c.InventorySlots()
	if err != nil {
		return -1, nil, err
	}

	for index, slot := range slots {
		if ok, _ := utils.Contains(itemIDs, slot.ItemID); ok {
			if index >= 0x43 && index <= 0x132 {
				continue
			}

			if callback == nil || callback(slot) {
				return int16(index), slot, nil
			}
		}
	}

	return -1, nil, nil
}

func (c *Character) FindItemInInventoryByType(callback func(*InventorySlot) bool, Type int) (int16, *InventorySlot, error) {

	slots, err := c.InventorySlots()
	if err != nil {
		return -1, nil, err
	}

	for index, slot := range slots {
		iteminfo, ok := GetItemInfo(slot.ItemID)
		if !ok || iteminfo == nil {
			continue
		}
		if iteminfo.GetType() == Type {
			if index >= 0x43 && index <= 0x132 {
				continue
			}

			if callback == nil || callback(slot) {
				return int16(index), slot, nil
			}
		}
	}

	return -1, nil, nil
}
func (c *Character) FindItemAlreadyInUse(callback func(*InventorySlot) bool, itemIDs ...int64) (int16, *InventorySlot, error) {

	slots, err := c.InventorySlots()
	if err != nil {
		return -1, nil, err
	}

	for index, slot := range slots {
		if ok, _ := utils.Contains(itemIDs, slot.ItemID); ok {
			if index >= 0x43 && index <= 0x132 {
				continue
			}

			if callback == nil || callback(slot) {
				return int16(index), slot, nil
			}
		}
	}

	return -1, nil, nil
}
func (c *Character) FindStockableItemInInventory(callback func(*InventorySlot) bool, item *InventorySlot) (int16, *InventorySlot, error) {

	slots, err := c.InventorySlots()
	if err != nil {
		return -1, nil, err
	}

	for index, slot := range slots {
		if slot.ItemID == item.ItemID {
			if item.Plus != slot.Plus {
				continue
			}
			if index >= 0x43 && index <= 0x132 || index >= 402 {
				continue
			}
			if (callback == nil || callback(slot)) && slot.Quantity < 10000 {

				return int16(index), slot, nil
			}
		}
	}

	return -1, nil, nil
}

func (c *Character) DecrementItem(slotID int16, amount uint) *utils.Packet {

	slots, err := c.InventorySlots()
	if err != nil {
		return nil
	}

	slot := slots[slotID]
	if slot == nil || slot.ItemID == 0 || slot.Quantity < amount {
		return nil
	}

	slot.Quantity -= amount

	info, ok := GetItemInfo(slot.ItemID)
	if !ok {
		return nil
	}
	resp := utils.Packet{}

	if info.TimerType == 3 {
		resp = GREEN_ITEM_COUNT
		resp.Insert(utils.IntToBytes(uint64(slotID), 2, true), 8)         // slot id
		resp.Insert(utils.IntToBytes(uint64(slot.Quantity), 4, true), 10) // item quantity
	} else {
		resp = ITEM_COUNT
		resp.Insert(utils.IntToBytes(uint64(slot.ItemID), 4, true), 8)    // item id
		resp.Insert(utils.IntToBytes(uint64(slotID), 2, true), 12)        // slot id
		resp.Insert(utils.IntToBytes(uint64(slot.Quantity), 2, true), 14) // item quantity
	}

	if slot.Quantity == 0 {
		err = slot.Delete()
		if err != nil {
			log.Print(err)
			return nil
		}
		*slot = *NewSlot()
	} else {
		err = slot.Update()
		if err != nil {
			log.Print(err)
			return nil
		}
	}

	return &resp
}

func (c *Character) FindFreeSlot() (int16, error) {

	slotID := 11
	slots, err := c.InventorySlots()
	if err != nil {
		return -1, err
	}

	for ; slotID <= 66; slotID++ {
		slot := slots[slotID]
		if slot.ItemID == 0 {
			return int16(slotID), nil
		}
	}

	if c.DoesInventoryExpanded() {
		slotID = 341
		for ; slotID <= 396; slotID++ {
			slot := slots[slotID]
			if slot.ItemID == 0 {
				return int16(slotID), nil
			}
		}
	}

	return -1, nil
}

func (c *Character) FindFreeSlots(count int) ([]int16, error) {

	var slotIDs []int16
	slots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}

	for slotID := int16(11); slotID <= 66; slotID++ {
		slot := slots[slotID]
		if slot.ItemID == 0 {
			slotIDs = append(slotIDs, slotID)
		}
		if len(slotIDs) == count {
			return slotIDs, nil
		}
	}

	if c.DoesInventoryExpanded() {
		for slotID := int16(341); slotID <= 396; slotID++ {
			slot := slots[slotID]
			if slot.ItemID == 0 {
				slotIDs = append(slotIDs, slotID)
			}
			if len(slotIDs) == count {
				return slotIDs, nil
			}
		}
	}

	return nil, fmt.Errorf("not enough inventory space")
}

func (c *Character) DoesInventoryExpanded() bool {
	buffs, err := FindBuffsByCharacterID(c.ID)
	if err != nil || len(buffs) == 0 {
		return false
	}

	buffs = funk.Filter(buffs, func(b *Buff) bool {
		return b.BagExpansion
	}).([]*Buff)

	return len(buffs) > 0
}
func (c *Character) FindExtentionBagBuff() []*Buff {
	buffs, err := FindBuffsByCharacterID(c.ID)
	if err != nil || len(buffs) == 0 {
		return nil
	}

	buffs = funk.Filter(buffs, func(b *Buff) bool {
		return b.BagExpansion
	}).([]*Buff)

	return buffs
}

func (c *Character) AddItem(itemToAdd *InventorySlot, slotID int16, lootingDrop bool) (*utils.Packet, int16, error) {

	var (
		item *InventorySlot
	)

	if itemToAdd == nil {
		return nil, -1, nil
	}

	itemToAdd.CharacterID = null.IntFrom(int64(c.ID))
	itemToAdd.UserID = null.StringFrom(c.UserID)

	iteminfo, ok := GetItemInfo(itemToAdd.ItemID)
	if !ok || iteminfo == nil {
		log.Printf("Additem GetItemInfo error %d", itemToAdd.ItemID)
		return nil, -1, nil
	}
	stackable := iteminfo.Stackable

	slots, err := c.InventorySlots()
	if err != nil {
		return nil, -1, err
	}

	stacking := false
	resp := utils.Packet{}
	if slotID == -1 {
		if stackable { // stackable item
			slotID, item, err = c.FindStockableItemInInventory(nil, itemToAdd)
			if err != nil {
				return nil, -1, err
			} else if slotID == -1 { // no same item found => find free slot
				slotID, err = c.FindFreeSlot()
				if err != nil {
					return nil, -1, err
				} else if slotID == -1 { // no free slot

					return nil, -1, nil
				}
				stacking = false
			} else if item.ItemID != itemToAdd.ItemID { // slot is not available

				return nil, -1, nil

				/*} else if item.Quantity >= 10000 { // max items stockable 999
				slotID, item, err = c.FindStockableItemInInventory(nil, itemToAdd)
				if err != nil {
					return nil, -1, nil
				}
				if slotID == -1 {
					slotID, err = c.FindFreeSlot()
					if err != nil {
						return nil, -1, err
					} else if slotID == -1 {
						return nil, -1, nil
					}
				} else {
					itemToAdd.Quantity += item.Quantity
					stacking = true
				}*/

			} else if item != nil { // can be stacked
				itemToAdd.Quantity += item.Quantity
				stacking = true
			}
		} else { // not stackable item => find free slot
			slotID, err = c.FindFreeSlot()
			if err != nil {
				return nil, -1, err
			} else if slotID == -1 {
				return nil, -1, nil
			}
		}
	}

	itemToAdd.SlotID = slotID
	slot := slots[slotID]
	id := slot.ID
	*slot = *itemToAdd
	slot.ID = id

	if !stacking && !stackable {

		if lootingDrop {
			iteminfo, ok := GetItemInfo(itemToAdd.ItemID)
			if !ok || iteminfo == nil {
				log.Printf("Additem GetItemInfo error %d", itemToAdd.ItemID)
				return nil, -1, nil
			}
			r := ITEM_LOOTED
			r.Insert(utils.IntToBytes(uint64(itemToAdd.ItemID), 4, true), 9) // item id
			r[14] = 0xA1
			if itemToAdd.Plus > 0 || itemToAdd.SocketCount > 0 {
				r[14] = 0xA2
			}

			r.Insert(utils.IntToBytes(uint64(slot.Quantity), 2, true), 15) // item count
			r.Insert(utils.IntToBytes(uint64(slotID), 2, true), 17)        // slot id
			r.Insert(itemToAdd.GetUpgrades(), 19)                          // item upgrades
			resp.Concat(r)
		} else {
			iteminfo, ok := GetItemInfo(itemToAdd.ItemID)
			if !ok || iteminfo == nil {
				log.Printf("Additem GetItemInfo error %d", itemToAdd.ItemID)
				return nil, -1, nil
			}
			resp = ITEM_ADDED
			resp.Insert(utils.IntToBytes(uint64(itemToAdd.ItemID), 4, true), 6) // item id
			resp.Insert(utils.IntToBytes(uint64(slot.Quantity), 2, true), 12)   // item quantity
			resp.Insert(utils.IntToBytes(uint64(slotID), 2, true), 14)          // slot id
			resp.Insert(utils.IntToBytes(uint64(0), 8, true), 20)               // ures
			//info := Items[itemToAdd.ItemID]
			//if info.GetType() == PET_TYPE && slot.Pet.Name != "" {
			//resp.Overwrite([]byte(slot.Pet.Name), 32)
			//}

			resp.Overwrite(utils.IntToBytes(uint64(itemToAdd.ItemType), 1, true), 41) // ures
			resp.Overwrite(utils.IntToBytes(uint64(itemToAdd.JudgementStat), 4, true), 42)
			if slot.Appearance != 0 {
				resp.Overwrite(utils.IntToBytes(uint64(slot.Appearance), 4, true), 46) // KINÉZET id 16 volt
			}

			resp.Concat(messaging.InfoMessage(fmt.Sprintf("You acquired %s.", iteminfo.Name)))
		}
		resp.Concat(c.GetGold())
	} else {

		if lootingDrop {
			iteminfo, ok := GetItemInfo(itemToAdd.ItemID)
			if !ok || iteminfo == nil {
				log.Printf("Additem GetItemInfo error %d", itemToAdd.ItemID)
				return nil, -1, nil
			}
			r := ITEM_LOOTED
			r.Insert(utils.IntToBytes(uint64(itemToAdd.ItemID), 4, true), 9) // item id
			r[14] = 0xA1
			r.Insert(utils.IntToBytes(uint64(slot.Quantity), 2, true), 15) // item count
			r.Insert(utils.IntToBytes(uint64(slotID), 2, true), 17)        // slot id
			r.Insert(itemToAdd.GetUpgrades(), 19)                          // item upgrades
			resp.Concat(r)
		} else if stacking {
			iteminfo, ok := GetItemInfo(itemToAdd.ItemID)
			if !ok || iteminfo == nil {
				log.Printf("Additem GetItemInfo error %d", itemToAdd.ItemID)
				return nil, -1, nil
			}
			resp = ITEM_COUNT
			resp.Insert(utils.IntToBytes(uint64(itemToAdd.ItemID), 4, true), 8)    // item id
			resp.Insert(utils.IntToBytes(uint64(slotID), 2, true), 12)             // slot id
			resp.Insert(utils.IntToBytes(uint64(itemToAdd.Quantity), 2, true), 14) // item quantity
			resp.Concat(messaging.InfoMessage(fmt.Sprintf("You acquired %s.", iteminfo.Name)))
		} else if !stacking {
			iteminfo, ok := GetItemInfo(itemToAdd.ItemID)
			if !ok || iteminfo == nil {
				log.Printf("Additem GetItemInfo error %d", itemToAdd.ItemID)
				return nil, -1, nil
			}
			slot := slots[slotID]
			slot.ItemID = itemToAdd.ItemID
			slot.Quantity = itemToAdd.Quantity
			slot.Plus = itemToAdd.Plus
			slot.UpgradeArr = itemToAdd.UpgradeArr

			resp = ITEM_ADDED
			resp.Insert(utils.IntToBytes(uint64(itemToAdd.ItemID), 4, true), 6)    // item id
			resp.Insert(utils.IntToBytes(uint64(itemToAdd.Quantity), 2, true), 12) // item quantity
			resp.Insert(utils.IntToBytes(uint64(slotID), 2, true), 14)             // slot id
			resp.Insert(utils.IntToBytes(uint64(0), 8, true), 20)                  // gold
			info, _ := GetItemInfo(itemToAdd.ItemID)
			if info.GetType() == PET_TYPE && slot.Pet.Name != "" {
				resp.Overwrite([]byte(slot.Pet.Name), 32)
			}
			resp.Overwrite(utils.IntToBytes(uint64(itemToAdd.ItemType), 1, true), 41) // ures
			resp.Overwrite(utils.IntToBytes(uint64(itemToAdd.JudgementStat), 4, true), 42)
			if slot.Appearance != 0 {
				resp.Overwrite(utils.IntToBytes(uint64(slot.Appearance), 4, true), 46) // KINÉZET id 16 volt
			}
			resp.Concat(messaging.InfoMessage(fmt.Sprintf("You acquired %s.", iteminfo.Name)))
		}
		resp.Concat(c.GetGold())
	}

	if slot.ID > 0 {
		err = slot.Update()
	} else {
		err = slot.Insert()
	}

	if err != nil {
		*slot = *NewSlot()
		resp = utils.Packet{}
		resp.Concat(slot.GetData(slotID, c.ID))
		return &resp, -1, nil
	}

	InventoryItems.Add(slot.ID, slot)
	resp.Concat(slot.GetData(slotID, c.ID))
	return &resp, slotID, nil
}

func (c *Character) ShowItemsByCharacter() ([]byte, error) {

	if c == nil {
		return nil, nil
	}

	slots := c.GetAppearingItemSlots()
	inventory, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}

	helm := inventory[slots[0]]
	mask := inventory[slots[1]]
	armor := inventory[slots[2]]
	weapon1 := inventory[slots[3]]
	weapon2 := inventory[slots[4]]
	boots := inventory[slots[9]]
	pet := inventory[slots[10]].Pet

	count := byte(4)
	if weapon1.ItemID > 0 {
		count++
	}
	if weapon2.ItemID > 0 {
		count++
	}
	if pet != nil && pet.IsOnline {
		count++
	}
	weapon1ID := int64(0)
	if weapon1.Appearance != 0 {
		weapon1ID = weapon1.Appearance
	}
	weapon2ID := int64(0)
	if weapon2.Appearance != 0 {
		weapon2ID = weapon2.Appearance
	}
	helmID := int64(0)
	if slots[0] == 0 && helm.Appearance != 0 {
		helmID = helm.Appearance
	}
	if helmID == 0 {
		helmID = int64(c.HeadStyle)
	}
	maskID := int64(0)
	if slots[1] == 1 && mask.Appearance != 0 {
		maskID = mask.Appearance
	}
	if maskID == 0 {
		maskID = int64(c.FaceStyle)
	}
	armorID := int64(0)
	if slots[2] == 2 && armor.Appearance != 0 {
		armorID = armor.Appearance
	}
	bootsID := int64(0)
	if slots[9] == 9 && boots.Appearance != 0 {
		bootsID = boots.Appearance
	}
	resp := utils.Packet{}
	index := 0
	resp.Insert(utils.IntToBytes(uint64(count), 1, true), index) // count
	index++
	if helm.ItemID > 0 {
		resp.Insert(utils.IntToBytes(uint64(helm.ItemID), 4, true), index) // helm id
		index += 4

		resp.Insert(utils.IntToBytes(uint64(slots[0]), 2, true), index) // helm slot
		resp.Insert([]byte{0xA4}, index+2)
		index += 3

		resp.Insert(utils.IntToBytes(uint64(helm.Plus), 1, true), index) // helm plus
		resp.Insert(utils.IntToBytes(uint64(helmID), 4, true), index+1)  // Kinézet
		index += 5
	}
	if mask.ItemID > 0 {
		resp.Insert(utils.IntToBytes(uint64(mask.ItemID), 4, true), index) // mask id
		index += 4

		resp.Insert(utils.IntToBytes(uint64(slots[1]), 2, true), index) // mask slot
		resp.Insert([]byte{0xA4}, index+2)
		index += 3

		resp.Insert(utils.IntToBytes(uint64(mask.Plus), 1, true), index) // mask plus
		resp.Insert(utils.IntToBytes(uint64(maskID), 4, true), index+1)  // Kinézet
		index += 5
	}
	if armor.ItemID > 0 {
		resp.Insert(utils.IntToBytes(uint64(armor.ItemID), 4, true), index) // armor id
		index += 4

		resp.Insert(utils.IntToBytes(uint64(slots[2]), 2, true), index) // armor slot
		resp.Insert([]byte{0xA4}, index+2)
		index += 3

		resp.Insert(utils.IntToBytes(uint64(armor.Plus), 1, true), index) // armor plus
		resp.Insert(utils.IntToBytes(uint64(armorID), 4, true), index+1)  // Kinézet
		index += 5
	}
	if weapon1.ItemID > 0 {
		resp.Insert(utils.IntToBytes(uint64(weapon1.ItemID), 4, true), index) // weapon1 id
		index += 4

		resp.Insert([]byte{0x03, 0x00}, index) // weapon1 slot
		resp.Insert([]byte{0xA4}, index+2)
		index += 3

		resp.Insert(utils.IntToBytes(uint64(weapon1.Plus), 1, true), index) // weapon1 plus
		resp.Insert(utils.IntToBytes(uint64(weapon1ID), 4, true), index+1)  // Kinézet
		index += 5
	}
	if weapon2.ItemID > 0 {
		resp.Insert(utils.IntToBytes(uint64(weapon2.ItemID), 4, true), index) // weapon2 id
		index += 4

		resp.Insert([]byte{0x04, 0x00}, index) // weapon2 slot
		resp.Insert([]byte{0xA4}, index+2)
		index += 3

		resp.Insert(utils.IntToBytes(uint64(weapon2.Plus), 1, true), index) // weapon2 plus
		resp.Insert(utils.IntToBytes(uint64(weapon2ID), 4, true), index+1)  // Kinézet
		index += 5
	}
	if boots.ItemID > 0 {
		resp.Insert(utils.IntToBytes(uint64(boots.ItemID), 4, true), index) // boots id
		index += 4

		resp.Insert(utils.IntToBytes(uint64(slots[9]), 2, true), index) // boots slot
		resp.Insert([]byte{0xA4}, index+2)
		index += 3

		resp.Insert(utils.IntToBytes(uint64(boots.Plus), 1, true), index) // boots plus
		resp.Insert(utils.IntToBytes(uint64(bootsID), 4, true), index+1)  // Kinézet
		index += 5
	}
	if pet != nil {
		resp.Insert(utils.IntToBytes(uint64(inventory[10].ItemID), 4, true), index) // pet id
		index += 4

		resp.Insert(utils.IntToBytes(uint64(slots[10]), 2, true), index) // pet slot
		index += 2
		if pet != nil {
			resp.Insert(utils.IntToBytes(uint64(pet.Level), 1, true), index) // pet plus ?
			index++
		} else {
			resp.Insert(utils.IntToBytes(uint64(0), 1, true), index) // pet plus ?
			index++
		}
		resp.Insert([]byte{0x05, 0x00, 0x00, 0x00}, index)
		index += 4
	}

	//resp.SetLength(int16(binary.Size(resp) - 6))
	return resp, nil
}

func (c *Character) ReplaceItem(itemID int, where, to int16) ([]byte, error) {

	c.AntiDupeMutex.Lock()
	defer c.AntiDupeMutex.Unlock()
	sale := FindSale(c.PseudoID)

	boxbank, err := c.BoxOpenerStorage()
	if err != nil {
		return nil, err
	}

	invSlots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}

	toItem := invSlots[to]
	whereItem := invSlots[where]
	whereItemInfo, ok := GetItemInfo(whereItem.ItemID)
	if !ok {
		return nil, nil
	}

	if (toItem.ItemID >= 99059990 && toItem.ItemID <= 99059994) || (whereItem.ItemID >= 99059990 && whereItem.ItemID <= 99059994) {
		resp := ITEM_REPLACEMENT
		resp.Insert(utils.IntToBytes(uint64(itemID), 4, true), 8) // item id
		resp.Insert(utils.IntToBytes(uint64(to), 2, true), 12)    // where slot id
		resp.Insert(utils.IntToBytes(uint64(where), 2, true), 14) // to slot id
		return resp, nil
	}
	if toItem.ItemID != 0 {
		return nil, nil
	}

	if sale != nil {
		return messaging.SystemMessage(messaging.CANNOT_MOVE_ITEM_IN_TRADE), nil
	} else if c.TradeID != "" {
		return messaging.SystemMessage(messaging.CANNOT_MOVE_ITEM_IN_TRADE), nil
	}
	if where >= 67 && where <= 306 && c.TypeOfBankOpened == 2 {
		whereItem = boxbank[where]
	} else if to >= 67 && to <= 306 && c.TypeOfBankOpened == 2 {
		toItem = boxbank[to]
	}

	if whereItem.ItemID == 0 {

		return messaging.SystemMessage(4411), nil
	}

	if (where >= 0x0043 && where <= 0x132) && (to >= 0x0043 && to <= 0x132) && toItem.ItemID == 0 && whereItemInfo.Tradable != 2 { // From: Bank, To: Bank
		if c.TypeOfBankOpened == 2 {
			to = 449
			where = 450
			goto OUT
		}
		whereItem.SlotID = to
		*toItem = *whereItem
		*whereItem = *NewSlot()

	} else if (where >= 0x0043 && where <= 0x132) && (to < 0x0043 || to > 0x132) && toItem.ItemID == 0 && whereItemInfo.Tradable != 2 { // From: Bank, To: Inventory

		whereItem.SlotID = to
		whereItem.CharacterID = null.IntFrom(int64(c.ID))
		*toItem = *whereItem
		*whereItem = *NewSlot()

	} else if (to >= 0x0043 && to <= 0x132) && (where < 0x0043 || where > 0x132) && toItem.ItemID == 0 &&
		!whereItem.Activated && !whereItem.InUse && whereItemInfo.Tradable != 2 { // From: Inventory, To: Bank

		whereItem.SlotID = to
		whereItem.CharacterID = null.IntFromPtr(nil)
		*toItem = *whereItem
		*whereItem = *NewSlot()

	} else if ((to < 0x0043 || to > 0x132) && (where < 0x0043 || where > 0x132)) && toItem.ItemID == 0 { // From: Inventory, To: Inventory
		whereItem.SlotID = to
		*toItem = *whereItem
		*whereItem = *NewSlot()

	} else {
		return messaging.SystemMessage(8047), nil
	}

	toItem.Update()
	InventoryItems.Add(toItem.ID, toItem)

OUT:
	resp := ITEM_REPLACEMENT
	resp.Insert(utils.IntToBytes(uint64(itemID), 4, true), 8) // item id
	resp.Insert(utils.IntToBytes(uint64(where), 2, true), 12) // where slot id
	resp.Insert(utils.IntToBytes(uint64(to), 2, true), 14)    // to slot id

	whereAffects, toAffects := DoesSlotAffectStats(where), DoesSlotAffectStats(to)

	info, _ := GetItemInfo(int64(itemID))
	if whereAffects {
		if info != nil && info.Timer > 0 {
			toItem.InUse = false
		}
	}
	if toAffects {
		if info != nil && info.Timer > 0 {
			toItem.InUse = true
		}
	}

	if whereAffects || toAffects {
		statData, err := c.GetStats()
		if err != nil {
			return nil, err
		}

		resp.Concat(statData)

		itemsData, err := c.ShowItems()
		if err != nil {
			return nil, err
		}

		p := nats.CastPacket{CastNear: true, CharacterID: c.ID, Type: nats.SHOW_ITEMS, Data: itemsData}
		if err = p.Cast(); err != nil {
			return nil, err
		}

		resp.Concat(itemsData)
	}

	if to == 0x0A {
		petSlot := invSlots[to]
		pet := petSlot.Pet
		if pet != nil {
			pet.IsOnline = false
		}
		resp.Concat(invSlots[to].GetPetStats(c))
		resp.Concat(SHOW_PET_BUTTON)
	} else if where == 0x0A {
		resp.Concat(DISMISS_PET)
	}

	if (where >= 317 && where <= 319) || (to >= 317 && to <= 319) {
		resp.Concat(c.GetPetStats())
	}

	return resp, nil
}

func (c *Character) SwapItems(where, to int16) ([]byte, error) {

	c.AntiDupeMutex.Lock()
	defer c.AntiDupeMutex.Unlock()
	sale := FindSale(c.PseudoID)
	if sale != nil {
		return nil, fmt.Errorf("cannot swap items on sale")
	} else if c.TradeID != "" {
		text := "Name: " + c.Name + "(" + c.UserID + ") tried to swap items in trade"
		utils.NewLog("logs/cheat_alert.txt", text)
		return nil, fmt.Errorf("cannot swap item on trade")
	}

	invSlots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}
	whereItem := invSlots[where]
	toItem := invSlots[to]
	whereItemInfo, ok := GetItemInfo(whereItem.ItemID)
	if !ok {
		text := "Name: " + c.Name + "(" + c.UserID + ") Where item !ok"
		utils.NewLog("logs/cheat_alert.txt", text)
	}
	toItemInfo, ok := GetItemInfo(toItem.ItemID)
	if !ok {
		goto OUT
	}
	if (toItem.ItemID >= 99059990 && toItem.ItemID <= 99059994) || (whereItem.ItemID >= 99059990 && whereItem.ItemID <= 99059994) {
		goto OUT
	}

	if whereItem.SlotID == where && (where == 317 || where == 318 || where == 319) && (to >= 317 && to <= 319) {
		text := "Name: " + c.Name + "(" + c.UserID + ") Where item !!! 2046"
		utils.NewLog("logs/cheat_alert.txt", text)
		return nil, nil
	}

	if whereItem.ItemID == 0 || toItem.ItemID == 0 {
		return nil, nil
	}

	if (whereItem.SlotID == where && (where == 0 || where == 1 || where == 2 || where == 3 || where == 9)) && (toItemInfo.TimerType > 0 && !toItem.Activated) {
		c.Socket.Write(messaging.SystemMessage(messaging.CANNOT_USE_ITEM))
		goto OUT
	}
	if (whereItem.SlotID == where && (where == 5 || where == 6 || where == 7 || where == 8)) && (toItemInfo.TimerType > 0 && !toItem.Activated) {
		c.Socket.Write(messaging.SystemMessage(messaging.CANNOT_USE_ITEM))
		goto OUT
	}
	if whereItem.SlotID == where && (where == 10) && (toItem.SlotID != 10) {
		c.Socket.Write(messaging.SystemMessage(messaging.CANNOT_USE_ITEM))
		goto OUT
	}
	if to == 10 {
		c.Socket.Write(messaging.SystemMessage(messaging.CANNOT_USE_ITEM))
		goto OUT
	}
	if whereItem.SlotID == where && (where == 10) && (toItemInfo.MinLevel < c.Level || toItemInfo.MaxLevel > c.Level) {
		c.Socket.Write(messaging.SystemMessage(messaging.CANNOT_USE_ITEM))
		goto OUT
	}

	if where >= 67 && where <= 306 && c.TypeOfBankOpened == 2 {
		goto OUT
	}
	if to >= 67 && to <= 306 && c.TypeOfBankOpened == 2 {
		goto OUT
	}

	if (where >= 0x0043 && where <= 0x132) && (to >= 0x0043 && to <= 0x132) && whereItemInfo.Tradable != 2 { // From: Bank, To: Bank
		temp := *toItem
		*toItem = *whereItem
		*whereItem = temp
		toItem.SlotID = to
		whereItem.SlotID = where

	} else if (where >= 0x0043 && where <= 0x132) && (to < 0x0043 || to > 0x132) &&
		!toItem.Activated && !toItem.InUse && whereItemInfo.Tradable != 2 { // From: Bank, To: Inventory

		temp := *toItem
		*toItem = *whereItem
		*whereItem = temp
		toItem.SlotID = to
		whereItem.SlotID = where

	} else if (to >= 0x0043 && to <= 0x132) && (where < 0x0043 || where > 0x132) &&
		!whereItem.Activated && !whereItem.InUse && whereItemInfo.Tradable != 2 { // From: Inventory, To: Bank

		temp := *toItem
		*toItem = *whereItem
		*whereItem = temp
		toItem.SlotID = to
		whereItem.SlotID = where

	} else if (to < 0x0043 || to > 0x132) && (where < 0x0043 || where > 0x132) { // From: Inventory, To: Inventory
		temp := *toItem
		*toItem = *whereItem
		*whereItem = temp
		toItem.SlotID = to
		whereItem.SlotID = where

	} else {
		return nil, nil
	}
OUT:
	whereItem.Update()
	toItem.Update()
	InventoryItems.Add(whereItem.ID, whereItem)
	InventoryItems.Add(toItem.ID, toItem)

	resp := ITEM_SWAP
	resp.Insert(utils.IntToBytes(uint64(where), 4, true), 9)  // where slot
	resp.Insert(utils.IntToBytes(uint64(where), 2, true), 13) // where slot
	resp.Insert(utils.IntToBytes(uint64(to), 2, true), 15)    // to slot
	resp.Insert(utils.IntToBytes(uint64(to), 4, true), 17)    // to slot
	resp.Insert(utils.IntToBytes(uint64(to), 2, true), 21)    // to slot
	resp.Insert(utils.IntToBytes(uint64(where), 2, true), 23) // where slot

	whereAffects, toAffects := DoesSlotAffectStats(where), DoesSlotAffectStats(to)

	if whereAffects {
		item := whereItem // new item
		info, _ := GetItemInfo(item.ItemID)
		if info != nil && info.Timer > 0 {
			item.InUse = true
		}

		item = toItem // old item
		info, _ = GetItemInfo(item.ItemID)
		if info != nil && info.Timer > 0 {
			item.InUse = false
		}
	}

	if toAffects {
		item := whereItem // old item
		info, _ := GetItemInfo(item.ItemID)
		if info != nil && info.Timer > 0 {
			item.InUse = false
		}

		item = toItem // new item
		info, _ = GetItemInfo(item.ItemID)
		if info != nil && info.Timer > 0 {
			item.InUse = true
		}
	}

	if whereAffects || toAffects {

		statData, err := c.GetStats()
		if err != nil {
			return nil, err
		}

		resp.Concat(statData)

		itemsData, err := c.ShowItems()
		if err != nil {
			return nil, err
		}

		p := nats.CastPacket{CastNear: true, CharacterID: c.ID, Data: itemsData, Type: nats.SHOW_ITEMS}
		if err = p.Cast(); err != nil {
			return nil, err
		}

		resp.Concat(itemsData)
	}

	if to == 0x0A {
		petSlot := invSlots[to]
		pet := petSlot.Pet
		if pet != nil {
			pet.IsOnline = false
		}
		resp.Concat(invSlots[to].GetPetStats(c))
		resp.Concat(SHOW_PET_BUTTON)
	}

	if (where >= 317 && where <= 319) || (to >= 317 && to <= 319) {
		resp.Concat(c.GetPetStats())
	}
	data := whereItem.GetData(where, c.ID)
	data2 := toItem.GetData(to, c.ID)

	resp.Concat(data)
	resp.Concat(data2)

	return resp, nil
}

func (c *Character) SplitItem(where, to, quantity uint16) ([]byte, error) {

	c.AntiDupeMutex.Lock()
	defer c.AntiDupeMutex.Unlock()
	sale := FindSale(c.PseudoID)
	if sale != nil {
		return nil, fmt.Errorf("cannot split item on sale")
	} else if c.TradeID != "" {
		text := "Name: " + c.Name + "(" + c.UserID + ") tried to split items in trade"
		utils.NewLog("logs/cheat_alert.txt", text)
		return nil, fmt.Errorf("cannot split item on trade")
	}

	slots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}
	whereItem := slots[where]
	toItem := slots[to]
	iteminfo, ok := GetItemInfo(whereItem.ItemID)
	if !ok {
		return nil, fmt.Errorf("SplitItem:item not found")
	}
	if quantity > 0 && iteminfo.TimerType == 0 {

		if whereItem.Quantity >= uint(quantity) {
			*toItem = *whereItem
			toItem.SlotID = int16(to)
			toItem.Quantity = uint(quantity)
			c.DecrementItem(int16(where), uint(quantity))

		} else {
			return nil, nil
		}

		toItem.Insert()
		InventoryItems.Add(toItem.ID, toItem)

		resp := SPLIT_ITEM
		resp.Insert(utils.IntToBytes(uint64(toItem.ItemID), 4, true), 8)       // item id
		resp.Insert(utils.IntToBytes(uint64(whereItem.Quantity), 2, true), 14) // remaining quantity
		resp.Insert(utils.IntToBytes(uint64(where), 2, true), 16)              // where slot id

		resp.Insert(utils.IntToBytes(uint64(toItem.ItemID), 4, true), 52) // item id
		resp.Insert(utils.IntToBytes(uint64(quantity), 2, true), 58)      // new quantity
		resp.Insert(utils.IntToBytes(uint64(to), 2, true), 60)            // to slot id
		resp.Concat(toItem.GetData(int16(to)))

		return resp, nil
	}

	return nil, nil
}

func (c *Character) GetHPandChi() []byte {
	hpChi := HP_CHI
	if c.Socket == nil {
		return nil
	}
	stat := c.Socket.Stats

	hpChi.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), 5)
	hpChi.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), 7)
	hpChi.Insert(utils.IntToBytes(uint64(stat.HP), 4, true), 9)
	hpChi.Insert(utils.IntToBytes(uint64(stat.CHI), 4, true), 13)

	count := 0
	buffs, _ := FindBuffsByCharacterID(c.ID)
	for _, buff := range buffs {

		_, ok := BuffInfections[buff.ID]
		if !ok {
			continue
		}
		if buff.ID == 10100 || buff.ID == 90098 {

			continue
		}

		hpChi.Insert(utils.IntToBytes(uint64(buff.ID), 4, true), 22)
		hpChi.Insert(utils.IntToBytes(uint64(buff.SkillPlus), 1, false), 26)
		hpChi.Insert([]byte{0x01}, 27)
		count++
	}

	if c.AidMode {
		hpChi.Insert(utils.IntToBytes(11121, 4, true), 22)
		hpChi.Insert([]byte{0x00, 0x00}, 26)
		count++
	}

	hpChi[21] = byte(count) // buff count

	injuryNumbers := c.CalculateInjury()
	injury1 := fmt.Sprintf("%x", injuryNumbers[1]) //0.7
	injury0 := fmt.Sprintf("%x", injuryNumbers[0]) //0.1
	injury3 := fmt.Sprintf("%x", injuryNumbers[3]) //17.48
	injury2 := fmt.Sprintf("%x", injuryNumbers[2]) //1.09
	injuryByte1 := string(injury0 + injury1)
	data, err := hex.DecodeString(injuryByte1)
	if err != nil {
		panic(err)
	}
	injuryByte2 := string(injury3 + injury2)
	data2, err := hex.DecodeString(injuryByte2)
	if err != nil {
		panic(err)
	}

	hpChi.Overwrite(data, len(hpChi)-18)
	hpChi.Overwrite(data2, len(hpChi)-16)

	hpChi.SetLength(int16(0x28 + count*6))

	return hpChi
}

func (c *Character) Handler() {

	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
			log.Printf("handler error: %+v", string(dbg.Stack()))
			c.HandlerCB = nil
			c.Update()
			c.Socket.Conn.Close()
		}
	}()
	st := c.Socket.Stats
	c.Epoch++
	if c.ArrangeCooldown > 0 {
		c.ArrangeCooldown--
	}
	//ip := c.Socket.Conn.RemoteAddr().String()
	//ip = strings.Split(ip, ":")[0]
	// if ip != "127.0.0.1" {
	// 	heartbeat := GetHeartBeatsByIp(ip)
	// 	if heartbeat == nil {
	// 		text := fmt.Sprintf("Heartbeat cheat detected: %s", ip)
	// 		utils.NewLog("logs/heartbeat_alert.txt", text)
	// 		c.Socket.Conn.Close()
	// 	} else {
	// 		if heartbeat.Last.Add(time.Second * 100).Before(time.Now()) {
	// 			text := fmt.Sprintf("Heartbeat cheat detected: %s", ip)
	// 			utils.NewLog("logs/heartbeat_alert.txt", text)
	// 			c.Socket.Conn.Close()
	// 		}
	// 	}
	// }
	if st.HP > 0 && c.Epoch%2 == 0 {

		hp, chi, injury := st.HP, st.CHI, c.Injury
		// if st.HP += st.HPRecoveryRate; st.HP > st.MaxHP {
		// 	st.HP = st.MaxHP
		// }

		// if st.CHI += st.CHIRecoveryRate; st.CHI > st.MaxCHI {
		// 	st.CHI = st.MaxCHI
		// }

		if c.Meditating {
			if c.Injury > 0 {
				c.Injury--
				if c.Injury < 0 {
					c.Injury = 0
				} else if c.Injury <= 70 {
					statData, err := c.GetStats()
					if err == nil {
						c.Socket.Write(statData)
					}
				}
			}
			if st.HP += st.HPRecoveryRate; st.HP > st.MaxHP {
				st.HP = st.MaxHP
			}

			if st.CHI += st.CHIRecoveryRate; st.CHI > st.MaxCHI {
				st.CHI = st.MaxCHI
			}
		}

		if hp != st.HP || chi != st.CHI || injury != c.Injury {
			c.Socket.Write(c.GetHPandChi()) // hp-chi packet
		}

	} else if st.HP > 0 && c.Epoch%5 == 0 {

		hp, chi, injury := st.HP, st.CHI, c.Injury
		if st.HP += st.HPRecoveryRate; st.HP > st.MaxHP {
			st.HP = st.MaxHP
		}

		if st.CHI += st.CHIRecoveryRate; st.CHI > st.MaxCHI {
			st.CHI = st.MaxCHI
		}
		if hp != st.HP || chi != st.CHI || injury != c.Injury {
			c.Socket.Write(c.GetHPandChi()) // hp-chi packet
		}

	} else if st.HP <= 0 && !c.Respawning { // dead
		if c.Injury < MAX_INJURY {
			c.Injury += 20
			if c.Injury >= MAX_INJURY {
				c.Injury = MAX_INJURY
			}
			if c.Injury >= 70 {
				statData, err := c.GetStats()
				if err == nil {
					c.Socket.Write(statData)
				}
			}
		}
		_, item, err := c.FindItemInInventory(nil, 99059990, 99059991, 99059992)
		if item != nil && err == nil {
			DropFlag(c, item.ItemID)
			rr, err := c.RemoveItem(item.SlotID)
			if err == nil {
				c.Socket.Write(rr)
			}
		}

		c.Respawning = true
		st.HP = 0
		c.Socket.Write(c.GetHPandChi())
		c.Socket.Write(CHARACTER_DIED)
		go c.RespawnCounter(10)

		if c.DuelID > 0 { // lost pvp
			opponent, _ := FindCharacterByID(c.DuelID)

			c.DuelID = 0
			c.DuelStarted = false
			c.Socket.Write(PVP_FINISHED)

			opponent.DuelID = 0
			opponent.DuelStarted = false
			opponent.Socket.Write(PVP_FINISHED)
		}
	}
	if c.AidTime <= 60 && c.AidMode {
		c.AidTime = 0
		c.AidMode = false
		c.Socket.Write(c.AidStatus())

		tpData, _ := c.ChangeMap(c.Map, nil)
		c.Socket.Write(tpData)
	}

	if c.AidMode {
		c.AidTime--
		if c.AidTime%60 == 0 {
			stData, _ := c.GetStats()
			c.Socket.Write(stData)
		}
	}

	if !c.AidMode && c.Epoch%2 == 0 && c.AidTime < 7200 {
		c.AidTime++
		if c.AidTime%60 == 0 {
			stData, _ := c.GetStats()
			c.Socket.Write(stData)
		}
	}

	if c.PartyID != "" {
		c.UpdatePartyStatus()
	}

	c.BuffsHandler()
	c.HandleLimitedItems()
	c.HandleItemsBuffs()

	go func() {
		if c.Epoch%10 == 0 {
			err := c.Update()
			if err != nil {
				log.Println(err)
			}
			err = st.Update()
			if err != nil {
				log.Println(err)
			}
			err = c.Socket.User.Update()
			if err != nil {
				log.Println(err)
			}
		}
	}()

	time.AfterFunc(time.Second, func() {
		if c.HandlerCB != nil {
			c.HandlerCB()
		}
	})
}

func (c *Character) RemoveNight() {
	c.DarkModeActive = false
	daylight := MAP_CHANGED
	c.Socket.Write(daylight)
}

func (c *Character) PetHandler() {

	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
			log.Printf("%+v", string(dbg.Stack()))
		}
	}()

	{
		slots, err := c.InventorySlots()
		if err != nil {
			log.Println(err)
			goto OUT
		}

		petSlot := slots[0x0A]
		pet := petSlot.Pet
		if pet == nil || petSlot.ItemID == 0 || !pet.IsOnline {
			return
		}

		petInfo, ok := Pets[petSlot.ItemID]
		if !ok {
			return
		}

		if pet.HP <= 0 {
			resp := utils.Packet{}
			resp.Concat(c.GetPetStats())
			resp.Concat(DISMISS_PET)
			c.Socket.Write(resp)
			c.IsMounting = false
			pet.IsOnline = false
			return
		}
		if c.AidMode {
			if c.PlayerAidSettings.PetFood1ItemID != 0 && pet.IsOnline {
				slotID, item, err := c.FindItemInInventory(nil, c.PlayerAidSettings.PetFood1ItemID)
				if err != nil {
					log.Print(err)
					return
				}
				if slotID == -1 || item == nil {
					return
				}
				percent := float32(c.PlayerAidSettings.PetFood1Percent) / float32(100)
				minPetHP := float32(pet.MaxHP) * percent
				if float32(pet.HP) <= minPetHP {
					petresp, err := c.UseConsumable(item, slotID)
					if err != nil {
						log.Printf("PetError: %s", err.Error())
						return
					} else {
						c.Socket.Write(petresp)
					}
				}

			}
			if c.PlayerAidSettings.PetChiPercent != 0 && pet.IsOnline {
				slotID, item, err := c.FindItemInInventory(nil, c.PlayerAidSettings.PetChiItemID)
				if err != nil {
					log.Print(err)
					return
				}
				if slotID == -1 || item == nil {
					return
				}
				percent := float32(c.PlayerAidSettings.PetChiPercent) / float32(100)
				minPetChi := float32(pet.MaxCHI) * percent
				if float32(pet.CHI) <= minPetChi {
					petresp, err := c.UseConsumable(item, slotID)
					if err != nil {
						log.Printf("PetError: %s", err.Error())
						return
					} else {
						c.Socket.Write(petresp)
					}
				}

			}
		}

		if petInfo.Combat && pet.Target == 0 && pet.Loyalty >= 10 {
			pet.Target, err = pet.FindTargetMobID(c) // 75% chance to trigger
			if err != nil {
				log.Println("AIHandler error:", err)
			}
		}

		if pet.Target > 0 {
			pet.IsMoving = false
		}

		if c.Epoch%60 == 0 {
			if pet.Fullness > 1 {
				pet.Fullness--
			}
			if pet.Fullness < 25 && pet.Loyalty > 1 {
				pet.Loyalty--
			} else if pet.Fullness >= 25 && pet.Loyalty < 100 {
				pet.Loyalty++
			}
		}
		cPetLevel := int(pet.Level)
		if c.Epoch%20 == 0 {
			if pet.HP < pet.MaxHP {
				pet.HP = int(math.Min(float64(pet.HP+cPetLevel*3), float64(pet.MaxHP)))
			}
			if pet.CHI < pet.MaxCHI {
				pet.CHI = int(math.Min(float64(pet.CHI+cPetLevel*2), float64(pet.CHI)))
			}
			pet.RefreshStats = true
		}

		if pet.RefreshStats {
			pet.RefreshStats = false
			c.Socket.Write(c.GetPetStats())
		}
		if !petInfo.Combat {
			pet.Target = 0
			goto OUT
		}
		if pet.IsMoving || pet.Casting {
			goto OUT
		}

		if pet.Loyalty < 10 {
			pet.Target = 0
		}

	BEGIN:
		ownerPos := ConvertPointToLocation(c.Coordinate)
		ownerdistance := utils.CalculateDistance(ownerPos, &pet.Coordinate)
		if pet.PetCombatMode == 2 && ownerdistance <= 10 {
			pet.Target = c.Selection
		} else if ownerdistance > 10 {
			pet.Target = 0
		}
		if pet.Target == 0 { // Idle mode

			/*ownerPos := ConvertPointToLocation(c.Coordinate)
			distance := utils.CalculateDistance(ownerPos, &pet.Coordinate)

			if distance > 10 { // Pet is so far from his owner
				pet.IsMoving = true
				targetX := utils.RandFloat(ownerPos.X-5, ownerPos.X+5)
				targetY := utils.RandFloat(ownerPos.Y-5, ownerPos.Y+5)

				target := utils.Location{X: targetX, Y: targetY}
				pet.TargetLocation = target
				speed := float64(10.0)

				token := pet.MovementToken
				for token == pet.MovementToken {
					pet.MovementToken = utils.RandInt(1, math.MaxInt64)
				}

				go pet.MovementHandler(pet.MovementToken, &pet.Coordinate, &target, speed)
			}*/

		} else { // Target mode
			target := GetFromRegister(c.Socket.User.ConnectedServer, c.Map, uint16(pet.Target))
			if _, ok := target.(*AI); ok { // attacked to ai
				mob, ok := GetFromRegister(c.Socket.User.ConnectedServer, c.Map, uint16(pet.Target)).(*AI)
				if !ok || mob == nil {
					pet.Target = 0
					goto OUT

				} else if mob.HP <= 0 {
					pet.Target = 0
					time.Sleep(time.Second)
					goto BEGIN
				}

				aiCoordinate := ConvertPointToLocation(mob.Coordinate)
				distance := utils.CalculateDistance(&pet.Coordinate, aiCoordinate)

				if distance <= 3 && pet.LastHit%2 == 0 { // attack
					seed := utils.RandInt(1, 1000)
					r := utils.Packet{}
					skillIds := []int{petInfo.Skill_1, petInfo.Skill_2, petInfo.Skill_3}
					skillsCount := len(skillIds) - 1
					randomSkill := utils.RandInt(0, int64(skillsCount))
					skillID := skillIds[randomSkill]
					skill, ok := SkillInfos[skillID]
					if seed < 500 && ok && pet.CHI >= skill.BaseChi && skillID != 0 {
						r.Concat(pet.CastSkill(c, skillID))
					} else {
						r.Concat(pet.Attack(c))
					}

					p := nats.CastPacket{CastNear: true, PetID: pet.PseudoID, Data: r, Type: nats.MOB_ATTACK}
					p.Cast()
					pet.LastHit++

				} else if distance > 3 && distance <= 50 { // chase
					pet.IsMoving = true
					target := GeneratePoint(aiCoordinate)
					pet.TargetLocation = target
					speed := float64(10.0)

					token := pet.MovementToken
					for token == pet.MovementToken {
						pet.MovementToken = utils.RandInt(1, math.MaxInt64)
					}

					go pet.MovementHandler(pet.MovementToken, &pet.Coordinate, &target, speed)
					pet.LastHit = 0

				} else {
					pet.LastHit++
				}
			} /*else { // FIX => attacked to player
				mob := FindCharacterByPseudoID(c.Socket.User.ConnectedServer, uint16(pet.Target))
				if mob.Socket.Stats.HP <= 0 || !c.CanAttack(mob) {
					pet.Target = 0
					time.Sleep(time.Second)
					goto BEGIN
				}
				aiCoordinate := ConvertPointToLocation(mob.Coordinate)
				distance := utils.CalculateDistance(&pet.Coordinate, aiCoordinate)

				if distance <= 3 && pet.LastHit%2 == 0 { // attack
					seed := utils.RandInt(1, 1000)
					r := utils.Packet{}
					skillIds := petInfo.GetSkills()
					skillsCount := len(skillIds) - 1
					randomSkill := utils.RandInt(0, int64(skillsCount))
					skillID := skillIds[randomSkill]
					skill, ok := SkillInfos[skillID]
					if seed < 500 && ok && pet.CHI >= skill.BaseChi && skillID != 0 {
						r.Concat(pet.CastSkill(c, skillID))
					} else {
						r.Concat(pet.PlayerAttack(c))
					}

					p := nats.CastPacket{CastNear: true, PetID: pet.PseudoID, Data: r, Type: nats.MOB_ATTACK}
					p.Cast()
					pet.LastHit++

				} else if distance > 3 && distance <= 50 { // chase
					pet.IsMoving = true
					target := GeneratePoint(aiCoordinate)
					pet.TargetLocation = target
					speed := float64(10.0)

					token := pet.MovementToken
					for token == pet.MovementToken {
						pet.MovementToken = utils.RandInt(1, math.MaxInt64)
					}

					go pet.MovementHandler(pet.MovementToken, &pet.Coordinate, &target, speed)
					pet.LastHit = 0

				} else {
					pet.LastHit++
				}
			}*/
			petSlot.Update()
		}
	}

OUT:
	time.AfterFunc(time.Second, func() {
		if c.PetHandlerCB != nil {
			c.PetHandlerCB()
		}
	})
}

func (c *Character) BuffsHandler() {
	buffs, err := FindBuffsByCharacterID(c.ID)
	if err != nil || len(buffs) == 0 {
		return
	}
	stat := c.Socket.Stats

	for _, buff := range buffs {

		if (!buff.IsServerEpoch && buff.StartedAt+buff.Duration <= c.Epoch || (buff.IsServerEpoch && buff.StartedAt+buff.Duration <= GetServerEpoch()) || buff.Duration == 0) && buff.CanExpire { // buff expired

			if buff.ID != 257 && buff.ID != 258 && buff.ID != 259 { //Poison

				stat.MinATK -= buff.ATK
				stat.MaxATK -= buff.ATK
				stat.ATKRate -= buff.ATKRate
				stat.Accuracy -= buff.Accuracy
				stat.MinArtsATK -= buff.ArtsATK
				stat.MaxArtsATK -= buff.ArtsATK
				stat.ArtsATKRate -= buff.ArtsATKRate
				stat.ArtsDEF -= buff.ArtsDEF
				stat.ArtsDEFRate -= buff.ArtsDEFRate
				stat.HPRecoveryRate -= buff.HPRecoveryRate
				stat.CHIRecoveryRate -= buff.CHIRecoveryRate
				stat.ConfusionDEF -= buff.ConfusionDEF
				stat.DEF -= buff.DEF
				stat.DefRate -= buff.DEFRate
				stat.DEXBuff -= buff.DEX
				stat.Dodge -= buff.Dodge
				stat.INTBuff -= buff.INT
				stat.MaxCHI -= buff.MaxCHI
				stat.MaxHP -= buff.MaxHP
				stat.ParalysisDEF -= buff.ParalysisDEF
				stat.PoisonDEF -= buff.PoisonDEF
				stat.STRBuff -= buff.STR

				stat.MinArtsATK -= buff.MinArtsAtk
				stat.MaxArtsATK -= buff.MaxArtsAtk

				stat.GoldMultiplier -= buff.GoldMultiplier / 100
				stat.Npc_gold_multiplier -= buff.Npc_gold_multiplier / 100
				stat.ExpMultiplier -= float64(buff.EXPMultiplier) / 1000
				stat.DropMultiplier -= float64(buff.DropMultiplier) / 1000

				stat.EnhancedProbabilitiesBuff -= buff.EnhancedProbabilitiesBuff
				stat.SyntheticCompositeBuff -= buff.SyntheticCompositeBuff
				stat.AdvancedCompositeBuff -= buff.AdvancedCompositeBuff
				stat.HyeolgongCost -= buff.HyeolgongCost

				stat.PetExpMultiplier -= buff.PetExpMultiplier / 1000
			}
			data, _ := c.GetStats()
			r := BUFF_EXPIRED
			r.Insert(utils.IntToBytes(uint64(buff.ID), 4, true), 6) // buff infection id
			r.Concat(data)

			if buff.ID == 258 {
				c.Confused = false
			} else if buff.ID == 259 {
				c.Paralised = false
			} else if buff.Name == "Floating" {
				c.Stunned = false
			}

			c.Socket.Write(r)
			err := buff.Delete()
			if err != nil {
				log.Print(err)
			}

			p := &nats.CastPacket{CastNear: true, CharacterID: c.ID, Data: c.GetHPandChi()}
			p.Cast()

			for _, invskillID := range InvisibilitySkillIDs {
				if buff.ID == invskillID {
					c.Invisible = false
					if c.DuelID > 0 {
						opponent, _ := FindCharacterByID(c.DuelID)
						sock := opponent.Socket
						if sock != nil {
							time.AfterFunc(time.Second*1, func() {
								sock.Write(opponent.OnDuelStarted())
							})
						}
					}
				}
			}
			if buff.ID == 242 || buff.ID == 245 || buff.ID == 59 || buff.ID == 73 { // detection arts
				c.DetectionMode = true
			}
		}

		if stat.HP <= 0 { //remove buff when dies
			bufftoremove := []int{55, 56, 66, 65, 257, 258, 259, 242, 245}
			if funk.Contains(bufftoremove, buff.ID) {
				buff.Duration = 0
				buff.Update()
			}
		}

	}
	for _, buff := range buffs {

		if buff.ID == 10100 || buff.ID == 10098 {
			continue
		}

		remainingTime := buff.StartedAt + buff.Duration - c.Epoch
		if buff.IsServerEpoch {
			remainingTime = buff.StartedAt + buff.Duration - GetServerEpoch()
		}
		if remainingTime == 0 {
			continue
		}
		data := BUFF_INFECTION
		data.Overwrite(utils.IntToBytes(uint64(buff.SkillPlus), 1, false), 6)
		data.Insert(utils.IntToBytes(uint64(buff.ID), 4, true), 6)        // infection id
		data.Insert(utils.IntToBytes(uint64(remainingTime), 4, true), 11) // buff remaining time

		if buff.ID == 257 && c.Epoch%5 == 0 {
			c.PoisonDamage(-buff.HPRecoveryRate)
		}

		c.Socket.Write(data)
	}
}
func (c *Character) Poison(npc *NPC) {
	damage := npc.PoisonATK - c.Socket.Stats.PoisonDEF

	rate := int(utils.RandInt(1, 1000))
	if damage < 0 || rate > damage {
		return
	}
	seconds := int64(npc.PoisInflictTime / 1000)

	infection := BuffInfections[257]
	infection.HPRecoveryRate = -damage
	c.AddBuff(infection, seconds)

}
func (c *Character) HandleLimitedItems() {

	invSlots, err := c.InventorySlots()
	if err != nil {
		return
	}

	slotIDs := []int16{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0133, 0x0134, 0x0135, 0x0136, 0x0137, 0x0138, 0x0139, 0x013A, 0x013B}

	for _, slotID := range slotIDs {
		slot := invSlots[slotID]
		item, ok := GetItemInfo(slot.ItemID)
		if ok && item != nil && (item.TimerType == 1 || item.TimerType == 3) { // time limited item

			if c.Epoch%60 == 0 {
				data := c.DecrementItem(slotID, 1)
				c.Socket.Write(*data)
			}
			if slot.Quantity == 1 {
				data := ITEM_EXPIRED
				data.Insert(utils.IntToBytes(uint64(item.ID), 4, true), 6)

				removeData, _ := c.RemoveItem(slotID)
				data.Concat(removeData)

				statData, _ := c.GetStats()
				data.Concat(statData)
				c.Socket.Write(data)
			}
		}
	}

	starts, ends := []int16{0x0B, 0x0155}, []int16{0x043, 0x018D}
	for j := 0; j < 2; j++ {
		start, end := starts[j], ends[j]
		for slotID := start; slotID <= end; slotID++ {
			slot := invSlots[slotID]
			item, ok := GetItemInfo(slot.ItemID)
			if ok && item != nil && slot.Activated {
				if c.Epoch%60 == 0 {
					data := c.DecrementItem(slotID, 1)
					c.Socket.Write(*data)
				}
				if slot.Quantity == 0 { // item expired
					data := ITEM_EXPIRED
					data.Insert(utils.IntToBytes(uint64(item.ID), 4, true), 6)

					c.RemoveItem(slotID)
					data.Concat(slot.GetData(slotID))

					statData, _ := c.GetStats()
					data.Concat(statData)
					c.Socket.Write(data)

					if slot.ItemID == 100080008 { // eyeball of divine
						c.DetectionMode = false
					}

					if item.GetType() == FORM_TYPE {
						c.Morphed = false
						c.MorphedNPCID = 0
						c.Socket.Write(FORM_DEACTIVATED)
						characters, err := c.GetNearbyCharacters()
						if err != nil {
							log.Println(err)
							//return
						}
						for _, chars := range characters {
							delete(chars.OnSight.Players, c.ID)
						}
						if c.DuelID > 0 {
							opponent, _ := FindCharacterByID(c.DuelID)
							spawnData, _ := c.SpawnCharacter()

							r := utils.Packet{}
							r.Concat(spawnData)
							r.Overwrite(utils.IntToBytes(500, 2, true), 13) // duel state

							sock := GetSocket(opponent.UserID)
							if sock != nil {
								sock.Write(r)
							}
						}
					}
					if (item.ID == 200000038 || item.ID == 200000039) && (c.Map == 72 || c.Map == 73 || c.Map == 74 || c.Map == 75) {
						resp, err := c.ChangeMap(1, nil)
						if err != nil {
							return
						}
						c.Socket.Write(resp)
					}

				} else { // item not expired
					if slot.ItemID == 100080008 && !c.DetectionMode { // eyeball of divine
						c.DetectionMode = true
					} else if item.GetType() == FORM_TYPE && !c.Morphed {
						c.Morphed = true
						c.MorphedNPCID = item.NPCID
						r := FORM_ACTIVATED
						r.Insert(utils.IntToBytes(uint64(item.NPCID), 4, true), 5) // form npc id
						data, err := c.GetStats()
						if err == nil {
							r.Concat(data)
						}

						c.Socket.Write(r)
						if c.DuelID > 0 {
							opponent, _ := FindCharacterByID(c.DuelID)
							spawnData, _ := c.SpawnCharacter()

							r := utils.Packet{}
							r.Concat(spawnData)
							r.Overwrite(utils.IntToBytes(500, 2, true), 13) // duel state

							sock := GetSocket(opponent.UserID)
							if sock != nil {
								sock.Write(r)
							}
						}
						characters, err := c.GetNearbyCharacters()
						if err != nil {
							log.Println(err)
							return
						}
						for _, chars := range characters {
							delete(chars.OnSight.Players, c.ID)
						}
					}
				}
			}
		}
	}
}
func (c *Character) RespawnCounter(seconds byte) {

	resp := RESPAWN_COUNTER
	resp[7] = seconds
	c.Socket.Write(resp)

	if seconds > 0 {
		time.AfterFunc(time.Second, func() {
			c.RespawnCounter(seconds - 1)
		})
	}
}

func (c *Character) Teleport(coordinate *utils.Location) []byte {

	if c.Respawning {
		return nil
	}

	c.SetCoordinate(coordinate)

	resp := TELEPORT_PLAYER
	resp.Insert(utils.FloatToBytes(coordinate.X, 4, true), 5) // coordinate-x
	resp.Insert(utils.FloatToBytes(coordinate.Y, 4, true), 9) // coordinate-x

	return resp
}

func (c *Character) ActivityStatus(remainingTime int) {

	var msg string
	if c.IsActive || remainingTime == 0 {
		msg = "Your character has been activated."
		c.IsActive = true

		data, err := c.SpawnCharacter()
		if err != nil {
			return
		}

		p := &nats.CastPacket{CastNear: true, CharacterID: c.ID, Data: data, Type: nats.PLAYER_SPAWN}
		if err = p.Cast(); err != nil {
			return
		}

	} else {
		msg = fmt.Sprintf("Your character will be activated %d seconds later.", remainingTime)

		if c.IsOnline {
			time.AfterFunc(time.Second, func() {
				if !c.IsActive {
					c.ActivityStatus(remainingTime - 1)
				}
			})
		}
	}

	info := messaging.InfoMessage(msg)
	if c.Socket != nil {
		c.Socket.Write(info)
	}
}

func contains(v int64, a []int64) bool {
	for _, i := range a {
		if i == v {
			return true
		}
	}
	return false
}

func unorderedEqual(first, second []int64, count int) bool {
	exists := make(map[int64]bool)
	match := 0
	for _, value := range first {
		exists[value] = true
	}
	for _, value := range second {
		if match >= count {
			return true
		}
		if !exists[value] {
			return false
		}
		match++
	}
	return true
}

func BonusActive(first, second []int64) bool {
	exists := make(map[int64]bool)
	for _, value := range first {
		exists[value] = true
	}
	for _, value := range second {
		if !exists[value] {
			return false
		}
	}
	return true
}

func (c *Character) ItemSetEffects(indexes []int16) []int64 {
	slots, _ := c.InventorySlots()
	playerItems := []int64{}
	for _, i := range indexes {
		if (i == 3 && c.WeaponSlot == 4) || (i == 4 && c.WeaponSlot == 3) {
			continue
		}
		item := slots[i]
		if item.ItemID == 0 {
			continue
		}
		iteminfo, ok := GetItemInfo(item.ItemID)
		if !ok || iteminfo == nil {
			continue
		} else if iteminfo.MinLevel > c.Level || (iteminfo.MaxLevel < c.Level && iteminfo.MaxLevel > 0) {
			continue
		}

		playerItems = append(playerItems, item.ItemID)

	}
	setEffect := []int64{}
	itemsets := GetItemsSets()
	for _, i := range playerItems {
		for _, sets := range itemsets {
			if contains(i, sets.SetItemsIDs) {
				if unorderedEqual(playerItems, sets.SetItemsIDs, sets.SetItemCount) {
					buffEffects := sets.SetBonusIDs
					for _, effect := range buffEffects {
						if effect == 0 {
							continue
						}
						if !contains(effect, setEffect) {
							setEffect = append(setEffect, effect)

						}
					}
				}
			}
		}
	}
	return setEffect
}
func (c *Character) applySetEffect(bonuses []int64, st *Stat) {
	for _, id := range bonuses {
		item, ok := GetItemInfo(id)
		if !ok && item == nil {
			continue
		}

		st.STRBuff += item.STR
		st.DEXBuff += item.DEX
		st.INTBuff += item.INT
		st.WindBuff += item.Wind
		st.WaterBuff += item.Water
		st.FireBuff += item.Fire

		st.DEF += item.Def + ((item.BaseDef1 + item.BaseDef2 + item.BaseDef3) / 3)
		st.DefRate += item.DefRate

		st.ArtsDEF += item.ArtsDef
		st.ArtsDEFRate += item.ArtsDefRate

		st.MaxHP += item.MaxHp
		st.MaxCHI += item.MaxChi

		st.Accuracy += item.Accuracy
		st.Dodge += item.Dodge

		st.MinATK += item.BaseMinAtk + item.MinAtk
		st.MaxATK += item.BaseMaxAtk + item.MaxAtk
		st.ATKRate += item.AtkRate

		st.MinArtsATK += item.MinArtsAtk
		st.MaxArtsATK += item.MaxArtsAtk
		st.ArtsATKRate += item.ArtsAtkRate

		st.ExpMultiplier += item.ExpRate / 100
		st.DropMultiplier += item.DropRate / 1000

		st.AdditionalRunningSpeed += item.RunningSpeed

		st.HPRecoveryRate += item.HPRecoveryRate
		st.CHIRecoveryRate += item.CHIRecoveryRate

		if item.SpecialEffectID == 5 { //increased pvp damage
			st.IncreasedPVPDamageRate += item.SpecialEffectValue
			st.IncreasedPVPDamageProbablility += item.SpecialEffectProbability
		} else if item.SpecialEffectID == 4 { //critical strike
			st.CriticalRate += item.SpecialEffectValue
			st.CriticalProbability += item.SpecialEffectProbability
		} else if item.SpecialEffectID == 3 { //damage converted to hp
			st.DamageConvertedToHpRate += item.SpecialEffectValue
			st.DamageConvertedToHpProbability += item.SpecialEffectProbability
		} else if item.SpecialEffectID == 2 { //Damage absorbed
			st.DamagedAbsorbedProbabilty += item.SpecialEffectProbability
			st.DamagedAbsobedRate += item.SpecialEffectValue
		} else if item.SpecialEffectID == 1 { //Reflect
			st.DamageReflectedProbabilty += item.SpecialEffectProbability
			st.DamageReflectedRate += item.SpecialEffectValue
		}

		st.PVPdmg += item.PVPdmg
		st.PVPsdmg += item.PVPsdmg
		st.AdditionalPVPdefRate += float32(item.PVPdefRate) / 1000
		st.AdditionalPVPsdefRate += float32(item.PVPsdefRate) / 1000

	}

}
func (c *Character) applyJudgementEffect(bonusID int64, st *Stat) {
	item := ItemJudgements[int(bonusID)]
	if item == nil {
		return
	}
	st.STRBuff += item.StrBonus
	st.DEXBuff += item.DexBonus
	st.INTBuff += item.IntBonus
	st.WindBuff += item.Wind
	st.WaterBuff += item.Water
	st.FireBuff += item.Fire

	st.DEF += item.DEF
	//st.DefRate += item.DefRate

	st.ArtsDEF += item.Sdef
	//st.ArtsDEFRate += item.ArtsDefRate

	st.MaxHP += item.MaxHpBonus
	st.MaxCHI += item.MaxChiBonus

	st.Accuracy += item.AccuracyBonus
	st.Dodge += item.DodgeBonus

	st.MinATK += item.AttackBonus
	st.MaxATK += item.AttackBonus

	st.AttackSpeed += item.AttackSpeedBonus
	st.AdditionalSkillRadius += item.ArtsRangeBonus
}
func (c *Character) ItemEffects(st *Stat, start, end int16) error {

	slots, err := c.InventorySlots()
	if err != nil {
		return err
	}

	indexes := []int16{}

	for i := start; i <= end; i++ {
		slot := slots[i]
		if c.Map == 251 && (slot.ItemID < 203001085 || slot.ItemID > 203001180) {
			continue
		}
		if start == 0x0B || start == 0x155 { //invenotory buff items
			if slot != nil && slot.Activated && slot.InUse {
				indexes = append(indexes, i)
			}
		} else {
			indexes = append(indexes, i)
		}
	}

	maxPoison, maxConfusion, maxPara := 0, 0, 0
	setEffects := c.ItemSetEffects(indexes)
	c.applySetEffect(setEffects, st)

	for _, i := range indexes {
		item := slots[i]
		if item == nil || item.ItemID == 0 {
			continue
		}
		info, ok := GetItemInfo(item.ItemID)
		if !ok && info == nil {
			continue
		}
		if info.Type == BOOK_OF_PET_TYPE {
			continue
		}

		if item.ItemID != 0 {
			if ((i == 3 && c.WeaponSlot == 4) || (i == 4 && c.WeaponSlot == 3)) && info.Type != 106 {
				continue
			}
			slotId := i
			if slotId == 4 {
				slotId = 3
			}

			if (info == nil || slotId != int16(info.Slot) || c.Level < info.MinLevel || (info.MaxLevel > 0 && c.Level > info.MaxLevel)) &&
				!(start == 0x0B || start == 0x155) {
				//	c.CanMove = false //test
				continue
			}

			ids := []int64{item.ItemID}

			if item.ItemID == 113110003 { //Anti-exp amulet
				c.AntiExp = true
			}

			if item.ItemType != 0 {
				c.applyJudgementEffect(item.JudgementStat, st)
			}
			for _, u := range item.GetUpgrades() {
				if u == 0 {
					break
				}
				ids = append(ids, int64(u))
			}
			if item.Appearance >= 240000051 && item.Appearance <= 240000086 {
				ids = append(ids, item.Appearance)
			}

			for _, s := range item.GetSockets() {
				if s == 0 || info.CanCreateSocket != 1 {
					break
				}
				ids = append(ids, int64(s))
			}
			for _, id := range ids {
				item, ok := GetItemInfo(id)
				if !ok && item == nil {
					continue
				}
				st.STRBuff += item.STR
				st.DEXBuff += item.DEX
				st.INTBuff += item.INT
				st.WindBuff += item.Wind
				st.WaterBuff += item.Water
				st.FireBuff += item.Fire

				st.DEF += item.Def + ((item.BaseDef1 + item.BaseDef2 + item.BaseDef3) / 3)
				st.DefRate += item.DefRate

				st.ArtsDEF += item.ArtsDef
				st.ArtsDEFRate += item.ArtsDefRate

				st.MaxHP += item.MaxHp
				st.MaxCHI += item.MaxChi

				st.Accuracy += item.Accuracy
				st.Dodge += item.Dodge

				st.MinATK += item.BaseMinAtk + item.MinAtk
				st.MaxATK += item.BaseMaxAtk + item.MaxAtk
				st.ATKRate += item.AtkRate

				st.PoisonATK += item.PoisonATK
				st.PoisonDEF += item.PoisonDEF
				st.ParalysisATK += item.ParaATK
				st.ParalysisDEF += item.ParaDEF
				st.ConfusionATK += item.ConfusionATK
				st.ConfusionDEF += item.ConfusionDEF

				st.MinArtsATK += item.MinArtsAtk
				st.MaxArtsATK += item.MaxArtsAtk
				st.ArtsATKRate += item.ArtsAtkRate
				st.ExpMultiplier += item.ExpRate / 100
				st.DropMultiplier += item.DropRate / 1000

				st.AdditionalRunningSpeed += item.RunningSpeed
				st.HPRecoveryRate += item.HPRecoveryRate
				st.CHIRecoveryRate += item.CHIRecoveryRate

				if item.SpecialEffectID == 5 { //increased pvp damage
					st.IncreasedPVPDamageRate += item.SpecialEffectValue
					st.IncreasedPVPDamageProbablility += item.SpecialEffectProbability
				} else if item.SpecialEffectID == 4 { //critical strike
					st.CriticalRate += item.SpecialEffectValue
					st.CriticalProbability += item.SpecialEffectProbability
				} else if item.SpecialEffectID == 3 { //damage converted to hp
					st.DamageConvertedToHpRate += item.SpecialEffectValue
					st.DamageConvertedToHpProbability += item.SpecialEffectProbability
				} else if item.SpecialEffectID == 2 { //Damage absorbed
					st.DamagedAbsorbedProbabilty += item.SpecialEffectProbability
					st.DamagedAbsobedRate += item.SpecialEffectValue
				} else if item.SpecialEffectID == 1 { //Reflect
					st.DamageReflectedProbabilty += item.SpecialEffectProbability
					st.DamageReflectedRate += item.SpecialEffectValue
				}

				st.PVPdmg += item.PVPdmg
				st.PVPsdmg += item.PVPsdmg
				st.AdditionalPVPdefRate += float32(item.PVPdefRate) / 1000
				st.AdditionalPVPsdefRate += float32(item.PVPsdefRate) / 1000

				if item.PoisonTime > maxPoison {
					maxPoison = item.PoisonTime
					st.PoisonTime = item.PoisonTime / 1000
				}
				if item.ParaTime > maxPara {
					maxPara = item.ParaTime
					st.Paratime = item.ParaTime / 1000
				}
				if item.ConfusionTime > maxConfusion {
					maxConfusion = item.ConfusionTime
					st.ConfusionTime = item.ConfusionTime / 1000
				}

			}
		}
	}

	return nil
}
func (c *Character) GetExpAndSkillPts() []byte {

	resp := EXP_SKILL_PT_CHANGED
	resp.Insert(utils.IntToBytes(uint64(c.Exp), 8, true), 5)                        // character exp
	resp.Insert(utils.IntToBytes(uint64(c.Socket.Skills.SkillPoints), 4, true), 13) // character skill points
	return resp
}

func (c *Character) GetPTS() []byte {

	resp := PTS_CHANGED
	resp.Insert(utils.IntToBytes(uint64(c.PTS), 4, true), 6) // character pts
	return resp
}

func (c *Character) AddExp(amount int64) ([]byte, bool) {

	if amount <= 0 {
		return nil, false
	}

	if c.Reborns > 0 && c.Level < 101 {
		if c.Level < 101 {
			amount /= int64(c.Reborns) + 1
		} /*else if c.Level > 100 && c.Level < 201 {
			amount -= (amount / int64(c.Reborns))
		}*/
	}
	if c.AntiExp {
		return nil, false
	}

	c.ExpMutex.Lock()
	defer c.ExpMutex.Unlock()
	pvpExpMultiplier := 0.0
	if funk.Contains(PVPServers, int16(c.Socket.User.ConnectedServer)) {
		pvpExpMultiplier = 0.25
	}

	expMultipler := c.Socket.Stats.ExpMultiplier + pvpExpMultiplier
	add := int64(float64(amount) * (expMultipler * EXP_RATE))
	if add <= 0 {
		return nil, false
	}
	exp := c.Exp + add
	spIndex := utils.SearchUInt64(SkillPoints, uint64(c.Exp))
	canLevelUp := true
	if exp > 233332051410 && c.Level <= 100 {
		exp = 233332051410
	}
	if exp > 544951059310 && c.Level <= 200 {
		exp = 544951059310
	}
	c.Exp = exp
	spIndex2 := utils.SearchUInt64(SkillPoints, uint64(c.Exp))

	//resp := c.GetExpAndSkillPts()

	st := c.Socket.Stats
	if st == nil {
		return nil, false
	}

	levelUp := false
	level := int16(c.Level)
	targetExp := EXPs[level].Exp
	skPts, sp := 0, 0
	np := 0                                             //nature pts
	for exp >= targetExp && level < 299 && canLevelUp { // Levelling up && level < 299

		if c.Type <= 59 && level >= 100 {
			level = 100
			canLevelUp = false
		} else if c.Type <= 69 && level >= 200 {
			level = 200
			canLevelUp = false
		} else {
			level++
			st.HP = st.MaxHP
			sp += int(level/10) + 4

			if c.Level < 101 {
				sp += int(c.Reborns)
			}

			targetExp = EXPs[level].Exp
			levelUp = true
		}
		if level >= 101 && level < 300 { //divine nature stats
			np += 4
			skPts += EXPs[level].SkillPoints
		}
	}
	c.Level = int(level)
	resp := EXP_SKILL_PT_CHANGED
	if level <= 100 {
		skPts = spIndex2 - spIndex
	}
	c.Socket.Skills.SkillPoints += skPts
	if levelUp {
		if c.Level < 101 {
			c.Socket.Skills.SkillPoints += int(3 * c.Reborns)
		}
		st.StatPoints += sp
		st.NaturePoints += np
		resp.Insert(utils.IntToBytes(uint64(exp), 8, true), 5)                          // character exp
		resp.Insert(utils.IntToBytes(uint64(c.Socket.Skills.SkillPoints), 4, true), 13) // character skill points
		if c.GuildID > 0 {
			guild, err := FindGuildByID(c.GuildID)
			if err == nil && guild != nil {
				guild.InformMembers(c)
			}
		}

		resp.Concat(messaging.SystemMessage(messaging.LEVEL_UP))
		resp.Concat(messaging.SystemMessage(messaging.LEVEL_UP_SP))
		resp.Concat(messaging.InfoMessage(c.GetLevelText()))

		spawnData, err := c.SpawnCharacter()
		if err == nil {
			c.Socket.Write(spawnData)
			go c.Update()
			//resp.Concat(spawnData)
		}
	} else {
		resp.Insert(utils.IntToBytes(uint64(exp), 8, true), 5)                          // character exp
		resp.Insert(utils.IntToBytes(uint64(c.Socket.Skills.SkillPoints), 4, true), 13) // character skill points
	}
	go c.Socket.Skills.Update()
	go c.Update()
	return resp, levelUp
}

func (c *Character) CombineItems(where, to int16) (int64, int16, error) {

	c.AntiDupeMutex.Lock()
	defer c.AntiDupeMutex.Unlock()
	invSlots, err := c.InventorySlots()
	if err != nil {
		return 0, 0, err
	}

	whereItem := invSlots[where]
	toItem := invSlots[to]

	if toItem.Quantity >= 10000 || whereItem.Quantity >= 10000 {
		return 0, 0, nil
	}
	itemInfo, ok := GetItemInfo(whereItem.ItemID)
	if !ok || !itemInfo.Stackable {
		return 0, 0, nil
	}

	if toItem.ItemID == whereItem.ItemID && whereItem.Plus == toItem.Plus {
		if toItem.Quantity+whereItem.Quantity > 10000 {
			diff := toItem.Quantity + whereItem.Quantity - 10000
			toItem.Quantity = 10000
			whereItem.Quantity = diff
			toItem.Update()
			whereItem.Update()

			return 0, 0, nil
		}

		toItem.Quantity += whereItem.Quantity

		toItem.Update()
		whereItem.Delete()
		*whereItem = *NewSlot()

	} else {
		return 0, 0, nil
	}
	return toItem.ItemID, int16(toItem.Quantity), nil
}

func (c *Character) BankItems() []byte {

	slots, err := c.InventorySlots()
	if err != nil {
		return nil
	}

	c.TypeOfBankOpened = 1

	gold := GET_GOLD
	gold.Insert(utils.IntToBytes(uint64(c.Gold), 8, true), 6)                  // gold
	gold.Insert(utils.IntToBytes(uint64(c.Socket.User.BankGold), 8, true), 14) // bank gold
	c.Socket.Write(gold)

	resp := BANK_ITEMS
	resp.SetLength(4)

	expbag := BANK_EXPANDED
	expiration := "UNLIMITED"
	expbag.Overwrite([]byte(expiration), 7) // bag expiration
	resp.Concat(expbag)

	for slotID := int16(67); slotID <= 306; slotID++ {
		slot := slots[slotID]
		go c.Socket.Write(slot.GetData(slotID))
	}

	return resp
}

func (c *Character) GetGold() []byte {

	user, err := FindUserByID(c.UserID)
	if err != nil || user == nil {
		return nil
	}
	/*if c.Gold < 0 {
		c.Gold = 0
	}*/

	resp := GET_GOLD
	resp.Insert(utils.IntToBytes(c.Gold, 8, true), 6)         // gold
	resp.Insert(utils.IntToBytes(user.BankGold, 8, true), 14) // bank gold

	return resp
}
func (c *Character) ChangeName(newname string) ([]byte, error) {
	ok, err := IsValidUsername(newname)
	if err != nil {
		return nil, err
	} else if !ok {
		return messaging.SystemMessage(messaging.INVALID_NAME), nil
	}
	if ok {
		c.Name = newname
		c.Update()
	}
	CHARACTER_MENU := utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x09, 0x09, 0x00, 0x55, 0xAA}
	resp := CHARACTER_MENU
	if c != nil {
		c.Socket.User.ConnectingIP = c.Socket.ClientAddr
		c.Socket.User.ConnectingTo = c.Socket.User.ConnectedServer
		c.Logout()
	}
	return resp, nil
}

func (c *Character) MapBookShow(slotID uint64) []byte {

	if c.MapBook == "" {
		c.MapBook = "0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0"
	}
	mapbook := strings.Split(c.MapBook, ",")

	resp := MAP_BOOK_SHOW
	index := 14
	resp.Overwrite(utils.IntToBytes(slotID, 2, true), index)
	index += 2

	for raw := 0; raw < 6; raw++ {
		mapid, _ := strconv.ParseInt(mapbook[raw*3], 10, 64)
		resp.Overwrite(utils.IntToBytes(uint64(mapid), 2, true), index)
		index++
		posX, _ := strconv.ParseInt(mapbook[raw*3+1], 10, 64)
		resp.Overwrite(utils.IntToBytes(uint64(posX), 4, true), index)
		index += 2
		posY, _ := strconv.ParseInt(mapbook[raw*3+2], 10, 64)
		resp.Overwrite(utils.IntToBytes(uint64(posY), 4, true), index)
		index += 2
	}
	return resp
}

func (c *Character) ChangeMap(mapID int16, coordinate *utils.Location, args ...interface{}) ([]byte, error) {
	/*if !funk.Contains(unlockedMaps, mapID) {
		return messaging.SystemMessage(messaging.CANNOT_MOVE_TO_AREA), nil
	}*/
	resp := utils.Packet{}
	/*if Maps[int(mapID)] == nil {
		return messaging.SystemMessage(messaging.CANNOT_MOVE_TO_AREA), nil
	}
	if c.Level < Maps[int(mapID)].MinLevelRequirment && Maps[int(mapID)].MinLevelRequirment > 0 {
		if c.Map == mapID && mapID > 1 {
			return c.ChangeMap(1, nil)
		}
		return messaging.SystemMessage(messaging.NO_LEVEL_REQUIREMENT), nil
	}*/ //restrict using form
	if mapID == 10 && c.Morphed {
		slots, _ := c.InventorySlots()
		for _, slot := range slots {
			iteminfo, ok := GetItemInfo(slot.ItemID)
			if !ok {
				continue
			}
			if slot != nil && iteminfo != nil {
				if iteminfo.Type == 174 && slot.Activated && slot.InUse {
					use, err := c.UseConsumable(slot, slot.SlotID)
					if err != nil {
						return nil, err
					}
					c.Socket.Write(use)
					time.Sleep(time.Duration(1) * time.Second)
				}

			}
		}
	}

	if c.Respawning {
		c.Respawning = false
		save := SavePoints[int(c.Map)]
		point := &utils.Location{X: 100, Y: 100}
		if c.Map == 255 { //faction war map
			if c.Faction == 1 {
				point = &utils.Location{X: 325, Y: 465}
			}
			if c.Faction == 2 {
				point = &utils.Location{X: 179, Y: 45}
			}
		} else if c.Map == 230 {
			if c.Faction == 1 {
				x := 75.0
				y := 45.0
				point = ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", x, y))
			} else {
				x := 81.0
				y := 475.0
				point = ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", x, y))
			}
		} else {
			point = &utils.Location{X: save.X, Y: save.Y}
		}

		teleportData := c.Teleport(point)
		resp.Concat(teleportData)

		c.IsActive = false
		stat := c.Socket.Stats
		stat.HP = stat.MaxHP
		stat.CHI = stat.MaxCHI
		hpData := c.GetHPandChi()
		resp.Concat(hpData)
		c.Socket.Write(resp)

		p := nats.CastPacket{CastNear: true, CharacterID: c.ID, Type: nats.PLAYER_RESPAWN}
		p.Cast()
	}
	if !c.IsAllowedInMap(mapID) {
		return messaging.SystemMessage(messaging.NO_LEVEL_REQUIREMENT), nil
	}
	if c.TradeID != "" && c.Map != 243 {
		text := "Name: " + c.Name + "(" + c.UserID + ") tried to change map while trading"
		utils.NewLog("logs/cheat_alert.txt", text)
		return messaging.SystemMessage(10053), nil //Cannot do that while trading
	}

	resp, r := MAP_CHANGED, utils.Packet{}
	//	pvpserver, _ := utils.Contains(PVPServers, int16(c.Socket.User.ConnectedServer))
	/*	if Maps[int(mapID)].IsWarZone || pvpserver {
		resp.Overwrite([]byte{0x01}, 29)
		resp.Overwrite([]byte{0x01}, 54)
	}*/
	c.Map = mapID
	c.EndPvP()

	if coordinate == nil { // if no coordinate then teleport home
		d := SavePoints[int(mapID)]
		if d == nil {
			d = &SavePoint{X: 100, Y: 100}
		}
		coordinate = &utils.Location{X: d.X, Y: d.Y}
	}
	//LoadQuestItems
	//c.GetMapQuestMobs()
	//END
	/*qList, _ := FindQuestsByCharacterID(c.ID)
	for _, quest := range qList {
		if quest.QuestState != 2 {
			questresp, _ := c.LoadReturnQuests(quest.ID, quest.QuestState)
			resp.Concat(questresp)
		}

	}*/
	if funk.Contains(sharedMaps, mapID) { // shared map
		c.Socket.User.ConnectedServer = 1
	} else if c.Socket.User.ConnectedServer >= 1 && c.Socket.User.ConnectedServer <= 7 {
		c.Socket.User.ConnectedServer = c.Socket.User.SelectedServerID
	} else {
		c.Socket.User.ConnectedServer = 1
	}

	if c.GuildID > 0 {
		guild, err := FindGuildByID(c.GuildID)
		if err != nil {
			return nil, err
		}
		if guild != nil {
			guild.InformMembers(c)
		}
	}

	consItems, _ := FindConsignmentItemsBySellerID(c.ID)
	consItems = (funk.Filter(consItems, func(item *ConsignmentItem) bool {
		return item.IsSold
	}).([]*ConsignmentItem))
	if len(consItems) > 0 {
		r.Concat(CONSIGMENT_ITEM_SOLD)
	}

	if c.AidMode {
		c.AidMode = false
		r.Concat(c.AidStatus())
	}
	RemovePetFromRegister(c)
	//RemoveFromRegister(c)
	//GenerateID(c)

	c.SetCoordinate(coordinate)

	if len(args) == 0 { // not logging in
		c.OnSight.DropsMutex.Lock()
		c.OnSight.Drops = map[int]interface{}{}
		c.OnSight.DropsMutex.Unlock()

		c.OnSight.MobMutex.Lock()
		c.OnSight.Mobs = map[int]interface{}{}
		c.OnSight.MobMutex.Unlock()

		c.OnSight.NpcMutex.Lock()
		c.OnSight.NPCs = map[int]interface{}{}
		c.OnSight.NpcMutex.Unlock()

		c.OnSight.PetsMutex.Lock()
		c.OnSight.Pets = map[int]interface{}{}
		c.OnSight.PetsMutex.Unlock()

		c.OnSight.BabyPetsMutex.Lock()
		c.OnSight.BabyPets = map[int]interface{}{}
		c.OnSight.BabyPetsMutex.Unlock()
	}

	resp[13] = byte(mapID)                                     // map id
	resp.Insert(utils.FloatToBytes(coordinate.X, 4, true), 14) // coordinate-x
	resp.Insert(utils.FloatToBytes(coordinate.Y, 4, true), 18) // coordinate-y
	resp[36] = byte(mapID)                                     // map id
	resp.Insert(utils.FloatToBytes(coordinate.X, 4, true), 46) // coordinate-x
	resp.Insert(utils.FloatToBytes(coordinate.Y, 4, true), 50) // coordinate-y
	resp[61] = byte(mapID)                                     // map id

	spawnData, _ := c.SpawnCharacter()
	r.Concat(spawnData)
	resp.Concat(r)
	c.Socket.Write(c.Socket.User.GetTime())

	/*if funk.Contains(DarkZones, mapID) {
		resp.Concat(DARK_MODE_ACTIVE)
	}*/

	statdata, _ := c.GetStats()
	resp.Concat(statdata)

	go time.AfterFunc(time.Second*time.Duration(5), func() {
		c.HousingDetails()
	})

	return resp, nil
}

func DoesSlotAffectStats(slotNo int16) bool {
	return slotNo < 0x0B || (slotNo >= 0x0133 && slotNo <= 0x013B) || (slotNo >= 0x18D && slotNo <= 0x192)
}

func (c *Character) RemoveItem(slotID int16) ([]byte, error) {

	slots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}
	if slotID >= 67 && slotID <= 306 && c.TypeOfBankOpened == 2 {
		return nil, err
	}

	item := slots[slotID]

	resp := ITEM_REMOVED
	resp.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), 9) // item id
	resp.Insert(utils.IntToBytes(uint64(slotID), 2, true), 13)     // slot id

	affects, activated := DoesSlotAffectStats(slotID), item.Activated
	if affects || activated {
		item.Activated = false
		item.InUse = false

		statData, err := c.GetStats()
		if err != nil {
			return nil, err
		}

		resp.Concat(statData)
	}

	info, ok := GetItemInfo(item.ItemID)
	if !ok {
		return nil, errors.New("item not found")
	}
	if activated {
		if item.ItemID == 100080008 { // eyeball of divine
			c.DetectionMode = false
		}

		if info != nil && info.GetType() == FORM_TYPE {
			c.Morphed = false
			c.MorphedNPCID = 0
			resp.Concat(FORM_DEACTIVATED)
		}

		data := ITEM_EXPIRED
		data.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), 6)
		resp.Concat(data)
	}

	if affects {
		itemsData, err := c.ShowItems()
		if err != nil {
			return nil, err
		}

		p := nats.CastPacket{CastNear: true, CharacterID: c.ID, Data: itemsData, Type: nats.SHOW_ITEMS}
		if err = p.Cast(); err != nil {
			return nil, err
		}

		resp.Concat(itemsData)
	}
	item.Delete()
	*item = *NewSlot()

	return resp, nil
}

func (c *Character) GetStats() ([]byte, error) {

	if c == nil {
		log.Println("c is nil")
		return nil, nil

	} else if c.Socket == nil {
		log.Println("socket is nil")
		return nil, nil
	}

	st := c.Socket.Stats
	if st == nil {
		return nil, nil
	}
	err := st.Calculate()
	if err != nil {
		return nil, err
	}

	slots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}
	resp := GET_STATS

	index := 5
	resp.Insert(utils.IntToBytes(uint64(c.Level), 4, true), index) // character level
	index += 4

	duelState := 1
	if c.DuelID > 0 && c.DuelStarted {
		duelState = 500
	}

	resp.Insert(utils.IntToBytes(uint64(duelState), 2, true), index) // duel state
	index += 4

	resp.Insert(utils.IntToBytes(uint64(st.StatPoints), 2, true), index) // stat points
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.NaturePoints), 2, true), index) // divine stat points
	index += 2

	resp.Insert(utils.IntToBytes(uint64(c.Socket.Skills.SkillPoints), 4, true), index) // character skill points
	index += 6

	resp.Insert(utils.IntToBytes(uint64(c.Exp), 8, true), index) // character experience
	index += 8

	resp.Insert(utils.IntToBytes(uint64(c.AidTime), 4, true), index) // remaining aid
	index += 4
	index++

	targetExp := EXPs[int16(c.Level)].Exp
	resp.Insert(utils.IntToBytes(uint64(targetExp), 8, true), index) // character target experience
	index += 8

	resp.Insert(utils.IntToBytes(uint64(st.STR), 2, true), index) // character str
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.STR+st.STRBuff), 2, true), index) // character str buff
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.DEX), 2, true), index) // character dex
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.DEX+st.DEXBuff), 2, true), index) // character dex buff
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.INT), 2, true), index) // character int
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.INT+st.INTBuff), 2, true), index) // character int buff
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.Wind), 2, true), index) // character wind
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.Wind+st.WindBuff), 2, true), index) // character wind buff
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.Water), 2, true), index) // character water
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.Water+st.WaterBuff), 2, true), index) // character water buff
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.Fire), 2, true), index) // character fire
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.Fire+st.FireBuff), 2, true), index) // character fire buff
	index += 7

	runningspeed := c.RunningSpeed + st.AdditionalRunningSpeed
	if c.IsinWar && runningspeed > 15 {
		runningspeed = 15
	}

	resp.Insert(utils.FloatToBytes(runningspeed, 4, true), index) // character running speed
	index += 4                                                    //10 volt
	resp.Insert(utils.IntToBytes(uint64(1000), 4, true), index)   //attack speed
	index += 4
	weapon := slots[c.WeaponSlot]
	if c.Type == MONK || c.Type == DIVINE_MONK || c.Type == DARKNESS_MONK {
		resp.Insert([]byte{0x03, 0x41}, index)
	} else if weapon.ItemID != 0 {
		itemInfo, ok := GetItemInfo(weapon.ItemID)
		if !ok {
			return nil, errors.New("item not found")
		}
		if itemInfo.Type == 105 || itemInfo.Type == 108 || c.Type == MONK || c.Type == DIVINE_MONK || c.Type == DARKNESS_MONK {
			resp.Insert([]byte{0x03, 0x41}, index)
		} else {
			resp.Insert([]byte{0x00, 0x40}, index)
		}
	} else {
		resp.Insert([]byte{0x00, 0x40}, index)
	}
	index += 2
	resp.Insert(utils.IntToBytes(uint64(st.MaxHP), 4, true), index) // character max hp
	index += 4

	resp.Insert(utils.IntToBytes(uint64(st.MaxCHI), 4, true), index) // character max chi
	index += 4

	resp.Insert(utils.IntToBytes(uint64(st.MinATK), 2, true), index) // character min atk
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.MaxATK), 2, true), index) // character max atk
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.DEF), 4, true), index) // character def
	index += 4
	resp.Insert(utils.IntToBytes(uint64(st.DEF), 4, true), index) // character def
	index += 4
	resp.Insert(utils.IntToBytes(uint64(st.DEF), 4, true), index) // character def
	index += 4

	resp.Insert(utils.IntToBytes(uint64(st.MinArtsATK), 4, true), index) // character min arts atk
	index += 4

	resp.Insert(utils.IntToBytes(uint64(st.MaxArtsATK), 4, true), index) // character max arts atk
	index += 4

	resp.Insert(utils.IntToBytes(uint64(st.ArtsDEF), 4, true), index) // character arts def
	index += 4

	resp.Insert(utils.IntToBytes(uint64(st.Accuracy), 2, true), index) // character accuracy
	index += 2

	resp.Insert(utils.IntToBytes(uint64(st.Dodge), 2, true), index) // character dodge
	index += 2
	index += 2
	resp.Insert(utils.IntToBytes(uint64(st.PoisonATK), 2, true), index) // character PoisonDamage
	index += 2
	resp.Insert(utils.IntToBytes(uint64(st.PoisonDEF), 2, true), index) // character PoisonDEF
	index += 2
	index++
	resp.Insert(utils.IntToBytes(uint64(st.ConfusionATK), 2, true), index) // character ParaATK
	index += 2
	resp.Insert(utils.IntToBytes(uint64(st.ConfusionDEF), 2, true), index) // character ParaDEF
	index += 2
	index++
	resp.Insert(utils.IntToBytes(uint64(st.ParalysisATK), 2, true), index) // character ConfusionATK
	index += 2
	resp.Insert(utils.IntToBytes(uint64(st.ParalysisDEF), 2, true), index) // character ConfusionDef
	index += 3

	resp.Overwrite(utils.IntToBytes(uint64(st.PVPdmg), 4, true), index) // character PoisonDamage
	index += 4
	resp.Overwrite(utils.IntToBytes(uint64(st.PVPsdmg), 4, true), index) // character PoisonDamage
	index += 4
	resp.Overwrite(utils.IntToBytes(uint64(st.PVPdef), 4, true), index) // character PoisonDamage
	index += 4
	resp.Overwrite(utils.IntToBytes(uint64(st.PVPsdef), 4, true), index) // character PoisonDamage
	index += 7

	statue := 0
	buff, err := FindBuffByID(10100, c.ID) // check for fire spirit
	if err != nil {
		return nil, err
	} else if buff != nil {
		statue = 5120
	}
	buff, err = FindBuffByID(10098, c.ID) // check for fire spirit
	if err != nil {
		return nil, err
	} else if buff != nil {
		statue = 5376
	}

	resp.Insert(utils.IntToBytes(uint64(statue), 4, true), index)
	index += 4
	resp.Insert([]byte{0x00, 0x13, 0x32, 0x30, 0x32, 0x32, 0x2D, 0x30, 0x33, 0x2D, 0x32, 0x34, 0x20, 0x31, 0x30, 0x3A, 0x35, 0x33, 0x3A, 0x33, 0x38}, index)

	resp.SetLength(int16(binary.Size(resp) - 6))
	resp.Concat(c.GetHPandChi()) // hp and chi
	return resp, nil
}

func IntToByteArray(num int64) []byte {
	size := int(unsafe.Sizeof(num))
	arr := make([]byte, size)
	for i := 0; i < size; i++ {
		byt := *(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(&num)) + uintptr(i)))
		arr[i] = byt
	}
	return arr
}
func (c *Character) BSUpgrade(slotID int64, stones []*InventorySlot, luck, protection *InventorySlot, stoneSlots []int64, luckSlot, protectionSlot int64) ([]byte, error) {

	slots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}

	if c.TradeID != "" {
		text := "Name: " + c.Name + "(" + c.UserID + ") tried to plus items while trading."
		utils.NewLog("logs/cheat_alert.txt", text)
		return messaging.SystemMessage(10053), nil //Cannot do that while trading
	}

	buff, err := FindBuffByID(60030, c.ID)
	if err == nil {
		if buff != nil {
			return messaging.SystemMessage(messaging.STR_COOLDOWN), nil
		}
	}

	item := slots[slotID]

	if item.Plus >= 15 { // cannot be upgraded more
		resp := utils.Packet{0xAA, 0x55, 0x31, 0x00, 0x54, 0x02, 0xA6, 0x0F, 0x01, 0x00, 0xA3, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}
		resp.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), 9) // item id
		resp.Insert(utils.IntToBytes(uint64(slotID), 2, true), 17)     // slot id
		resp.Insert(item.GetUpgrades(), 19)                            // item upgrades
		resp[34] = byte(item.SocketCount)                              // socket count
		resp.Insert(item.GetSockets(), 35)                             // item sockets

		return resp, nil
	}

	info, ok := GetItemInfo(item.ItemID)
	if !ok {
		return nil, errors.New("BSUpgrade:Item not found")
	}
	cost := (info.BuyPrice / 10) * int64(item.Plus+1) * int64(math.Pow(2, float64(len(stones)-1)))

	if uint64(cost) > c.Gold {
		resp := messaging.SystemMessage(messaging.INSUFFICIENT_GOLD)
		return resp, nil

	} else if len(stones) == 0 {
		resp := messaging.SystemMessage(messaging.INCORRECT_GEM_QTY)
		return resp, nil
	}
	if protection != nil && item.Plus > 4 {
		if protection.ItemID == 97700564 {
			resp := utils.Packet{0xAA, 0x55, 0x09, 0x00, 0x54, 0x02, 0xA6, 0x0F, 0x00, 0x55, 0xAA}
			resp.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), 9) // item id
			return resp, nil
		}
	}
	stone := stones[0]
	stoneInfo, ok := GetItemInfo(stone.ItemID)
	if !ok {
		resp := messaging.SystemMessage(messaging.INCORRECT_GEM)
		return resp, nil
	}

	if uint64(item.Plus) < stoneInfo.MinUpgradeLevel || stoneInfo.ID > 268 {
		resp := messaging.SystemMessage(messaging.INCORRECT_GEM)
		return resp, nil
	}

	itemType := info.GetType()

	typeMatch := (stoneInfo.Type == 190 && itemType == PET_ITEM_TYPE) || (stoneInfo.Type == 191 && itemType == HT_ARMOR_TYPE) ||
		(stoneInfo.Type == 192 && (itemType == ACC_TYPE || itemType == MASTER_HT_ACC)) || (stoneInfo.Type == 194 && itemType == WEAPON_TYPE) || (stoneInfo.Type == 195 && itemType == ARMOR_TYPE) || (stoneInfo.Type == 192 && itemType == 131)

	if stoneInfo.Type == 229 && item.ItemType == 2 && item.JudgementStat > 1 {
		if stoneInfo.HtType == 36 && itemType == WEAPON_TYPE {
			typeMatch = true
		} else if stoneInfo.HtType == 37 && itemType == ARMOR_TYPE {
			typeMatch = true
		} else if stoneInfo.HtType == 38 && itemType == ACC_TYPE {
			typeMatch = true
		}
	}

	if !typeMatch {

		resp := utils.Packet{0xAA, 0x55, 0x09, 0x00, 0x54, 0x02, 0xA4, 0x0F, 0x00, 0x55, 0xAA}
		resp.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), 9) // item id
		return resp, nil
	}

	strrate := STRRates[item.Plus]

	strrate += int(float64(STRRates[item.Plus]) * STRHappyHourRate)
	strrate += int(float64(strrate) * float64(c.Socket.Stats.EnhancedProbabilitiesBuff) / 1000)

	rate := float64(strrate * len(stones))
	plus := item.Plus + 1

	if stone.Plus > 0 { // Precious Pendent or Ghost Dagger or Dragon Scale
		for i := 0; i < len(stones); i++ {
			for j := i; j < len(stones); j++ {
				if stones[i].Plus != stones[j].Plus { // mismatch stone plus
					resp := utils.Packet{0xAA, 0x55, 0x09, 0x00, 0x54, 0x02, 0xA4, 0x0F, 0x00, 0x55, 0xAA}
					resp.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), 9) // item id
					return resp, nil
				}
			}
		}

		plus = item.Plus + stone.Plus
		if plus > 15 {
			plus = 15
		}

		strrate := BEASTstrRates[plus-1]
		strrate += int(float64(STRRates[item.Plus]) * STRHappyHourRate)
		strrate += int(float64(c.Socket.Stats.EnhancedProbabilitiesBuff) / 1000)
		rate = float64(strrate * len(stones))
	}

	if luck != nil {
		luckInfo, ok := GetItemInfo(luck.ItemID)
		if !ok {
			return nil, errors.New("BSUpgrade:Luck item not found")
		}
		if luckInfo.Type == 164 { // charm of luck
			if luckInfo.HtType == 39 { // soul stone charm
				k := 0.0
				if luckInfo.SellPrice == 50 {
					k = 1
				} else if luckInfo.SellPrice == 70 {
					k = 2
				} else if luckInfo.SellPrice == 100 {
					k = 3
				} else if luckInfo.SellPrice == 130 {
					k = 4
				} else if luckInfo.SellPrice == 150 {
					k = 5
				}
				rate += rate * k / float64(len(stones))
			} else {
				k := float64(luckInfo.SellPrice) / 100
				rate += rate * k / float64(len(stones))
			}

		} else if luckInfo.Type == 219 { // bagua
			if byte(luckInfo.SellPrice) != item.Plus { // item plus not matching with bagua
				resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x54, 0x02, 0xB6, 0x0F, 0x55, 0xAA}
				return resp, nil

			} else if len(stones) < 3 {
				resp := messaging.SystemMessage(messaging.INCORRECT_GEM_QTY)
				return resp, nil
			}

			rate = 1000

			if luckInfo.SellPrice == 6 {
				bagRates := []int{30, 40}
				for i := 0; i < len(bagRates); i++ {
					seed := utils.RandInt(0, 100)
					if int(seed) > bagRates[i] {
						plus++
					}
				}
			} else {
				bagRates := []int{30, 40, 50}
				for i := 0; i < len(bagRates); i++ {
					seed := utils.RandInt(0, 100)
					if int(seed) > bagRates[i] {
						plus++
					}
				}
			}

		}
	}

	protectionInfo := &Item{}
	if protection != nil {
		protectionInfo, _ = GetItemInfo(protection.ItemID)
	}
	resp := utils.Packet{}
	if c.Gold < uint64(cost) {
		resp.Concat(messaging.SystemMessage(messaging.INSUFFICIENT_GOLD))
		return resp, nil
	}
	if !c.SubtractGold(uint64(cost)) {
		return nil, nil
	}

	if rate > 1000 {
		rate = 1000
	}

	seed := int(utils.RandInt(0, 1000))

	if c.ShowUpgradingRate {
		msg := fmt.Sprintf("Upgrading %s: %0.2f%% success.", info.Name, float32(rate)/10)
		c.Socket.Write(messaging.InfoMessage(msg))
		resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x54, 0x02, 0xBB, 0x0F, 0x55, 0xAA} // message

		return resp, nil
	}

	if float64(seed) < rate { // upgrade successful
		var codes []byte
		for i := item.Plus; i < plus; i++ {
			codes = append(codes, byte(stone.ItemID))
		}

		before := item.GetUpgrades()
		resp.Concat(item.Upgrade(int16(slotID), codes...))
		logger.Log(logging.ACTION_UPGRADE_ITEM, c.ID, fmt.Sprintf("Item (%d) upgraded: %s -> %s", item.ID, before, item.GetUpgrades()), c.UserID)
		rr := UPG_SUCCESS
		c.Socket.Write(rr)

	} else if itemType == HT_ARMOR_TYPE || itemType == PET_ITEM_TYPE ||
		(protection != nil && protectionInfo.GetType() == SCALE_TYPE) { // ht or pet item failed or got protection

		if protectionInfo.GetType() == SCALE_TYPE { // if scale
			if item.Plus < uint8(protectionInfo.SellPrice) {
				item.Plus = 0
			} else {
				item.Plus -= uint8(protectionInfo.SellPrice)
			}
		} else {
			if item.Plus < stone.Plus {
				item.Plus = 0
			} else {
				item.Plus -= stone.Plus
			}
		}

		upgs := item.GetUpgrades()
		for i := int(item.Plus); i < len(upgs); i++ {
			item.SetUpgrade(i, 0)
		}

		r := HT_UPG_FAILED
		r.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), 9) // item id
		r.Insert(utils.IntToBytes(uint64(slotID), 2, true), 17)     // slot id
		r.Insert(item.GetUpgrades(), 19)                            // item upgrades
		r[34] = byte(item.SocketCount)                              // socket count
		r.Insert(item.GetSockets(), 35)                             // item sockets

		resp.Concat(r)

	} else { // casual item failed so destroy it
		r := UPG_FAILED
		r.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), 9) // item id
		resp.Concat(r)

		itemsData, err := c.RemoveItem(int16(slotID))
		if err != nil {
			return nil, err
		}

		resp.Concat(itemsData)
	}

	for _, slot := range stoneSlots {
		resp.Concat(*c.DecrementItem(int16(slot), 1))
	}

	if luck != nil {
		resp.Concat(*c.DecrementItem(int16(luckSlot), 1))
	}

	if protection != nil {
		resp.Concat(*c.DecrementItem(int16(protectionSlot), 1))
	}

	err = item.Update()
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *Character) BSProduction(book *InventorySlot, materials []*InventorySlot, special *InventorySlot, prodSlot int16, bookSlot, specialSlot int16, materialSlots []int16, materialCounts []uint) ([]byte, error) {

	production, ok := GetProductionById(int(book.ItemID))
	if !ok {
		log.Println("Invalid production id")
		return nil, nil
	}

	if c.TradeID != "" {
		text := "Name: " + c.Name + "(" + c.UserID + ") tried to use composition while trading."
		utils.NewLog("logs/cheat_alert.txt", text)
		return messaging.SystemMessage(10053), nil //Cannot do that while trading
	}

	canProduce := true

	if production.Item2 > 0 && production.Count2 > 0 {
		iteminfo, _ := GetItemInfo(materials[0].ItemID)
		if production.Item2 != iteminfo.ID {
			canProduce = false
		} else if production.Count2 > int(materials[0].Quantity) {
			canProduce = false
		}
	}

	if production.Item3 > 0 && production.Count3 > 0 {
		iteminfo, _ := GetItemInfo(materials[1].ItemID)
		if production.Item3 != iteminfo.ID {
			canProduce = false
		} else if production.Count3 > int(materials[1].Quantity) {
			canProduce = false
		}
	}
	if production.Special > 0 && production.SpecialCount > 0 {
		if production.Special != special.ItemID {
			canProduce = false
		} else if production.SpecialCount > int(special.Quantity) {
			canProduce = false
		}
	}

	cost := uint64(production.Cost)
	if cost > c.Gold || !canProduce {
		resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x54, 0x04, 0x07, 0x10, 0x55, 0xAA}
		return resp, nil
	}
	c.LootGold(-cost)

	rate := float64(production.Probability)
	rate += rate * (float64(c.Socket.Stats.SyntheticCompositeBuff) / 1000)
	resp := &utils.Packet{}
	seed := int(utils.RandInt(0, 1000))
	if float64(seed) < rate { // Success
		productioniteminfo, _ := GetItemInfo(int64(production.Production))
		if productioniteminfo.TimerType > 0 {
			time := productioniteminfo.Timer
			rr, _, err := c.AddItem(&InventorySlot{ItemID: int64(production.Production), Quantity: uint(time)}, prodSlot, false)
			if err != nil {
				return nil, err
			}
			resp.Concat(*rr)
		} else {
			rr, _, err := c.AddItem(&InventorySlot{ItemID: int64(production.Production), Quantity: 1}, prodSlot, false)
			if err != nil {
				return nil, err
			}
			resp.Concat(*rr)
		}

		resp.Concat(PRODUCTION_SUCCESS)
		logger.Log(logging.ACTION_PRODUCTION, c.ID, fmt.Sprintf("Production (%d) success", book.ItemID), c.UserID)
		resp.Concat(*c.DecrementItem(int16(bookSlot), 1))

	} else { // Failed
		resp.Concat(PRODUCTION_FAILED)
		resp.Concat(c.GetGold())
		logger.Log(logging.ACTION_PRODUCTION, c.ID, fmt.Sprintf("Production (%d) failed", book.ItemID), c.UserID)
		if !production.KeepTheBook {
			resp.Concat(*c.DecrementItem(int16(bookSlot), 1))
		}
		resp.Concat(book.GetData(book.SlotID))
	}

	for i := 0; i < len(materialSlots); i++ {
		resp.Concat(*c.DecrementItem(int16(materialSlots[i]), uint(materialCounts[i])))
	}

	if special != nil {
		resp.Concat(*c.DecrementItem(int16(specialSlot), 1))
	}

	return *resp, nil
}
func (c *Character) AdvancedFusion(items []*InventorySlot, special *InventorySlot, prodSlot int16) ([]byte, bool, error) {

	if len(items) < 3 {
		return nil, false, nil
	}

	if c.TradeID != "" {
		text := "Name: " + c.Name + "(" + c.UserID + ") tried to fusion items while trading."
		utils.NewLog("logs/cheat_alert.txt", text)
		return messaging.SystemMessage(10053), false, nil //Cannot do that while trading
	}

	fusion := Fusions[int(items[0].ItemID)]
	if fusion == nil {
		return nil, false, nil
	}
	seed := int(utils.RandInt(0, 1000))

	if fusion.Item1 >= 2400200 && fusion.Item1 <= 2400530 {
		if items[0].Plus != 10 {
			return FUSION_FAILED, false, nil
		}

	}

	cost := uint64(fusion.Cost)
	if c.Gold < cost {
		return FUSION_FAILED, false, nil
	}

	if int(items[0].ItemID) != fusion.Item1 || int(items[1].ItemID) != fusion.Item2 || int(items[2].ItemID) != fusion.Item3 {
		return FUSION_FAILED, false, nil
	}
	if items[0].SlotID == items[1].SlotID || items[0].SlotID == items[2].SlotID || items[1].SlotID == items[2].SlotID {
		return FUSION_FAILED, false, nil
	}
	if special != nil && (int(special.ItemID) != fusion.SpecialItem || int(special.Quantity) < fusion.SpecialItemCount) {
		return FUSION_FAILED, false, nil
	}
	if !c.SubtractGold(cost) {
		return nil, false, nil
	}
	rate := float64(fusion.Probability)
	if special != nil {
		specialiteminfo, ok := GetItemInfo(special.ItemID)
		if ok {
			rate *= float64(specialiteminfo.SellPrice+100) / 100
		}
	}
	rate += rate * (float64(c.Socket.Stats.AdvancedCompositeBuff) / 1000)
	if float64(seed) < rate { // Success
		resp := utils.Packet{}
		itemData, _, err := c.AddItem(&InventorySlot{ItemID: int64(fusion.Production), Quantity: 1}, prodSlot, false)
		if err != nil {
			return nil, false, err
		} else if itemData == nil {
			return nil, false, nil
		}

		resp.Concat(*itemData)
		resp.Concat(FUSION_SUCCESS)
		logger.Log(logging.ACTION_ADVANCED_FUSION, c.ID, fmt.Sprintf("Advanced fusion (%d) success", items[0].ItemID), c.UserID)
		return resp, true, nil

	} else { // Failed
		resp := FUSION_FAILED
		resp.Concat(c.GetGold())
		logger.Log(logging.ACTION_ADVANCED_FUSION, c.ID, fmt.Sprintf("Advanced fusion (%d) failed", items[0].ItemID), c.UserID)
		return resp, false, nil
	}
}

func (c *Character) Dismantle(item, special *InventorySlot, bundle bool) ([]byte, bool, error) {
	meltings := GetMeltings()
	melting := meltings[int(item.ItemID)]
	if melting == nil {
		return nil, false, nil
	}
	cost := uint64(melting.Cost)

	if c.TradeID != "" {
		text := "Name: " + c.Name + "(" + c.UserID + ") tried to dismintle items while trading."
		utils.NewLog("logs/cheat_alert.txt", text)
		return messaging.SystemMessage(10053), false, nil //Cannot do that while trading
	}

	if c.Gold < cost {
		return nil, false, nil
	}

	slots, err := c.InventorySlots()
	if err != nil {
		return nil, false, nil
	}

	_, err = c.FindFreeSlots(3)
	if err != nil {
		return messaging.SystemMessage(messaging.DISMINTLE_NOT_ENOUGH_SPACE), true, nil
	}

	if !c.SubtractGold(cost) {
		return nil, false, nil
	}

	info, ok := GetItemInfo(item.ItemID)
	if !ok {
		return nil, false, nil
	}

	profitmin := utils.RandFloat(1, float64(melting.GoldMultiplier)) * float64(info.BuyPrice)
	profitmax := utils.RandFloat(1, float64(melting.GoldMultiplier)) * float64(info.BuyPrice*2)
	profit := utils.RandInt(int64(profitmin), int64(profitmax))

	if profit < 0 {
		return nil, false, nil
	}

	c.LootGold(uint64(profit))

	resp := utils.Packet{}
	r := DISMANTLE_SUCCESS
	r.Insert(utils.IntToBytes(uint64(profit), 8, true), 9) // profit

	count, index := 0, 18
	for i := 0; i < 3; i++ {
		id := melting.MeltedItems[i]
		if id == 0 {
			continue
		}

		maxCount := int64(melting.Quantities[i])
		meltedCount := utils.RandInt(0, maxCount+1)
		if meltedCount == 0 {
			continue
		}
		itemData, slotID, err := c.AddItem(&InventorySlot{ItemID: int64(id), Quantity: uint(meltedCount)}, -1, false)
		if err != nil {
			return nil, false, err
		} else if itemData == nil {
			return nil, false, nil
		}
		itemSlot := slots[slotID]

		count++
		r.Insert(utils.IntToBytes(uint64(id), 4, true), index) // melted item id
		index += 4

		r.Insert([]byte{0x00, 0xA2}, index)
		index += 2

		r.Insert(utils.IntToBytes(uint64(meltedCount), 2, true), index) // melted item count
		index += 2

		r.Insert(utils.IntToBytes(uint64(slotID), 2, true), index) // free slot id
		index += 2

		r.Insert([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, index) // upgrades
		index += 34

		resp.Concat(*itemData)
		resp.Concat(itemSlot.GetData(itemSlot.SlotID))
	}

	r[17] = byte(count)
	length := int16(44*count) + 14

	if melting.SpecialItem > 0 {
		seed := uint16(utils.RandInt(0, 1000))

		if seed < melting.SpecialItemProbability {

			freeSlot, err := c.FindFreeSlot()
			if err != nil {
				return nil, false, err
			}

			r.Insert([]byte{0x01}, index)
			index++

			r.Insert(utils.IntToBytes(uint64(melting.SpecialItem), 4, true), index) // special item id
			index += 4

			r.Insert([]byte{0x00, 0xA2, 0x01, 0x00}, index)
			index += 4

			r.Insert(utils.IntToBytes(uint64(freeSlot), 2, true), index) // free slot id
			index += 2

			r.Insert([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, index) // upgrades
			index += 34

			itemData, _, err := c.AddItem(&InventorySlot{ItemID: int64(melting.SpecialItem), Quantity: 1}, freeSlot, false)
			if err != nil {
				return nil, false, err
			} else if itemData == nil {
				return nil, false, nil
			}

			resp.Concat(*itemData)
			length += 45
		}
	}

	r.SetLength(length)
	if !bundle {
		resp.Concat(r)
	} else {
		itemData, err := c.RemoveItem(item.SlotID)
		if err != nil {
			return nil, false, err
		}
		resp.Concat(itemData)

		if special != nil {
			itemData, err := c.RemoveItem(special.SlotID)
			if err != nil {
				return nil, false, err
			}
			resp.Concat(itemData)
		}
	}
	resp.Concat(c.GetGold())

	return resp, true, nil
}

func (c *Character) Extraction(item, special *InventorySlot, itemSlot int16) ([]byte, bool, error) {

	info, ok := GetItemInfo(item.ItemID)
	if !ok {
		return nil, false, nil
	}
	code := int(item.GetUpgrades()[item.Plus-1])
	cost := uint64(info.SellPrice) * uint64(HaxCodes[code].ExtractionMultiplier) / 1000

	if c.TradeID != "" {
		text := "Name: " + c.Name + "(" + c.UserID + ") tried to extract items while trading."
		utils.NewLog("logs/cheat_alert.txt", text)
		return messaging.SystemMessage(10053), false, nil //Cannot do that while trading
	}
	if c.Gold < cost {
		return nil, false, nil
	}

	if !c.SubtractGold(cost) {
		return nil, false, nil
	}
	item.Plus--
	item.SetUpgrade(int(item.Plus), 0)

	resp := utils.Packet{}
	r := EXTRACTION_SUCCESS
	r.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), 9)    // item id
	r.Insert(utils.IntToBytes(uint64(item.Quantity), 2, true), 15) // item quantity
	r.Insert(utils.IntToBytes(uint64(itemSlot), 2, true), 17)      // item slot
	r.Insert(item.GetUpgrades(), 19)                               // item upgrades
	r[34] = byte(item.SocketCount)                                 // item socket count
	r.Insert(item.GetSockets(), 35)                                // item sockets

	count := 1          //int(utils.RandInt(1, 4))
	r[53] = byte(count) // stone count

	index, length := 54, int16(51)
	for i := 0; i < count; i++ {

		freeSlot, err := c.FindFreeSlot()
		if err != nil {
			return nil, false, err
		}

		id := int64(HaxCodes[code].ExtractedItem)
		itemData, _, err := c.AddItem(&InventorySlot{ItemID: id, Quantity: 1}, freeSlot, false)
		if err != nil {
			return nil, false, err
		} else if itemData == nil {
			return nil, false, nil
		}

		resp.Concat(*itemData)

		r.Insert(utils.IntToBytes(uint64(id), 4, true), index) // extracted item id
		index += 4

		r.Insert([]byte{0x00, 0xA2, 0x01, 0x00}, index)
		index += 4

		r.Insert(utils.IntToBytes(uint64(freeSlot), 2, true), index) // free slot id
		index += 2

		r.Insert([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, index) // upgrades
		index += 34

		length += 44
	}

	r.SetLength(length)
	resp.Concat(r)
	resp.Concat(c.GetGold())

	err := item.Update()
	if err != nil {
		return nil, false, err
	}

	logger.Log(logging.ACTION_EXTRACTION, c.ID, fmt.Sprintf("Extraction success for item (%d)", item.ID), c.UserID)
	return resp, true, nil
}

func (c *Character) CreateSocket(item, special *InventorySlot, itemSlot, specialSlot int16) ([]byte, error) {

	info, ok := GetItemInfo(item.ItemID)
	if !ok {
		return nil, nil
	}
	if info.CanCreateSocket != 1 {
		return nil, nil
	}
	if c.TradeID != "" {
		text := "Name: " + c.Name + "(" + c.UserID + ") tried to create socket on items while trading."
		utils.NewLog("logs/cheat_alert.txt", text)

	}

	cost := uint64(info.SellPrice * 164)
	cost -= uint64(float64(cost) * (float64(c.Socket.Stats.HyeolgongCost) / 1000))
	if c.Gold < cost {
		return nil, nil
	}

	var specialslot *Item
	if special != nil {
		specialiteminfo, ok := GetItemInfo(special.ItemID)
		if ok && specialiteminfo != nil {
			specialslot = specialiteminfo
		}
	}

	if item.SocketCount > 0 && special != nil && specialslot.GetType() == SOCKET_INITIALIZATION { // socket init
		resp := c.DecrementItem(specialSlot, 1)
		resp.Concat(item.CreateSocket(itemSlot, 0))
		item.Update()
		return *resp, nil

	} else if item.SocketCount > 0 { // item already has socket
		resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x54, 0x16, 0x0B, 0xCF, 0x55, 0xAA}
		return resp, nil

	} else if item.SocketCount == 0 && special != nil && specialslot.GetType() == SOCKET_INITIALIZATION { // socket init with no sockets
		resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x54, 0x16, 0x0A, 0xCF, 0x55, 0xAA}
		return resp, nil
	}

	seed := utils.RandInt(0, 1000)
	socketCount := int8(1)
	if seed >= 900 {
		socketCount = 4
	} else if seed >= 700 {
		socketCount = 3
	} else if seed >= 400 {
		socketCount = 2
	}

	c.LootGold(-cost)
	resp := utils.Packet{}
	if special != nil && specialslot.GetType() == SOCKET_MILED_STONE {
		socketCount += int8(specialslot.SellPrice)
		if socketCount > 5 {
			socketCount = 5
		}

		resp.Concat(*c.DecrementItem(specialSlot, 1))
	}
	item.SocketCount = socketCount
	item.Update()
	resp.Concat(item.CreateSocket(itemSlot, socketCount))
	resp.Concat(c.GetGold())
	return resp, nil
}

func (c *Character) UpgradeSocket(item, socket, special, edit *InventorySlot, itemSlot, socketSlot, specialSlot, editSlot int16, locks []bool) ([]byte, error) {

	info, ok := GetItemInfo(item.ItemID)
	if !ok {
		return nil, nil
	}

	if info.CanCreateSocket != 1 {
		return nil, nil
	}
	cost := uint64(info.SellPrice * 164)
	cost -= uint64(float64(cost) * (float64(c.Socket.Stats.HyeolgongCost) / 1000))
	if c.Gold < cost {
		return nil, nil
	}
	if c.TradeID != "" {
		text := "Name: " + c.Name + "(" + c.UserID + ") tried to upgrade sockets on items while trading."
		utils.NewLog("logs/cheat_alert.txt", text)
		return messaging.SystemMessage(10053), nil //Cannot do that while trading
	}

	if item.SocketCount == 0 { // No socket on item
		resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x54, 0x16, 0x10, 0xCF, 0x55, 0xAA}
		return resp, nil
	}

	if socket.Plus < uint8(item.SocketCount) { // Insufficient socket
		resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x54, 0x17, 0x0D, 0xCF, 0x55, 0xAA}
		return resp, nil
	}

	stabilize := false
	if special != nil {
		specialiteminfo, ok := GetItemInfo(special.ItemID)
		if ok && specialiteminfo != nil {
			if specialiteminfo.GetType() == SOCKET_STABILIZER_TYPE {
				stabilize = true
			}
		}
	}

	if edit != nil {
		edititeminfo, ok := GetItemInfo(edit.ItemID)
		if ok && edititeminfo != nil {
			if edititeminfo.GetType() != SOCKET_REVISION_TYPE {
				return nil, nil
			}
		}
	}

	upgradesArray := bytes.Join([][]byte{ArmorUpgrades, WeaponUpgrades, AccUpgrades, HTarmorSockets}, []byte{}) //armors
	/*if info.Slot == 3 {
		upgradesArray = bytes.Join([][]byte{WeaponUpgrades, WeaponUpgrades, AccUpgrades}, []byte{}) //Weapon
	}
	if info.Slot == 5 || info.Slot == 6 || info.Slot == 7 || info.Slot == 8 { //Normal accessories
		upgradesArray = bytes.Join([][]byte{ArmorUpgrades, AccUpgrades, WeaponUpgrades}, []byte{})
	}
	if info.Slot == 312 || info.Slot == 313 || info.Slot == 314 || info.Slot == 315 {
		upgradesArray = bytes.Join([][]byte{ArmorUpgrades, AccUpgrades, WeaponUpgrades}, []byte{}) //2nd accessories
	}*/
	sockets := make([]byte, item.SocketCount)
	socks := item.GetSockets()
	for i := int8(0); i < item.SocketCount; i++ {
		if locks[i] {
			sockets[i] = socks[i]
			continue
		}

		seed := utils.RandInt(0, int64(len(upgradesArray)+1))
		code := upgradesArray[seed]
		if stabilize && code%5 > 0 {
			code++
		} else if !stabilize && code%5 == 0 {
			code -= 4
		}

		sockets[i] = code
	}
	c.LootGold(-cost)
	resp := utils.Packet{}
	resp.Concat(item.UpgradeSocket(itemSlot, sockets))
	resp.Concat(c.GetGold())
	resp.Concat(*c.DecrementItem(socketSlot, 1))

	if special != nil {
		resp.Concat(*c.DecrementItem(specialSlot, 1))
	}

	if edit != nil {
		resp.Concat(*c.DecrementItem(editSlot, 1))
	}

	return resp, nil
}

func (c *Character) CoProduction(craftID, bFinished int) ([]byte, error) {
	resp := utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x54, 0x20, 0x0a, 0x00, 0x00, 0x55, 0xAA}
	resp.Concat(utils.Packet{0xAA, 0x55, 0x28, 0x00, 0x16, 0x2d, 0x03, 0x2d, 0x03, 0xfe, 0x9a, 0x00, 0x00, 0x83, 0x1b, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x54, 0x83, 0x05, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x27, 0x00, 0x00, 0x55, 0xAA})

	if c.TradeID != "" {
		text := "Name: " + c.Name + "(" + c.UserID + ") tried to use co-production while trading."
		utils.NewLog("logs/cheat_alert.txt", text)
		return messaging.SystemMessage(10053), nil //Cannot do that while trading
	}
	discrimination := false
	if bFinished == 1 {
		if (craftID >= 14000000 && craftID <= 14600020) || (craftID >= 55000001 && craftID <= 55000432) {
			discrimination = true
		}
		production, ok := CraftItems[int(craftID)]
		if !ok {
			return nil, nil
		}
		var prodMaterials []int
		var prodQty []int
		var probabilities []int
		var craftedItems []int

		var upgrades string
		var plus uint8
		var sockets string
		var socketcount int8

		prodMaterials = append(prodMaterials, production.Material1)
		prodMaterials = append(prodMaterials, production.Material2)
		prodMaterials = append(prodMaterials, production.Material3)
		prodMaterials = append(prodMaterials, production.Material4)
		prodMaterials = append(prodMaterials, production.Material5)
		prodMaterials = append(prodMaterials, production.Material6)
		prodQty = append(prodQty, production.Material1Count)
		prodQty = append(prodQty, production.Material2Count)
		prodQty = append(prodQty, production.Material3Count)
		prodQty = append(prodQty, production.Material4Count)
		prodQty = append(prodQty, production.Material5Count)
		prodQty = append(prodQty, production.Material6Count)
		probabilities = append(probabilities, production.Probability1)
		probabilities = append(probabilities, production.Probability2)
		probabilities = append(probabilities, production.Probability3)
		craftedItems = append(craftedItems, production.Probability1Result)
		craftedItems = append(craftedItems, production.Probability2Result)
		craftedItems = append(craftedItems, production.Probability3Result)

		for i := 0; i < len(prodMaterials); i++ {
			if int64(prodMaterials[i]) == 0 {
				continue
			} else {
				slotID, slot, _ := c.FindItemInInventory(nil, int64(prodMaterials[i]))
				if discrimination && (slot.SocketCount > 0 || slot.Plus > 0) {
					upgrades = slot.UpgradeArr
					plus = slot.Plus
					sockets = slot.SocketArr
					socketcount = slot.SocketCount

				}

				if slot.ItemType != 0 && discrimination {
					return messaging.SystemMessage(4025), nil //already discriminated
				}
				matCount := uint(prodQty[i])

				itemData := c.DecrementItem(slotID, matCount)
				c.Socket.Write(*itemData)
			}
		}
		cost := uint64(production.Cost)
		if !c.SubtractGold(cost) {
			return nil, nil
		}
		slots, err := c.InventorySlots()
		if err != nil {
			return nil, err
		}
		index := 0
		seed := int(utils.RandInt(0, 1000))

		for _, prob := range probabilities {
			if float64(seed) > float64(prob) {
				index++
				continue
			}
			break
		}

		reward := NewSlot()
		reward.ItemID = int64(craftedItems[index])
		if reward.ItemID == 0 {
			return PRODUCTION_FAILED, nil
		}
		reward.Quantity = 1
		iteminfo, ok := GetItemInfo(reward.ItemID)
		if !ok {
			return nil, nil
		}
		if iteminfo.TimerType > 0 {
			reward.Quantity = uint(iteminfo.Timer)
		}
		if reward.ItemID == 204001072 {
			reward.Quantity = 100
		}
		if reward.ItemID == 204001073 {
			reward.Quantity = 250
		}
		if reward.ItemID == 204001074 {
			reward.Quantity = 500
		}
		if reward.ItemID == 204001075 {
			reward.Quantity = 1000
		}
		if discrimination {
			reward.UpgradeArr = upgrades
			reward.Plus = plus
			reward.SocketArr = sockets
			reward.SocketCount = socketcount
			rand := utils.RandInt(0, 100)
			if rand < 30 {
				reward.ItemType = 1
			} else {
				c.Socket.Write(PRODUCTION_FAILED)
			}
		}
		_, new, _ := c.AddItem(reward, -1, true)
		resp.Concat(slots[new].GetData(new))
		if reward.ItemID >= 57000000 && reward.ItemID <= 57000021 {
			c.NearbyNpcCastSkill(50007, 10067)
		}

	}
	return resp, nil
}

func (c *Character) NearbyNpcCastSkill(npcID int, skillid int) {

	ids, err := c.GetNearbyNPCIDs()
	if err != nil {
		log.Println(err)
		return
	}
	for _, id := range ids {

		npcPos := GetNPCPosByID(id)
		if npcPos == nil {
			continue
		}
		if npcPos.NPCID == npcID {

			mC := ConvertPointToLocation(npcPos.MinLocation)

			resp := MOB_SKILL
			resp.Insert(utils.IntToBytes(uint64(npcPos.PseudoID), 2, true), 7)  // mob pseudo id
			resp.Insert(utils.IntToBytes(uint64(skillid), 4, true), 9)          // pet skill id
			resp.Insert(utils.FloatToBytes(mC.X, 4, true), 13)                  // pet-x
			resp.Insert(utils.FloatToBytes(mC.Y, 4, true), 17)                  // pet-x
			resp.Insert(utils.IntToBytes(uint64(npcPos.PseudoID), 2, true), 25) // target pseudo id
			resp.Insert(utils.IntToBytes(uint64(npcPos.PseudoID), 2, true), 28) // target pseudo id

			p := nats.CastPacket{CastNear: true, CharacterID: c.ID, Data: resp, Type: nats.MOB_ATTACK}
			p.Cast()
		}
	}
}

func (c *Character) CookFood(craftID int) ([]byte, error) {
	resp := utils.Packet{}

	if c.TradeID != "" {
		text := "Name: " + c.Name + "(" + c.UserID + ") tried to cook food items while trading."
		utils.NewLog("logs/cheat_alert.txt", text)
		return messaging.SystemMessage(10053), nil //Cannot do that while trading
	}

	receipe := CookingItems[craftID]
	if receipe == nil {
		return COOKING_ERROR, nil
	}
	receipeMaterials := receipe.GetMaterials()
	receipeAmounts := receipe.GetAmounts()
	receipeProductions := receipe.GetProductions()
	receipeProbabilities := receipe.GetProbabilities()

	for i, materialID := range receipeMaterials {
		if materialID == 0 || receipeAmounts[i] == 0 {
			continue
		} else {
			slotID, _, _ := c.FindItemInInventory(nil, int64(materialID))
			matCount := uint(receipeAmounts[i])
			itemData := c.DecrementItem(slotID, matCount)
			c.Socket.Write(*itemData)
		}
	}
	cost := uint64(receipe.Cost)
	if !c.SubtractGold(cost) {
		return nil, nil
	}

	slots, _ := c.InventorySlots()

	seed := int(utils.RandInt(0, 1000))

	index := 0
	for i, prob := range receipeProbabilities {
		if float64(seed) > float64(prob) {
			index = i
			continue
		}
		break
	}

	reward := NewSlot()
	reward.ItemID = int64(receipeProductions[index])
	if reward.ItemID == 0 {
		return COOKING_FAILED, nil
	}
	reward.Quantity = 1
	iteminfo, ok := GetItemInfo(reward.ItemID)
	if !ok {
		return nil, errors.New("CookFood: Item not found")
	}
	if iteminfo.TimerType > 0 {
		reward.Quantity = uint(iteminfo.Timer)
	}
	_, slot, _ := c.AddItem(reward, -1, true)
	resp.Concat(slots[slot].GetData(slot))

	return resp, nil
}
func (c *Character) HolyWaterUpgrade(item, holyWater *InventorySlot, itemSlot, holyWaterSlot int16) ([]byte, error) {
	itemInfo, ok := GetItemInfo(item.ItemID)
	if !ok {
		log.Println("HolyWaterUpgrade: Iteminfo not found")
		return nil, nil
	}
	hwInfo, ok := GetItemInfo(holyWater.ItemID)
	if !ok {
		log.Println("HolyWaterUpgrade: Itemhwinfo not found")
		return nil, nil
	}

	if c.TradeID != "" {
		text := "Name: " + c.Name + "(" + c.UserID + ") tried to use holy water items while trading."
		utils.NewLog("logs/cheat_alert.txt", text)
		return messaging.SystemMessage(10053), nil //Cannot do that while trading
	}

	if (itemInfo.GetType() == WEAPON_TYPE && (hwInfo.HolyWaterUpg1 < 66 || hwInfo.HolyWaterUpg1 > 105)) ||
		(itemInfo.GetType() == ARMOR_TYPE && (hwInfo.HolyWaterUpg1 < 41 || hwInfo.HolyWaterUpg1 > 65)) ||
		(itemInfo.GetType() == ACC_TYPE && hwInfo.HolyWaterUpg1 > 40) || (itemInfo.GetType() == MASTER_HT_ACC && hwInfo.HolyWaterUpg1 > 40) { // Mismatch type

		resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x54, 0x10, 0x36, 0x11, 0x55, 0xAA}
		return resp, nil
	}
	if hwInfo.HolyWaterPlus > 0 {
		if item.Plus < uint8(hwInfo.HolyWaterPlus) {
			resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x54, 0x10, 0x36, 0x11, 0x55, 0xAA}
			return resp, nil
		}
	}

	if item.Plus == 0 {
		resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x54, 0x10, 0x37, 0x11, 0x55, 0xAA}
		return resp, nil
	}

	resp := utils.Packet{}
	seed, upgrade := int(utils.RandInt(0, 60)), 0
	if seed < hwInfo.HolyWaterRate1 {
		upgrade = hwInfo.HolyWaterUpg1
	} else if seed < hwInfo.HolyWaterRate2 {
		upgrade = hwInfo.HolyWaterUpg2
	} else if seed < hwInfo.HolyWaterRate3 {
		upgrade = hwInfo.HolyWaterUpg3
	} else {
		resp = HOLYWATER_FAILED
	}
	if upgrade > 0 {
		randSlot := utils.RandInt(0, int64(item.Plus))
		if hwInfo.HolyWaterPlus > 0 {
			randSlot = int64(hwInfo.HolyWaterPlus - 1)
		}
		preUpgrade := item.GetUpgrades()[randSlot]
		item.SetUpgrade(int(randSlot), byte(upgrade))

		resp = HOLYWATER_SUCCESS

		r := ITEM_UPGRADED
		r.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), 9) // item id
		r.Insert(utils.IntToBytes(uint64(itemSlot), 2, true), 17)   // slot id
		r.Insert(item.GetUpgrades(), 19)                            // item upgrades
		r[34] = byte(item.SocketCount)                              // socket count
		r.Insert(item.GetSockets(), 35)                             // item sockets
		resp.Concat(r)

		new := funk.Map(item.GetUpgrades()[:item.Plus], func(upg byte) string {
			return HaxCodes[int(upg)].Code
		}).([]string)

		old := make([]string, len(new))
		copy(old, new)
		old[randSlot] = HaxCodes[int(preUpgrade)].Code

		msg := fmt.Sprintf("[%s] has been upgraded from [%s] to [%s].", itemInfo.Name, strings.Join(old, ""), strings.Join(new, ""))
		msgData := messaging.InfoMessage(msg)
		resp.Concat(msgData)

	}

	itemData, _ := c.RemoveItem(holyWaterSlot)
	resp.Concat(itemData)

	err := item.Update()
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *Character) RegisterItem(item *InventorySlot, price uint64, itemSlot int16) ([]byte, error) {
	items, err := FindConsignmentItemsBySellerID(c.ID)
	if err != nil {
		return nil, err
	}
	if len(items) >= 10 {
		return nil, nil
	}

	commision := uint64(math.Min(float64(price/100), float64(price)))
	if c.Gold < commision {
		return nil, nil
	}
	sale := FindSale(c.PseudoID)
	if sale != nil {
		return messaging.SystemMessage(messaging.CANNOT_MOVE_ITEM_IN_TRADE), nil
	}
	if c.TradeID != "" {
		return messaging.SystemMessage(messaging.CANNOT_MOVE_ITEM_IN_TRADE), nil
	}

	info, ok := GetItemInfo(item.ItemID)
	if !ok || item.Activated || item.InUse || info.Tradable != 1 {
		return nil, nil
	}

	consItem := &ConsignmentItem{
		ID:             item.ID,
		UserID:         item.UserID,
		SellerID:       item.CharacterID,
		ItemID:         item.ItemID,
		SlotID:         item.SlotID,
		Quantity:       item.Quantity,
		Plus:           item.Plus,
		UpgradeArr:     item.UpgradeArr,
		SocketCount:    item.SocketCount,
		SocketArr:      item.SocketArr,
		Activated:      item.Activated,
		InUse:          item.InUse,
		PetInfo:        item.PetInfo,
		Consignment:    true,
		Appearance:     item.Appearance,
		ItemType:       item.ItemType,
		JudgementStat:  item.JudgementStat,
		Buff:           item.Buff,
		IsServerEpoch:  item.IsServerEpoch,
		ActivationTime: item.ActivationTime,
		Price:          price,
		IsSold:         false,
		ExpiresAt:      null.NewTime(time.Now().Add(time.Hour*time.Duration(24*7*2)), true),
		Pet:            item.Pet,
	}

	if err := consItem.Insert(); err != nil {
		return nil, err
	}
	ReadConsignmentData()

	resp := ITEM_REGISTERED
	resp.Insert(utils.IntToBytes(uint64(consItem.ID), 4, true), 9)  // consignment item id
	resp.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), 29) // item id

	if item.Pet != nil {
		resp[34] = byte(item.SocketCount)
	}

	resp.Insert(utils.IntToBytes(uint64(item.Quantity), 2, true), 35) // item count
	resp.Insert(item.GetUpgrades(), 37)                               // item upgrades

	if item.Pet != nil {
		resp[42] = 0 // item socket count
	} else {
		resp[42] = byte(item.SocketCount) // item socket count
	}

	resp.Insert(item.GetSockets(), 43) // item sockets

	del, _ := c.RemoveItem(item.SlotID)
	c.Socket.Write(del)

	claimData, err := c.ClaimMenu()
	if err != nil {
		return nil, err
	}
	resp.Concat(claimData)

	if !c.SubtractGold(commision) {
		return nil, nil
	}

	return resp, nil
}

func (c *Character) ClaimMenu() ([]byte, error) {

	ReadConsignmentData()
	items, err := FindConsignmentItemsBySellerID(c.ID)
	if err != nil {
		return nil, err
	}
	if c.TradeID != "" {
		text := "Name: " + c.Name + "(" + c.UserID + ") tried to claim consignment items while trading."
		utils.NewLog("logs/cheat_alert.txt", text)
		return messaging.SystemMessage(10053), nil //Cannot do that while trading
	}

	resp := CLAIM_MENU
	resp.SetLength(int16(len(items)*0x6B + 6))
	resp.Insert(utils.IntToBytes(uint64(len(items)), 2, true), 8) // items count

	index := 10
	for _, item := range items {

		info, ok := GetItemInfo(item.ItemID)
		if !ok {
			continue
		}

		if item.IsSold {
			resp.Insert([]byte{0x01}, index)
		} else if item.IsExpired {
			resp.Insert([]byte{0x02}, index)
		} else {
			resp.Insert([]byte{0x00}, index)
		}
		index++

		resp.Insert(utils.IntToBytes(uint64(item.ID), 4, true), index) // consignment item id
		index += 4

		resp.Insert([]byte{0x5E, 0x15, 0x01, 0x00}, index)
		index += 4

		resp.Insert([]byte(c.Name), index) // seller name
		index += len(c.Name)

		for j := len(c.Name); j < 20; j++ {
			resp.Insert([]byte{0x00}, index)
			index++
		}

		resp.Insert(utils.IntToBytes(item.Price, 8, true), index) // item price
		index += 8

		time := item.ExpiresAt.Time.Format("2006-01-02 15:04:05") // expires at
		resp.Insert([]byte(time), index)
		index += 19

		resp.Insert([]byte{0x00, 0x09, 0x00, 0x00, 0x00, 0x99, 0x31, 0xF5, 0x00}, index)
		index += 9

		resp.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), index) // item id
		index += 4

		resp.Insert([]byte{0x00, 0xA1}, index)
		index += 2

		if info.GetType() == PET_TYPE {
			resp[index-1] = byte(item.SocketCount)
		}

		resp.Insert(utils.IntToBytes(uint64(item.Quantity), 2, true), index) // item count
		index += 2

		resp.Insert(item.GetUpgrades(), index) // item upgrades
		index += 15

		resp.Insert([]byte{byte(item.SocketCount)}, index) // socket count
		index++

		resp.Insert(item.GetSockets(), index)
		index += 15

		resp.Insert([]byte{0x00, 0x00, 0x00}, index)
		index += 3
	}

	return resp, nil
}

func (c *Character) BuyConsignmentItem(consignmentID int) ([]byte, error) {

	consignmentItemInfo, ok := GetConsignmentDataById(consignmentID)
	if !ok || consignmentItemInfo == nil {
		return nil, nil
	}
	if c.TradeID != "" {
		text := "Name: " + c.Name + "(" + c.UserID + ") tried to buy consignment items while trading."
		utils.NewLog("logs/cheat_alert.txt", text)
		return messaging.SystemMessage(10053), nil //Cannot do that while trading
	}

	if c.Gold < consignmentItemInfo.Price {
		return nil, nil
	}

	seller, err := FindCharacterByID(int(consignmentItemInfo.SellerID.Int64))
	if err != nil {
		return nil, err
	}

	resp := CONSIGMENT_ITEM_BOUGHT
	resp.Insert(utils.IntToBytes(uint64(consignmentID), 4, true), 8) // consignment item id

	newItem := &InventorySlot{
		ID:             consignmentItemInfo.ID,
		UserID:         consignmentItemInfo.UserID,
		CharacterID:    consignmentItemInfo.SellerID,
		ItemID:         consignmentItemInfo.ItemID,
		Quantity:       consignmentItemInfo.Quantity,
		Plus:           consignmentItemInfo.Plus,
		UpgradeArr:     consignmentItemInfo.UpgradeArr,
		SocketCount:    consignmentItemInfo.SocketCount,
		SocketArr:      consignmentItemInfo.SocketArr,
		Activated:      false,
		InUse:          false,
		PetInfo:        consignmentItemInfo.PetInfo,
		Appearance:     consignmentItemInfo.Appearance,
		ItemType:       consignmentItemInfo.ItemType,
		JudgementStat:  consignmentItemInfo.JudgementStat,
		Buff:           consignmentItemInfo.Buff,
		IsServerEpoch:  consignmentItemInfo.IsServerEpoch,
		ActivationTime: consignmentItemInfo.ActivationTime,
		Pet:            consignmentItemInfo.Pet,
		RFU:            nil,
		Cooldown:       0,
	}

	json.Unmarshal(newItem.PetInfo, &newItem.Pet)

	freeslot, err := c.FindFreeSlot()
	if err != nil {
		return nil, nil
	}
	if freeslot == -1 {
		return nil, nil
	}
	r, _, err := c.AddItem(newItem, freeslot, false)
	if err != nil || r == nil {
		return nil, err
	}
	c.Socket.Write(*r)

	c.LootGold(-consignmentItemInfo.Price)

	resp.Concat(c.GetGold())

	s, ok := Sockets[seller.UserID]
	if ok {
		s.Write(CONSIGMENT_ITEM_SOLD)
	}

	logger.Log(logging.ACTION_BUY_CONS_ITEM, c.ID, fmt.Sprintf("Bought consignment item (%d) with %d gold from (%d)", newItem.ID, consignmentItemInfo.Price, seller.ID), c.UserID)

	text := fmt.Sprintf("Character :(%s)(%s) Bought consignment item (%d) with %d gold from seller (%s)(%s)", c.Name, c.UserID, consignmentItemInfo.ID, consignmentItemInfo.Price, seller.Name, seller.UserID)
	utils.NewLog("logs/consignment_bought_logs.txt", text)

	consignmentItemInfo.IsSold = true
	consignmentItemInfo.Update()
	ReadConsignmentData()

	return resp, nil
}

func (c *Character) ClaimConsignmentItem(consignmentID int, isCancel byte) ([]byte, error) {
	consignmentItemInfo, ok := GetConsignmentDataById(consignmentID)
	if !ok || consignmentItemInfo == nil {
		return nil, nil
	}
	if (isCancel == 1 && !consignmentItemInfo.IsSold) || (isCancel == 2 && !consignmentItemInfo.IsExpired) {
		return nil, nil
	}
	if c.TradeID != "" {
		msg := "cannot do that while trading."
		info := messaging.InfoMessage(msg)

		text := "Name: " + c.Name + "(" + c.UserID + ") tried to claim consignment items while trading."
		utils.NewLog("logs/cheat_alert.txt", text)
		return info, nil
	}
	if consignmentItemInfo.SellerID != null.IntFrom(int64(c.ID)) {
		msg := "Something went wrong. Please contact an administrator."
		info := messaging.InfoMessage(msg)
		return info, nil
	}

	resp := CONSIGMENT_ITEM_CLAIMED
	resp.Insert(utils.IntToBytes(uint64(consignmentID), 4, true), 10) // consignment item id

	if isCancel == 0 || isCancel == 2 {
		if consignmentItemInfo.IsSold {
			return nil, nil
		}

		newItem := &InventorySlot{
			ID:             consignmentItemInfo.ID,
			UserID:         consignmentItemInfo.UserID,
			CharacterID:    consignmentItemInfo.SellerID,
			ItemID:         consignmentItemInfo.ItemID,
			Quantity:       consignmentItemInfo.Quantity,
			Plus:           consignmentItemInfo.Plus,
			UpgradeArr:     consignmentItemInfo.UpgradeArr,
			SocketCount:    consignmentItemInfo.SocketCount,
			SocketArr:      consignmentItemInfo.SocketArr,
			Activated:      false,
			InUse:          false,
			PetInfo:        consignmentItemInfo.PetInfo,
			Appearance:     consignmentItemInfo.Appearance,
			ItemType:       consignmentItemInfo.ItemType,
			JudgementStat:  consignmentItemInfo.JudgementStat,
			Buff:           consignmentItemInfo.Buff,
			IsServerEpoch:  consignmentItemInfo.IsServerEpoch,
			ActivationTime: consignmentItemInfo.ActivationTime,
			Pet:            consignmentItemInfo.Pet,
			RFU:            nil,
			Cooldown:       0,
		}

		json.Unmarshal(newItem.PetInfo, &newItem.Pet)

		itemdata, slotID, err := c.AddItem(newItem, -1, false)
		if err != nil {
			return nil, err
		} else if itemdata == nil {
			return nil, nil
		} else if slotID == -1 {
			return messaging.InfoMessage("Not enough space in inventory"), nil
		}
		resp.Concat(*itemdata)

		if isCancel == 2 {
			c.LootGold(uint64(math.Min(float64(consignmentItemInfo.Price/100), float64(consignmentItemInfo.Price))))

			resp.Concat(c.GetGold())
		}

	} else {
		if !consignmentItemInfo.IsSold {
			return nil, nil
		}

		c.LootGold(consignmentItemInfo.Price)
		resp.Concat(c.GetGold())
	}

	consignmentItemInfo.Delete()
	RemoveConsignmentData(consignmentItemInfo.ID)
	ReadConsignmentData()
	return resp, nil
}

func (c *Character) UseConsumable(item *InventorySlot, slotID int16) ([]byte, error) {
	c.AntiDupeMutex.Lock()
	defer c.AntiDupeMutex.Unlock()
	if c.TradeID != "" {
		text := "Name: " + c.Name + "(" + c.UserID + ") tried to use consumable items while trading."
		utils.NewLog("logs/cheat_alert.txt", text)
		return messaging.SystemMessage(10053), nil //Cannot do that while trading
	}
	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
			log.Printf("%+v", string(dbg.Stack()))
		}
	}()

	c.ConsumableMutex.Lock()
	defer c.ConsumableMutex.Unlock()

	stat := c.Socket.Stats
	if stat.HP <= 0 {
		return *c.DecrementItem(slotID, 0), nil
	}

	info, ok := GetItemInfo(item.ItemID)
	if !ok && info == nil {
		return nil, nil
	} else if info.MinLevel > c.Level || (info.MaxLevel > 0 && info.MaxLevel < c.Level) {
		resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x59, 0x04, 0xF0, 0x03, 0x55, 0xAA} // inappropriate level
		return resp, nil
	}

	resp := utils.Packet{}
	canUse := c.CanUse(info)

	if c.Map == 251 && !(info.ID >= 203001085 && info.ID <= 203001188) {
		goto FALLBACK
	}
	if item.Cooldown > 0 {
		goto FALLBACK
	}

	switch info.GetType() {

	case AFFLICTION_TYPE:
		err := stat.Reset()
		if err != nil {
			return nil, err
		}

		statData, _ := c.GetStats()
		resp.Concat(statData)

	case TOTAL_TRANSFORMATION:
		slot, _, err := c.FindItemInInventory(nil, 15830000, 15830001, 17502883)
		if err != nil {
			log.Println(err)
			return nil, err
		} else if slot == -1 {
			return nil, nil
		}
		resp = utils.Packet{0xAA, 0x55, 0x09, 0x00, 0x01, 0xB4, 0x0A, 0x00, 0xFF, 0xFF, 0xFF, 0xFF, 0xB4, 0x55, 0xAA}

	case CHARM_OF_RETURN_TYPE:

		if c.IsinWar || c.Map == 230 {
			goto FALLBACK
		}
		d := SavePoints[int(c.Map)]
		coordinate := &utils.Location{X: d.X, Y: d.Y}
		resp.Concat(c.Teleport(coordinate))

		slots, err := c.InventorySlots()
		if err == nil {
			pet := slots[0x0A].Pet
			if pet != nil && pet.IsOnline {
				pet.IsOnline = false
				resp.Concat(DISMISS_PET)
				showpet, _ := c.ShowItems()
				resp.Concat(showpet)
				c.IsMounting = false
			}
		}

	case DEAD_SPIRIT_INCENSE_TYPE:
		slots, err := c.InventorySlots()
		if err != nil {
			return nil, err
		}

		pet := slots[0x0A].Pet
		if pet != nil && !pet.IsOnline && pet.HP <= 0 {
			pet.HP = pet.MaxHP / 10
			resp.Concat(c.GetPetStats())
			resp.Concat(c.TogglePet())
			item.SetCooldown(300)
		} else {
			goto FALLBACK
		}

	case TRANSFORMATION_PAPER_TYPE:
		if c.Level <= 200 {
			c.Class = 0
		} else {
			c.Class = 40
		}
		c.Update()
		resp = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x09, 0x00, 0x55, 0xAA}
		resp[6] = byte(c.Class)

		c.ResetPlayerSkillBook()

		skills, err := FindSkillsByID(c.ID)
		if err != nil {
			return nil, err
		}

		skillSlots, err := skills.GetSkills()
		if err != nil {
			return nil, err
		}
		for index, skill := range skillSlots.Slots {
			info, ok := SkillBooks[skill.BookID]
			if ok {
				if info.Type == 1 || info.Type == 2 {
					removeskill, err := c.RemoveSkill(utils.IntToBytes(uint64(index), 1, false)[0], skill.BookID)
					if err == nil {
						resp.Concat(removeskill)
					}
				}
			}
		}

		set := skillSlots.Slots[5]
		if set.BookID != 0 {
			removepassive, _ := c.RemovePassiveSkill(0, 5, set.BookID)
			resp.Concat(removepassive)
		}
		stats, err := c.GetStats()
		if err == nil {
			resp.Concat(stats)
		}

	case FOOD_TYPE:
		food, ok := GetItemInfo(item.ItemID)
		if !ok {
			return nil, nil
		}
		buff, err := FindBuffByID(int(food.Buff), c.ID)
		if err != nil {
			return nil, err
		} else if buff != nil {
			c.Socket.Write(EFFECT_ALREADY_EXIST)
			goto FALLBACK
		}
		infection := BuffInfections[food.Buff]
		c.AddBuff(infection, int64(food.Timer))

	case MOVEMENT_SCROLL_TYPE:
		mapID := int16(info.SellPrice)
		data, _ := c.ChangeMap(mapID, nil)
		resp.Concat(data)

	case BAG_EXPANSION_TYPE:
		buff, err := FindBuffByID(int(item.ItemID), c.ID)
		if err != nil {
			return nil, err
		} else if buff != nil {
			c.Socket.Write(EFFECT_ALREADY_EXIST)
			goto FALLBACK
		}

		buff = &Buff{ID: int(item.ItemID), CharacterID: c.ID, Name: info.Name, BagExpansion: true, StartedAt: c.Epoch, Duration: int64(info.Timer) * 60}
		err = buff.Create()
		if err != nil {
			return nil, err
		}

		remainingTime := buff.StartedAt + buff.Duration - c.Epoch
		expiration := null.NewTime(time.Now().Add(time.Second*time.Duration(remainingTime)), true) //ADD HOURS*/
		expbag := BAG_EXPANDED
		expbag.Overwrite([]byte(utils.ParseDate(expiration)), 7) // bag expiration
		resp.Concat(expbag)
		c.Socket.Write(resp)

		r, err := c.RemoveItem(slotID)
		if err != nil {
			return nil, err
		}
		c.Socket.Write(r)
		return nil, nil

	case FIRE_SPIRIT:
		_, err := c.FindFreeSlots(3)
		if err != nil {
			return messaging.SystemMessage(messaging.DISMINTLE_NOT_ENOUGH_SPACE), nil
		}
		characters, err := FindCharactersByUserID(c.UserID)
		if err == nil {
			for _, char := range characters {

				buff, err := FindBuffByID(10100, char.ID) // check for fire spirit
				if err != nil {
					return nil, err
				} else if buff != nil {
					c.Socket.Write(EFFECT_ALREADY_EXIST)
					goto FALLBACK
				}
				buff, err = FindBuffByID(10098, char.ID) // check for fire spirit
				if err != nil {
					return nil, err
				} else if buff != nil {
					c.Socket.Write(EFFECT_ALREADY_EXIST)
					goto FALLBACK
				}

			}
			for _, char := range characters {

				buff := &Buff{ID: int(10100), CharacterID: char.ID, Name: info.Name, EXPMultiplier: 300, GoldMultiplier: 5, Npc_gold_multiplier: 20, DEFRate: 5, ArtsDEFRate: 5, DropMultiplier: 5,
					ATKRate: 4, ArtsATKRate: 4, StartedAt: GetServerEpoch(), Duration: 2592000, CanExpire: true, IsServerEpoch: true, PetExpMultiplier: 300}
				err = buff.Create()
				if err != nil {
					continue
				}
			}
		}
		itemData, _, _ := c.AddItem(&InventorySlot{ItemID: 17502645, Quantity: 1}, -1, false)
		resp.Concat(*itemData)

		data, _ := c.GetStats()
		resp.Concat(data)

	case WATER_SPIRIT:
		_, err := c.FindFreeSlots(3)
		if err != nil {
			return messaging.SystemMessage(messaging.DISMINTLE_NOT_ENOUGH_SPACE), nil
		}
		characters, err := FindCharactersByUserID(c.UserID)
		if err == nil {
			for _, char := range characters {

				buff, err := FindBuffByID(10100, char.ID) // check for fire spirit
				if err != nil {
					return nil, err
				} else if buff != nil {
					c.Socket.Write(EFFECT_ALREADY_EXIST)
					goto FALLBACK
				}
				buff, err = FindBuffByID(10098, char.ID) // check for fire spirit
				if err != nil {
					return nil, err
				} else if buff != nil {
					c.Socket.Write(EFFECT_ALREADY_EXIST)
					goto FALLBACK
				}

			}
			for _, char := range characters {

				buff := &Buff{ID: int(10098), CharacterID: char.ID, Name: info.Name, EXPMultiplier: 600, GoldMultiplier: 10, Npc_gold_multiplier: 30, DEFRate: 15, ArtsDEFRate: 15,
					ATKRate: 8, ArtsATKRate: 8, StartedAt: GetServerEpoch(), Duration: 2592000, CanExpire: true, IsServerEpoch: true, PetExpMultiplier: 300}
				err = buff.Create()
				if err != nil {
					continue
				}
			}
		}

		itemData, _, _ := c.AddItem(&InventorySlot{ItemID: 17502646, Quantity: 1}, -1, false)
		resp.Concat(*itemData)

		data, _ := c.GetStats()
		resp.Concat(data)

	case FORTUNE_BOX_TYPE:
		boxID := item

		c.InvMutex.Lock()
		defer c.InvMutex.Unlock()
		_, err := c.FindFreeSlots(1)
		if err != nil {
			goto FALLBACK
		}
		gambling, ok := GamblingItems[int(item.ItemID)]
		if !ok {
			fmt.Printf("ItemID %d not found in GamblingItems", item.ItemID)
			goto FALLBACK
		}

		slots, err := c.InventorySlots()
		if err != nil {
			return nil, err
		}

		if !c.SubtractGold(gambling.Cost) {
			return nil, nil
		}

		drop, ok := GetDropInfo(gambling.DropID)
		if drop == nil || !ok {
			fmt.Printf("DropID %d not found in Drops", gambling.DropID)
			goto FALLBACK
		}

		var itemID int
		for ok {
			index := 0
			seed := int(utils.RandInt(0, 1000))
			items := drop.Items
			probabilities := drop.Probabilities

			for _, prob := range probabilities {
				if float64(seed) > float64(prob) {
					index++
					continue
				}
				break
			}

			if index >= len(items) {
				break
			}

			itemID = items[index]
			drop, ok = GetDropInfo(itemID)
		}

		plus, quantity, upgs := uint8(0), uint(1), []byte{}
		if itemID != 0 {
			rewardInfo, ok := GetItemInfo(int64(itemID))
			if ok && rewardInfo != nil {
				if rewardInfo.ID == 235 || rewardInfo.ID == 242 || rewardInfo.ID == 254 || rewardInfo.ID == 255 { // Socket-PP-Ghost Dagger-Dragon Scale
					var rates []int
					if rewardInfo.ID == 235 { // Socket
						rates = []int{300, 550, 750, 900, 980}
					} else {
						rates = []int{500, 800, 900, 930, 950, 975, 990}
					}

					seed := int(utils.RandInt(0, 1000))
					for i, rate := range rates {
						if seed >= rate {
							plus = uint8(i)
						}
					}
					plus++

					upgs = utils.CreateBytes(byte(rewardInfo.ID), int(plus), 15)

				} else if rewardInfo.GetType() == MARBLE_TYPE { // Marble
					rates := []int{200, 300, 500, 750, 950, 1000}
					seed := int(utils.RandInt(0, 1000))
					for i := 0; seed > rates[i]; i++ {
						itemID++
					}
					rewardInfo, _ = GetItemInfo(int64(itemID))

				} else if funk.Contains(haxBoxes, item.ItemID) { // Hax Box
					seed := utils.RandInt(0, 1000)
					plus = uint8(sort.SearchInts(plusRates, int(seed)) + 1)

					upgradesArray := []byte{}
					rewardType := rewardInfo.GetType()
					if rewardType == WEAPON_TYPE {
						upgradesArray = WeaponUpgrades
					} else if rewardType == ARMOR_TYPE {
						upgradesArray = ArmorUpgrades
					} else if rewardType == ACC_TYPE {
						upgradesArray = AccUpgrades
					}

					index := utils.RandInt(0, int64(len(upgradesArray)))
					code := upgradesArray[index]
					if code <= 0 {
						return nil, fmt.Errorf(fmt.Sprintf("Use consumable,upgradesArray : %d", code))
					}
					if (code-1)%5 == 3 {
						code--
					} else if (code-1)%5 == 4 {
						code -= 2
					}

					upgs = utils.CreateBytes(byte(code), int(plus), 15)
				}
				rewardCounts := GamblingItems[int(item.ItemID)].RewardCounts
				if GamblingItems[int(item.ItemID)] != nil {
					if rewardCounts != 0 {
						stackable := info.Stackable
						if !stackable {
							slots, err := c.FindFreeSlots(int(rewardCounts))
							if err != nil {
								goto FALLBACK
							}
							if len(slots) < int(rewardCounts) {
								goto FALLBACK
							} else {
								for i := 0; i < len(slots)-1; i++ {
									itemData, _, err := c.AddItem(&InventorySlot{ItemID: int64(itemID), Quantity: quantity}, -1, false)
									if err != nil {
										goto FALLBACK
									}
									c.Socket.Write(*itemData)
								}
							}
						} else {
							quantity = uint(GamblingItems[int(item.ItemID)].RewardCounts)
							if itemID == 17700160 || itemID == 17700161 || itemID == 17700162 {
								quantity = 1
							}
						}
					}
				}

				if box, ok := rewardCounts2[int(item.ItemID)]; ok {
					if q, ok := box[int(rewardInfo.ID)]; ok {
						quantity = q
					}
				}

				item := &InventorySlot{ItemID: rewardInfo.ID, Plus: uint8(plus), Quantity: quantity}
				item.SetUpgrades(upgs)

				if rewardInfo.GetType() == PET_TYPE {
					petInfo := Pets[int64(rewardInfo.ID)]
					petExpInfo := PetExps[int(petInfo.Level)]

					targetExps := []int{petExpInfo.ReqExpEvo1, petExpInfo.ReqExpEvo2, petExpInfo.ReqExpEvo3, petExpInfo.ReqExpHt, petExpInfo.ReqExpDivEvo1, petExpInfo.ReqExpDivEvo2, petExpInfo.ReqExpDivEvo3, petExpInfo.ReqExpDivEvo4, petExpInfo.ReqExpDivEvo4}
					item.Pet = &PetSlot{
						Fullness: 100, Loyalty: 100,
						Exp:   float64(targetExps[petInfo.Evolution-1]),
						HP:    petInfo.BaseHP,
						Level: byte(petInfo.Level),
						Name:  petInfo.Name,
						CHI:   petInfo.BaseChi,
					}
				}

				if rewardInfo.TimerType > 0 {
					item.Quantity = uint(rewardInfo.Timer)
				}
				if boxID.ItemID == 17502920 {
					item.ItemType = 1
				} else if boxID.ItemID == 17503253 {
					rand := utils.RandInt(0, 1000)
					if rand <= 350 {
						item.ItemType = 1
					}
				}
				_, slot, err := c.AddItem(item, -1, true)
				if err != nil {
					return nil, err
				}

				if _, ok := Relics[itemID]; ok { // relic drop
					relicDrop := c.RelicDrop(int64(itemID))
					p := nats.CastPacket{CastNear: false, Data: relicDrop, Type: nats.ITEM_DROP}
					p.Cast()
				}

				resp.Concat(slots[slot].GetData(slot))
			}
		}

	case NPC_SUMMONER_TYPE:
		if info.SellPrice == 6 {
			r := utils.Packet{0xAA, 0x55, 0x07, 0x00, 0x57, 0x03, 0x01, 0x19, 0x00, 0x00, 0x00, 0x55, 0xAA}
			resp.Concat(r)
		} else if info.SellPrice == 0 { // Bank
			resp.Concat(c.BankItems())
		}

	case PASSIVE_SKILL_BOOK_TYPE:
		if info.CharacterType > 0 && !canUse { // invalid character type
			return INVALID_CHARACTER_TYPE, nil
		}

		skills, err := FindSkillsByID(c.ID)
		if err != nil {
			return nil, err
		}

		skillSlots, err := skills.GetSkills()
		if err != nil {
			return nil, err
		}

		if JobPassives[info.ID] == nil || c.Level < JobPassives[info.ID].MinLevelRequirment || (JobPassives[info.ID].JobRequirement != c.Class && JobPassives[info.ID].JobRequirement > 1) {
			return INVALID_CHARACTER_TYPE, nil
		}

		i := -1
		if info.Name == "Air Slide Arts" || info.Name == "Wind Drift Arts" || info.ID == 16200005 {
			i = 7
			if skillSlots.Slots[i].BookID > 0 {
				return SKILL_BOOK_EXISTS, nil
			}

		} else if info.ID >= 100031005 && info.ID <= 100031072 {
			for j := 8; j <= 10; j++ {
				if skillSlots.Slots[j].BookID == 0 {
					i = j
					break
				} else if skillSlots.Slots[j].BookID == item.ItemID { // skill book exists
					return SKILL_BOOK_EXISTS, nil
				}
			}
		} else {
			for j := 5; j < 7; j++ {
				if skillSlots.Slots[j].BookID == 0 {
					i = j
					break
				} else if skillSlots.Slots[j].BookID == item.ItemID { // skill book exists
					return SKILL_BOOK_EXISTS, nil
				}
			}
		}

		if i == -1 {
			return NO_SLOTS_FOR_SKILL_BOOK, nil // FIX resp
		}

		set := &SkillSet{BookID: item.ItemID}
		set.Skills = append(set.Skills, &SkillTuple{SkillID: int(info.ID), Plus: 0})
		skillSlots.Slots[i] = set
		skills.SetSkills(skillSlots)

		skills.Update()

		skillsData, err := skills.GetSkillsData()
		if err != nil {
			return nil, err
		}

		resp.Concat(skillsData)

	case PET_POTION_TYPE:
		slots, err := c.InventorySlots()
		if err != nil {
			return nil, err
		}

		petSlot := slots[0x0A]
		pet := petSlot.Pet

		if pet == nil || !pet.IsOnline {
			goto FALLBACK
		}

		pet.HP = int(math.Min(float64(pet.HP+info.HpRecovery), float64(pet.MaxHP)))
		pet.CHI = int(math.Min(float64(pet.CHI+info.ChiRecovery), float64(pet.MaxCHI)))
		pet.Fullness = byte(math.Min(float64(pet.Fullness+5), float64(100)))
		resp.Concat(c.GetPetStats())

	case POTION_TYPE:

		if item.ItemID == 203001181 {
			if !c.CanRun {
				c.CanRun = true
			} else {
				goto FALLBACK
			}
		}
		if item.ItemID == 203001187 {
			buff, err := FindBuffByID(60030, c.ID)
			if err == nil && buff != nil {
				buff.Duration = 0
				buff.Update()
			}
		}
		statData, err := c.GetStats()
		if err != nil {
			return nil, err
		}

		resp.Concat(statData)

		hpRec := info.HpRecovery
		chiRec := info.ChiRecovery
		if hpRec == 0 && chiRec == 0 {
			hpRec = 50000
			chiRec = 50000
		}

		stat.HP = int(math.Min(float64(stat.HP+hpRec), float64(stat.MaxHP)))
		stat.CHI = int(math.Min(float64(stat.CHI+chiRec), float64(stat.MaxCHI)))
		resp.Concat(c.GetHPandChi())
		item.SetCooldown(1)

	case FILLER_POTION_TYPE:
		hpRecovery, chiRecovery := math.Min(float64(stat.MaxHP-stat.HP), 50000), float64(0)
		if hpRecovery > float64(item.Quantity) {
			hpRecovery = float64(item.Quantity)
		} else {
			chiRecovery = math.Min(float64(stat.MaxCHI-stat.CHI), 50000)
			if chiRecovery+hpRecovery > float64(item.Quantity) {
				chiRecovery = float64(item.Quantity) - hpRecovery
			}
		}

		stat.HP = int(math.Min(float64(stat.HP)+hpRecovery, float64(stat.MaxHP)))
		stat.CHI = int(math.Min(float64(stat.CHI)+chiRecovery, float64(stat.MaxCHI)))
		resp.Concat(c.GetHPandChi())
		resp.Concat(*c.DecrementItem(slotID, uint(hpRecovery+chiRecovery)))
		resp.Concat(item.GetData(slotID))
		return resp, nil

	case MAP_BOOK:
		c.Socket.Write(c.MapBookShow(uint64(slotID)))
		goto FALLBACK

	case SKILL_BOOK_TYPE:
		if !canUse { // invalid character type
			return INVALID_CHARACTER_TYPE, nil
		}

		skills, err := FindSkillsByID(c.ID)
		if err != nil {
			return nil, err
		}

		skillSlots, err := skills.GetSkills()
		if err != nil {
			return nil, err
		}

		i := -1
		for j := 0; j < 5; j++ {
			if skillSlots.Slots[j].BookID == 0 {
				i = j
				break
			} else if skillSlots.Slots[j].BookID == item.ItemID { // skill book exists
				return SKILL_BOOK_EXISTS, nil
			}
		}

		if i == -1 {
			return NO_SLOTS_FOR_SKILL_BOOK, nil // FIX resp
		}

		skillInfos := SkillBooks[item.ItemID].SkillTree
		set := &SkillSet{BookID: item.ItemID}
		c := 0
		for i := 1; i <= 24; i++ { // there should be 24 skills with empty ones

			if len(skillInfos) <= c {
				set.Skills = append(set.Skills, &SkillTuple{SkillID: 0, Plus: 0})
			} else if si := skillInfos[c]; si.Slot == i {
				tuple := &SkillTuple{SkillID: si.ID, Plus: 0}
				set.Skills = append(set.Skills, tuple)

				c++
			} else {
				set.Skills = append(set.Skills, &SkillTuple{SkillID: 0, Plus: 0})
			}
		}

		skillSlots.Slots[i] = set
		divtuple := &DivineTuple{DivineID: 0, DivinePlus: 0}
		div2tuple := &DivineTuple{DivineID: 1, DivinePlus: 0}
		div3tuple := &DivineTuple{DivineID: 2, DivinePlus: 0}
		set.DivinePoints = append(set.DivinePoints, divtuple, div2tuple, div3tuple)
		skills.SetSkills(skillSlots)

		err = skills.Update()
		if err != nil {
			return nil, err
		}

		skillsData, err := skills.GetSkillsData()
		if err != nil {
			return nil, err
		}
		resp.Concat(skillsData)
	case ESOTERIC_POTION_TYPE:
		if item == nil {
			goto FALLBACK
		}
		gambling := GamblingItems[int(item.ItemID)]
		if gambling != nil {
			d, ok := GetDropInfo(gambling.DropID)
			if ok && d != nil {

				items := d.Items
				_, err := c.FindFreeSlots(len(items))
				if err != nil {
					goto FALLBACK
				}
				slots, err := c.FindFreeSlots(3)
				if err != nil || len(slots) < 3 {
					c.Socket.Write(messaging.SystemMessage(messaging.NO_ENOUGH_SPACE)) // no enough space in inventory
					goto FALLBACK
				}

				itemData, _, err := c.AddItem(&InventorySlot{ItemID: int64(items[0]), Quantity: uint(gambling.RewardCounts)}, -1, false)
				if err != nil {
					goto FALLBACK
				}
				resp = *itemData
			}
		} else {
			c.Injury = 0 // reset injury
			c.Update()

			resp = c.GetHPandChi()
			stat, _ := c.GetStats()
			resp.Concat(stat)
		}

	case WRAPPER_BOX_TYPE:

		c.InvMutex.Lock()
		defer c.InvMutex.Unlock()
		if item == nil {
			goto FALLBACK
		} else if item.ItemID == 240000048 {
			babypet := &BabyPet{
				Npcid:      50080,
				Mapid:      c.Map,
				Server:     c.Socket.User.ConnectedServer,
				Created_at: null.NewTime(time.Now(), true),
				Coordinate: c.Coordinate,
				OwnerID:    c.ID,
				Name:       c.Name,
				Level:      1,
				HP:         100,
				Max_HP:     100,
				Hunger:     100,
			}

			err := babypet.Create()
			if err != nil {
				return nil, err
			}
			babypet.IsDead = false
			GenerateIDForBabyPet(babypet)
			babypet.OnSightPlayers = make(map[int]interface{})
			BabyPets[babypet.ID] = babypet
			BabyPetsByMap[babypet.Server][babypet.Mapid] = append(BabyPetsByMap[babypet.Server][babypet.Mapid], babypet)

			goto DELETEFALLBACK

		} else if item.ItemID >= 20050000 && item.ItemID <= 20050069 {

			var itemids []int64
			id := item.ItemID
			for {
				id++
				iteminfo, ok := GetItemInfo(id)
				if !ok {
					continue
				}
				if iteminfo.Type == 255 {
					itemids = append(itemids, id)
				} else {
					break
				}
			}
			rand := utils.RandInt(0, int64(len(itemids)))
			itemid := itemids[rand]
			rand = utils.RandInt(1, 3)

			stone := &InventorySlot{
				ItemID:   itemid,
				Quantity: uint(rand),
			}
			itemData, _, err := c.AddItem(stone, -1, false)
			if err != nil {
				goto FALLBACK
			}
			c.Socket.Write(*itemData)
			goto DELETEFALLBACK

		} else if item.ItemID >= 20050000 && item.ItemID <= 20050069 {

			var itemids []int64
			id := item.ItemID
			for {
				id++
				iteminfo, ok := GetItemInfo(id)
				if !ok {
					continue
				}
				if iteminfo.Type == 255 {
					itemids = append(itemids, id)
				} else {
					break
				}
			}
			rand := utils.RandInt(0, int64(len(itemids)))
			itemid := itemids[rand]
			rand = utils.RandInt(1, 3)

			stone := &InventorySlot{
				ItemID:   itemid,
				Quantity: uint(rand),
			}
			itemData, _, err := c.AddItem(stone, -1, false)
			if err != nil {
				goto FALLBACK
			}
			c.Socket.Write(*itemData)
			goto DELETEFALLBACK

		} else if item.ItemID == 15712003 || item.ItemID == 15712004 || item.ItemID == 15712005 {
			stone := &InventorySlot{
				ItemID:   15712002,
				SlotID:   slotID,
				Quantity: 1,
			}
			itemData, _, err := c.AddItem(stone, -1, false)
			if err != nil {
				goto FALLBACK
			}
			c.Socket.Write(*itemData)
			goto DELETEFALLBACK

		} else if item.ItemID == 17700171 {
			rand := utils.RandInt(10, 200)
			tea := &InventorySlot{
				ItemID:   20050029,
				SlotID:   slotID,
				Quantity: uint(rand),
			}
			itemData, _, err := c.AddItem(tea, -1, false)
			if err != nil {
				goto FALLBACK
			}
			c.Socket.Write(*itemData)
			goto DELETEFALLBACK

		} else if item.ItemID == 17507811 {
			rand := utils.RandInt(10, 200)
			tea := &InventorySlot{
				ItemID:   17502865,
				SlotID:   slotID,
				Quantity: uint(rand),
			}
			itemData, _, err := c.AddItem(tea, -1, false)
			if err != nil {
				goto FALLBACK
			}
			c.Socket.Write(*itemData)
			goto DELETEFALLBACK

		} else if item.ItemID == 90000304 {
			if c.Exp >= 544951059310 && c.Level == 200 {
				c.GoDarkness()
				goto DELETEFALLBACK
			}
			goto FALLBACK
		} else if item.ItemID == 17504123 {
			if c.Exp >= 233332051410 && c.Level == 100 {
				c.GoDivine()
				goto DELETEFALLBACK
			}
			goto FALLBACK
		} else if funk.Contains(AidTonics, item.ItemID) {
			c.AidTime += uint32(info.Timer)
			stData, _ := c.GetStats()
			c.Socket.Write(stData)
			goto DELETEFALLBACK

		} else if item.ItemID == 17402314 {
			socketOre := &InventorySlot{ItemID: 235, Quantity: 1, Plus: 1, UpgradeArr: "{235, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}"}
			itemData, _, err := c.AddItem(socketOre, -1, false)
			if err != nil {
				goto FALLBACK
			}
			c.Socket.Write(*itemData)
			goto DELETEFALLBACK
		} else if item.ItemID == 17402315 {
			socketOre := &InventorySlot{ItemID: 235, Quantity: 1, Plus: 2, UpgradeArr: "{235, 235, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}"}
			itemData, _, err := c.AddItem(socketOre, -1, false)
			if err != nil {
				goto FALLBACK
			}
			c.Socket.Write(*itemData)
			goto DELETEFALLBACK
		} else if item.ItemID == 17402316 {
			socketOre := &InventorySlot{ItemID: 235, Quantity: 1, Plus: 3, UpgradeArr: "{235, 235, 235, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}"}
			itemData, _, err := c.AddItem(socketOre, -1, false)
			if err != nil {
				goto FALLBACK
			}
			c.Socket.Write(*itemData)
			goto DELETEFALLBACK
		} else if item.ItemID == 17402317 {
			socketOre := &InventorySlot{ItemID: 235, Quantity: 1, Plus: 4, UpgradeArr: "{235, 235, 235, 235, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}"}
			itemData, _, err := c.AddItem(socketOre, -1, false)
			if err != nil {
				goto FALLBACK
			}
			c.Socket.Write(*itemData)
			goto DELETEFALLBACK
		} else if item.ItemID == 17402318 {
			socketOre := &InventorySlot{ItemID: 235, Quantity: 1, Plus: 5, UpgradeArr: "{235, 235, 235, 235, 235, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}"}
			itemData, _, err := c.AddItem(socketOre, -1, false)
			if err != nil {
				goto FALLBACK
			}

			c.Socket.Write(*itemData)
			goto DELETEFALLBACK
		}

		gambling, ok := GamblingItems[int(item.ItemID)]
		if gambling == nil || !ok {
			log.Printf("GamblingItems error ! itemid %d", item.ItemID)
			goto FALLBACK
		}
		d, ok := GetDropInfo(gambling.DropID)
		if d == nil || !ok {
			log.Printf("GetDropInfo error ! %d", gambling.DropID)
			goto FALLBACK
		}
		items := d.Items
		_, err := c.FindFreeSlots(len(items))
		if err != nil {
			goto FALLBACK
		}

		slots, err := c.InventorySlots()
		if err != nil {
			goto FALLBACK
		}

		i := 0
		for _, itemID := range items {
			if itemID == 0 {
				continue
			}
			info, _ := GetItemInfo(int64(itemID))
			if info == nil {
				log.Printf("GetItemInfo error %d", itemID)
				goto FALLBACK
			}
			reward := NewSlot()
			reward.ItemID = int64(itemID)
			reward.Quantity = 1

			i++

			itemType := info.GetType()

			if info.Timer > 0 && itemType != BAG_EXPANSION_TYPE {
				reward.Quantity = uint(info.Timer)
			} else if itemType == FILLER_POTION_TYPE {
				reward.Quantity = uint(info.SellPrice)
			}

			rewardCounts := GamblingItems[int(item.ItemID)].RewardCounts
			if rewardCounts != 0 {
				stackable := info.Stackable
				if !stackable {
					slots, err := c.FindFreeSlots(int(rewardCounts))
					if err != nil {
						goto FALLBACK
					}
					if len(slots) < int(rewardCounts) {
						goto FALLBACK
					} else {
						for i := 0; i < len(slots)-1; i++ {
							itemData, _, err := c.AddItem(&InventorySlot{ItemID: int64(itemID), Quantity: reward.Quantity}, -1, false)
							if err != nil {
								goto FALLBACK
							}
							c.Socket.Write(*itemData)
						}
					}
				} else {
					reward.Quantity = uint(GamblingItems[int(item.ItemID)].RewardCounts)
				}
			}

			plus, upgs := uint8(0), []byte{}
			if info.ID == 235 || info.ID == 242 || info.ID == 254 || info.ID == 255 { // Socket-PP-Ghost Dagger-Dragon Scale
				var rates []int
				if info.ID == 235 { // Socket
					rates = []int{300, 550, 750, 900, 1000}
				} else {
					rates = []int{500, 900, 950, 975, 990, 995, 998, 100}
				}

				seed := int(utils.RandInt(0, 1000))
				for ; seed > rates[plus]; plus++ {
				}
				plus++

				upgs = utils.CreateBytes(byte(info.ID), int(plus), 15)
			}

			reward.Plus = plus
			reward.SetUpgrades(upgs)

			_, slot, _ := c.AddItem(reward, -1, true)
			resp.Concat(slots[slot].GetData(slot))
		}

	case HOLY_WATER_TYPE:
		goto FALLBACK

	case FORM_TYPE:
		if !c.Morphed && c.Map == 10 {
			goto FALLBACK
		}

		info, ok := GetItemInfo(int64(item.ItemID))
		if !ok || item.Activated != c.Morphed {
			goto FALLBACK
		}

		item.Activated = !item.Activated
		item.InUse = !item.InUse
		c.Morphed = item.Activated
		c.MorphedNPCID = info.NPCID
		resp.Concat(item.GetData(slotID))
		if item.Activated {
			r := FORM_ACTIVATED
			r.Insert(utils.IntToBytes(uint64(info.NPCID), 4, true), 5) // form npc id
			resp.Concat(r)
			characters, err := c.GetNearbyCharacters()
			if err != nil {
				log.Println(err)
			}

			for _, chars := range characters {
				delete(chars.OnSight.Players, c.ID)
			}
		} else {
			c.MorphedNPCID = 0
			resp.Concat(FORM_DEACTIVATED)
			characters, err := c.GetNearbyCharacters()
			if err != nil {
				log.Println(err)
			}

			for _, chars := range characters {
				delete(chars.OnSight.Players, c.ID)
			}
		}

		err := item.Update()
		if err != nil {
			return nil, err
		}

		statData, err := c.GetStats()
		if err != nil {
			return nil, err
		}

		resp.Concat(statData)
		resp.Concat(item.GetData(item.SlotID))
		goto FALLBACK

	case CHARM_TYPE:
		slots, err := c.InventorySlots()
		if err != nil {
			return nil, err
		}

		slotID := item.SlotID

		info, ok := GetItemInfo(item.ItemID)
		if !ok || info == nil || info.Timer == 0 {
			return nil, nil
		}

		hasSameBuff := len(funk.Filter(slots, func(slot *InventorySlot) bool {
			return slot.Activated && slot.ItemID == item.ItemID
		}).([]*InventorySlot)) > 0

		if hasSameBuff && !item.Activated {
			return []byte{0xAA, 0x55, 0x04, 0x00, 0x59, 0x04, 0xFA, 0x03, 0x55, 0xAA}, nil
		}

		item.Activated = !item.Activated
		item.InUse = !item.InUse
		resp.Concat(item.GetData(slotID))

		statsData, _ := c.GetStats()
		resp.Concat(statsData)
		goto FALLBACK

	case CHARM_OF_FACTION_TYPE:
		if c.GuildID != -1 {
			c.Socket.Write(messaging.InfoMessage("First quit from your guild!"))
			goto FALLBACK
		}
		r := []byte{0xaa, 0x55, 0x05, 0x00, 0x2f, 0xff, 0x01, 0x00, 0x00, 0x55, 0xaa}
		if c.Faction == 1 {
			c.Faction = 2
			r[6] = 0x02
		} else {
			c.Faction = 1
			r[7] = 0x02
		}
		c.Update()
		characters, err := c.GetNearbyCharacters()
		if err != nil {
			log.Println(err)
		}

		for _, chars := range characters {
			delete(chars.OnSight.Players, c.ID)
		}
		chars, err := FindCharactersByUserID(c.UserID)
		if err == nil {
			for _, char := range chars {
				char.Faction = c.Faction
				char.Update()
				if c.GuildID != -1 {
					guild, err := FindGuildByID(c.GuildID)
					if err != nil {
						err = guild.RemoveMember(char.ID)
						if err != nil {
							return nil, err
						}

						err := guild.Update()
						if err != nil {
							return nil, err
						}
					}
				}
			}
			resp.Concat(r)
			break
		}
		goto FALLBACK

	case EXPANSION:

		slots, _ := c.InventorySlots()
		lastslotid := slotID

		slotID = int16(422)
		for ; slotID <= 434; slotID++ {
			slot := slots[slotID]
			if slot.ItemID == 0 {
				break
			}
		}

		toItem := slots[slotID]

		item.SlotID = slotID
		item.Update()
		*toItem = *item
		*item = *NewSlot()

		InventoryItems.Add(toItem.ID, toItem)

		c.Socket.Write(item.GetData(lastslotid))

		goto FALLBACK

	case SPECIAL_USAGE:

		if item.ItemID == 204001067 || item.ItemID == 204001068 || item.ItemID == 204001069 || item.ItemID == 204001070 || item.ItemID == 204001071 { //wooden shield
			where := item.SlotID
			to := where
			slots, err := c.InventorySlots()
			if err != nil {
				return nil, err
			}
			if slots[3].ItemID == 0 {
				to = 3
			} else if slots[4].ItemID == 0 {
				to = 4
			} else {
				goto FALLBACK
			}
			data, err := c.ReplaceItem(item.ID, where, to)
			if err != nil {
				return nil, err
			}
			resp.Concat(data)
			resp.Concat(*c.DecrementItem(to, 0))
			return resp, nil

		} else {

			f := func(item *InventorySlot) bool {
				return item.Activated
			}
			_, itm, err := c.FindItemInInventory(f, item.ItemID)
			if err != nil {
				return nil, err
			} else if itm != nil {
				goto FALLBACK
			}
			if info.Timer > 0 {
				item.Activated = !item.Activated
				item.InUse = !item.InUse
				resp.Concat(item.GetData(slotID))

				statsData, _ := c.GetStats()
				resp.Concat(statsData)
				goto FALLBACK
			} else {
				goto FALLBACK
			}
		}

	case SPECIAL_USAGE2:

		f := func(item *InventorySlot) bool {
			return item.Activated
		}
		_, itm, err := c.FindItemInInventory(f, item.ItemID)
		if err != nil {
			return nil, err
		} else if itm != nil {
			goto FALLBACK
		}
		if info.Timer > 0 {
			item.Activated = !item.Activated
			item.InUse = !item.InUse
			resp.Concat(item.GetData(slotID))

			statsData, _ := c.GetStats()
			resp.Concat(statsData)
			goto FALLBACK
		} else {
			goto FALLBACK
		}

	default:

		if info.Timer > 0 {
			item.Activated = !item.Activated
			item.InUse = !item.InUse
			resp.Concat(item.GetData(slotID))

			statsData, _ := c.GetStats()
			resp.Concat(statsData)
			goto FALLBACK
		} else {
			goto FALLBACK
		}
	}

	resp.Concat(*c.DecrementItem(slotID, 1))
	return resp, nil

FALLBACK:
	if item != nil && c != nil && slotID != 0 && slotID != -1 {

		resp.Concat(*c.DecrementItem(slotID, 0))
	}
	return resp, nil
DELETEFALLBACK:
	resp.Concat(*c.DecrementItem(slotID, 1))
	return resp, nil
}

func (c *Character) CanUse(item *Item) bool {
	if item.MaxLevel < c.Level {
		return false
	}
	if item.MinLevel > c.Level {
		return false
	}
	t := item.CharacterType
	if c.Type == 0x32 && (t == 0x32 || t == 0x01 || t == 0x03) { // MALE BEAST
		return true
	} else if c.Type == 0x33 && (t == 0x33 || t == 0x02 || t == 0x03) { // FEMALE BEAST
		return true
	} else if c.Type == 0x34 && (t == 0x34 || t == 0x01) { // Monk
		return true
	} else if c.Type == 0x35 && (t == 0x35 || t == 0x37 || t == 0x01) { //MALE_BLADE
		return true
	} else if c.Type == 0x36 && (t == 0x36 || t == 0x37 || t == 0x02) { //FEMALE_BLADE
		return true
	} else if c.Type == 0x38 && (t == 0x38 || t == 0x3A || t == 0x01) { //AXE
		return true
	} else if c.Type == 0x39 && (t == 0x39 || t == 0x3A || t == 0x02) { //FEMALE_ROD
		return true
	} else if c.Type == 0x3B && (t == 0x3B || t == 0x02) { //DUAL_BLADE
		return true
	} else if c.Type == 0x3C && (t == 0x3C || t == 0x01 || t == 0x03 || t == 0x0A) { // DIVINE MALE BEAST
		return true
	} else if c.Type == 0x3D && (t == 0x3D || t == 0x02 || t == 0x03 || t == 0x0A) { // DIVINE FEMALE BEAST
		return true
	} else if c.Type == 0x3E && (t == 0x3E || t == 0x01 || t == 0x34 || t == 0x0A) { //DIVINE MONK
		return true
	} else if c.Type == 0x3F && (t == 0x3F || t == 0x41 || t == 0x01 || t == 0x35 || t == 0x37 || t == 0x0A) { //DIVINE MALE_BLADE
		return true
	} else if c.Type == 0x40 && (t == 0x40 || t == 0x41 || t == 0x02 || t == 0x36 || t == 0x37 || t == 0x0A) { //DIVINE FEMALE_BLADE
		return true
	} else if c.Type == 0x42 && (t == 0x42 || t == 0x44 || t == 0x01 || t == 0x38 || t == 0x3A || t == 0x0A) { //DIVINE MALE_AXE
		return true
	} else if c.Type == 0x43 && (t == 0x43 || t == 0x44 || t == 0x02 || t == 0x39 || t == 0x3A || t == 0x0A) { //DIVINE FEMALE_ROD
		return true
	} else if c.Type == 0x45 && (t == 0x45 || t == 0x02 || t == 0x3B || t == 0x0A) { //DIVINE Dual Sword
		return true
	} else if c.Type == 0x46 && (t == 0x46 || t == 0x01 || t == 0x03 || t == 0x0A) { // DARK LORD MALE BEAST
		return true
	} else if c.Type == 0x47 && (t == 0x47 || t == 0x02 || t == 0x03 || t == 0x0A) { // DARK LORD FEMALE BEAST
		return true
	} else if c.Type == 0x48 && (t == 0x48 || t == 0x01 || t == 0x3E || t == 0x34 || t == 0x14) { //DARK LORD MONK
		return true
	} else if c.Type == 0x49 && (t == 0x49 || t == 0x4B || t == 0x01 || t == 0x35 || t == 0x37 || t == 0x41 || t == 0x3F || t == 0x14) { //DARK LORD MALE_BLADE
		return true
	} else if c.Type == 0x4A && (t == 0x4A || t == 0x4B || t == 0x02 || t == 0x36 || t == 0x37 || t == 0x40 || t == 0x41 || t == 0x14) { //DARK LORD FEMALE_BLADE
		return true
	} else if c.Type == 0x4C && (t == 0x4C || t == 0x4E || t == 0x01 || t == 0x38 || t == 0x3A || t == 0x42 || t == 0x44 || t == 0x14) { //DARK LORD MALE_AXE
		return true
	} else if c.Type == 0x4D && (t == 0x4D || t == 0x4E || t == 0x02 || t == 0x39 || t == 0x3A || t == 0x43 || t == 0x44 || t == 0x14) { //DARK LORD FEMALE_ROD
		return true
	} else if c.Type == 0x4F && (t == 0x4F || t == 0x02 || t == 0x45 || t == 0x3B) { //DARK LORD Dual Sword
		return true
	} else if t == 0x00 || t == 0x20 { //All character Type
		return true
	}

	return false
}

func (c *Character) UpgradeSkill(slotIndex, skillIndex byte) ([]byte, error) {
	skills, err := FindSkillsByID(c.ID)
	if err != nil {
		return nil, err
	}

	skillSlots, err := skills.GetSkills()
	if err != nil {
		return nil, err
	}

	set := skillSlots.Slots[slotIndex]
	skill := set.Skills[skillIndex]

	info := SkillInfos[skill.SkillID]
	if int8(skill.Plus) >= info.MaxPlus {
		return nil, nil
	}

	requiredSP := 1
	if info.ID >= 28000 && info.ID <= 28207 { // 2nd job passives (non-divine)
		requiredSP = SkillPTS["sjp"][skill.Plus]
	} else if info.ID >= 29000 && info.ID <= 29207 { // 2nd job passives (divine)
		requiredSP = SkillPTS["dsjp"][skill.Plus]
	} else if info.ID >= 20193 && info.ID <= 21023 { // 3nd job passives (darkness)
		requiredSP = SkillPTS["dsjp"][skill.Plus]
	}

	if skills.SkillPoints < requiredSP {
		return nil, nil
	}

	skills.SkillPoints -= requiredSP
	skill.Plus++
	resp := SKILL_UPGRADED
	resp[8] = slotIndex
	resp[9] = skillIndex
	resp.Insert(utils.IntToBytes(uint64(skill.SkillID), 4, true), 10) // skill id
	resp[14] = byte(skill.Plus)

	skills.SetSkills(skillSlots)
	skills.Update()

	if info.Passive {
		statData, err := c.GetStats()
		if err == nil {
			resp.Concat(statData)
		}
	}

	resp.Concat(c.GetExpAndSkillPts())
	return resp, nil
}

func (c *Character) DowngradeSkill(slotIndex, skillIndex byte) ([]byte, error) {

	skills, err := FindSkillsByID(c.ID)
	if err != nil {
		return nil, err
	}

	skillSlots, err := skills.GetSkills()
	if err != nil {
		return nil, err
	}

	set := skillSlots.Slots[slotIndex]
	skill := set.Skills[skillIndex]

	for i := skillIndex + 1; i < 24; i++ {
		if set.Skills[i].Plus > 0 {
			return nil, nil
		}
	}

	info := SkillInfos[skill.SkillID]
	if int8(skill.Plus) <= 0 {
		return nil, nil
	}

	skill.Plus--
	requiredSP := 1
	if info.ID >= 28000 && info.ID <= 28207 { // 2nd job passives (non-divine)
		requiredSP = SkillPTS["sjp"][skill.Plus]
	} else if info.ID >= 29000 && info.ID <= 29207 { // 2nd job passives (divine)
		requiredSP = SkillPTS["dsjp"][skill.Plus]
	} else if info.ID >= 20193 && info.ID <= 21023 { // 3nd job passives (darkness)
		requiredSP = SkillPTS["dsjp"][skill.Plus]
	}
	skills.SkillPoints += requiredSP

	resp := SKILL_DOWNGRADED
	resp[8] = slotIndex
	resp[9] = skillIndex
	resp.Insert(utils.IntToBytes(uint64(skill.SkillID), 4, true), 10) // skill id
	resp[14] = byte(skill.Plus)
	resp.Insert([]byte{0, 0, 0}, 15) //

	skills.SetSkills(skillSlots)
	skills.Update()

	if info.Passive {
		statData, err := c.GetStats()
		if err == nil {
			resp.Concat(statData)
		}
	}

	resp.Concat(c.GetExpAndSkillPts())
	return resp, nil
}
func (c *Character) DivineUpgradeSkills(skillIndex, slot int, bookID int64) ([]byte, error) {
	skills, err := FindSkillsByID(c.ID)
	if err != nil {
		return nil, err
	}

	skillSlots, err := skills.GetSkills()
	if err != nil {
		return nil, err
	}
	resp := utils.Packet{}
	//divineID := 0
	bonusPlus := 0
	usedPoints := 0
	for _, skill := range skillSlots.Slots {
		if skill.BookID == bookID {
			if len(skill.DivinePoints) == 0 {
				divtuple := &DivineTuple{DivineID: 0, DivinePlus: 0}
				div2tuple := &DivineTuple{DivineID: 1, DivinePlus: 0}
				div3tuple := &DivineTuple{DivineID: 2, DivinePlus: 0}
				skill.DivinePoints = append(skill.DivinePoints, divtuple, div2tuple, div3tuple)
				skills.SetSkills(skillSlots)
				skills.Update()
			}
			for _, point := range skill.DivinePoints {
				usedPoints += point.DivinePlus
				//if point.DivineID == slot {
				if usedPoints >= 10 {
					return nil, nil
				}
				//	divineID = point.DivineID
				if point.DivineID == slot {
					bonusPlus = point.DivinePlus
				}
			}
			skill.DivinePoints[slot].DivinePlus++
		}
	}
	bonusPlus++
	resp = DIVINE_SKILL_BOOk
	resp[8] = byte(skillIndex)
	index := 9
	resp.Insert([]byte{byte(slot)}, index) // divine id
	index++
	resp.Insert(utils.IntToBytes(uint64(bookID), 4, true), index) // book id
	index += 4
	resp.Insert([]byte{byte(bonusPlus)}, index) // divine plus
	index++
	skills.SetSkills(skillSlots)
	skills.Update()
	return resp, nil
}

func (c *Character) RemoveSkill(slotIndex byte, bookID int64) ([]byte, error) {
	log.Print(slotIndex)
	skills, err := FindSkillsByID(c.ID)
	if err != nil {
		return nil, err
	}

	skillSlots, err := skills.GetSkills()
	if err != nil {
		return nil, err
	}

	set := skillSlots.Slots[slotIndex]
	if set.BookID != bookID {
		return nil, fmt.Errorf("RemoveSkill: skill book not found")
	}

	skillSlots.Slots[slotIndex] = &SkillSet{}
	skills.SetSkills(skillSlots)
	skills.Update()

	resp := SKILL_REMOVED
	resp[8] = slotIndex
	resp.Insert(utils.IntToBytes(uint64(bookID), 4, true), 9) // book id

	return resp, nil
}

func (c *Character) UpgradePassiveSkill(slotIndex, skillIndex byte) ([]byte, error) {
	skills, err := FindSkillsByID(c.ID)
	if err != nil {
		return nil, err
	}

	skillSlots, err := skills.GetSkills()
	if err != nil {
		return nil, err
	}

	set := skillSlots.Slots[skillIndex]
	if len(set.Skills) == 0 || JobPassives[set.BookID] == nil || set.Skills[0].Plus >= JobPassives[set.BookID].MaxPlus {
		return nil, nil
	}

	if skillIndex == 5 || skillIndex == 6 { // 1st job skill
		requiredSP := SkillPTS["fjp"][set.Skills[0].Plus]
		if skills.SkillPoints < requiredSP {
			return nil, nil
		}

		skills.SkillPoints -= requiredSP

	} else if skillIndex == 7 { // running
		requiredSP := SkillPTS["wd"][set.Skills[0].Plus]
		if skills.SkillPoints < requiredSP {
			return nil, nil
		}

		skills.SkillPoints -= requiredSP
		c.RunningSpeed = 10.0 + (float64(set.Skills[0].Plus) * 0.4)
	}

	set.Skills[0].Plus++

	skills.SetSkills(skillSlots)
	skills.Update()

	resp := PASSIVE_SKILL_UGRADED
	resp[8] = slotIndex
	resp[9] = byte(set.Skills[0].Plus)

	statData, err := c.GetStats()
	if err != nil {
		return nil, err
	}

	resp.Concat(statData)
	resp.Concat(c.GetExpAndSkillPts())
	return resp, nil
}

func (c *Character) DowngradePassiveSkill(slotIndex, skillIndex byte) ([]byte, error) {
	skills, err := FindSkillsByID(c.ID)
	if err != nil {
		return nil, err
	}

	skillSlots, err := skills.GetSkills()
	if err != nil {
		return nil, err
	}

	set := skillSlots.Slots[skillIndex]

	if len(set.Skills) == 0 || set.Skills[0].Plus <= 0 {
		return nil, nil
	}

	if skillIndex == 5 && set.Skills[0].Plus > 0 { // 1st job skill
		requiredSP := SkillPTS["fjp"][set.Skills[0].Plus-1]

		skills.SkillPoints += requiredSP

	} else if skillIndex == 7 && set.Skills[0].Plus > 0 { // running
		requiredSP := SkillPTS["wd"][set.Skills[0].Plus-1]

		skills.SkillPoints += requiredSP
		c.RunningSpeed = 10.0 + (float64(set.Skills[0].Plus-1) * 0.2)
	}

	set.Skills[0].Plus--

	skills.SetSkills(skillSlots)
	skills.Update()

	resp := PASSIVE_SKILL_UGRADED
	resp[8] = slotIndex
	resp[9] = byte(set.Skills[0].Plus)

	statData, err := c.GetStats()
	if err != nil {
		return nil, err
	}

	resp.Concat(statData)
	resp.Concat(c.GetExpAndSkillPts())
	return resp, nil
}

func (c *Character) RemovePassiveSkill(slotIndex, skillIndex byte, bookID int64) ([]byte, error) {

	skills, err := FindSkillsByID(c.ID)
	if err != nil {
		return nil, err
	}

	skillSlots, err := skills.GetSkills()
	if err != nil {
		return nil, err
	}

	set := skillSlots.Slots[skillIndex]
	if set.BookID != bookID {
		return nil, fmt.Errorf("RemovePassiveSkill: skill book not found")
	}

	skillSlots.Slots[skillIndex] = &SkillSet{}
	skills.SetSkills(skillSlots)
	skills.Update()

	resp := PASSIVE_SKILL_REMOVED
	resp.Insert(utils.IntToBytes(uint64(bookID), 4, true), 8) // book id
	resp[12] = slotIndex

	return resp, nil
}

func (c *Character) CastSkill(attackCounter, skillID, targetID int, cX, cY, cZ float64) ([]byte, error) {

	slots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}

	petSlot := slots[0x0A]
	pet := petSlot.Pet
	petInfo, ok := Pets[petSlot.ItemID]
	if pet != nil && ok && pet.IsOnline && !petInfo.Combat {
		return nil, fmt.Errorf("Pet error")
	}

	resp := utils.Packet{}
	stat := c.Socket.Stats
	user := c.Socket.User
	skills := c.Socket.Skills
	character := c

	t := c.SkillHistory.Get(skillID)

	plusCooldown := 0
	plusChiCost := 0
	divinePlus := 0
	canuseskill := false
	canCast := false

	skillInfo := SkillInfos[skillID]
	if skillInfo == nil {
		return nil, nil
	}
	weapon := slots[c.WeaponSlot]

	plus, err := skills.GetPlus(skillID)
	if err != nil {
		return nil, err
	}
	skillSlots, err := c.Socket.Skills.GetSkills()
	if err != nil {
		return nil, err
	}

	ch := FindCharacterByPseudoID(c.Socket.User.ConnectedServer, uint16(c.Selection))
	if ch != nil {
		character = ch
	}
	target := GetFromRegister(user.ConnectedServer, c.Map, uint16(targetID))
	if skillInfo.PassiveType == 34 {
		teleport := c.Teleport(ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", cX, cY)))
		c.Socket.Write(teleport)
	}

	addChiCost := float64(skillInfo.AdditionalChi*int(plus)) * 2.2 / 3 // some bad words here
	chiCost := skillInfo.BaseChi + int(addChiCost) - (plusChiCost * divinePlus)
	if stat.CHI < chiCost {
		goto OUT
	}

	if weapon.ItemID != 0 {
		weaponInfo, _ := GetItemInfo(weapon.ItemID)
		canuseskill = weaponInfo.CanUse(skillInfo.Type)
	}
	if !canuseskill {
		if c.WeaponSlot == 3 {
			c.WeaponSlot = 4
		} else {
			c.WeaponSlot = 3
		}
		itemsData, err := c.ShowItems()
		if err != nil {
			return nil, err
		}

		p := nats.CastPacket{CastNear: true, CharacterID: c.ID, Type: nats.SHOW_ITEMS, Data: itemsData}
		if err = p.Cast(); err != nil {
			return nil, err
		}
		gsh, err := c.GetStats()
		if err != nil {
			return nil, err
		}
		resp1 := utils.Packet{}
		resp1.Concat(itemsData)
		resp1.Concat(gsh)
		c.Socket.Write(resp1)
		weapon := slots[c.WeaponSlot]
		if weapon.ItemID != 0 {
			weaponInfo, _ := GetItemInfo(weapon.ItemID)
			canCast = weaponInfo.CanUse(skillInfo.Type)
		}
	}

	if weapon.ItemID == 0 {
		if c.Type == MONK || c.Type == DIVINE_MONK || c.Type == DARKNESS_MONK || c.Type == BEAST_KING || c.Type == DIVINE_BEAST_KING || c.Type == DARKNESS_BEAST_KING || c.Type == EMPRESS || c.Type == DIVINE_EMPRESS || c.Type == DARKNESS_EMPRESS {
			canCast = true
		}
	} else {
		weapon := slots[c.WeaponSlot]
		weaponInfo, _ := GetItemInfo(weapon.ItemID)
		canCast = weaponInfo.CanUse(skillInfo.Type)
	}

	if skillInfo.Type == 0 {
		canCast = true
	}
	if !canCast {
		return nil, fmt.Errorf(fmt.Sprintf("%s Cant cast speels", c.Name))
	}

	for _, slot := range skillSlots.Slots {
		if slot.BookID == skillInfo.BookID {
			for _, points := range slot.DivinePoints {
				if points.DivineID == 0 && points.DivinePlus > 0 {
					divinePlus = points.DivinePlus
					plusChiCost = 50
				}
				if points.DivinePlus == 2 && points.DivinePlus > 0 {
					plusCooldown = 100
				}
			}
		}
	}

	if skillInfo.Name == "Floating" {
		c.SkillHistory.Clear()
	}
	if t != nil {
		castedAt := t.(time.Time)
		cooldown := time.Duration(skillInfo.Cooldown*100) * time.Millisecond
		cooldown -= time.Duration(plusCooldown * divinePlus) //plusCooldown * divinePlus
		if time.Since(castedAt) < cooldown {
			goto OUT
		}
	}
	c.SkillHistory.Add(skillID, time.Now())

	stat.CHI -= chiCost
	if target := skillInfo.Target; target == 0 || target == 2 { // buff skill
		character := c
		if target == 2 {
			ch := FindCharacterByPseudoID(c.Socket.User.ConnectedServer, uint16(c.Selection))
			if ch != nil {
				character = ch
			}
		}

		if skillInfo.InfectionID == 0 {
			goto COMBAT
		}
		if character != c && !c.CanAttack(character) { //target other player in non pvp
			c.DealInfection(nil, character, skillID)
		} else if character != c && c.CanAttack(character) { //target other player but is in pvp
			c.DealInfection(nil, c, skillID)
		} else if character == c {
			c.DealInfection(nil, character, skillID)
		}

		statData, _ := character.GetStats()
		character.Socket.Write(statData)

		p := &nats.CastPacket{CastNear: true, CharacterID: character.ID, Data: character.GetHPandChi()}
		p.Cast()

	} else if target := skillInfo.Target; target == 1 && skillInfo.InfectionID != 0 { //DEBUFF to player
		ch := FindCharacterByPseudoID(c.Socket.User.ConnectedServer, uint16(c.Selection))
		if ch != nil && c.CanAttack(ch) {
			c.DealInfection(nil, ch, skillID)
		}

	} else { // combat skill
		goto COMBAT
	}

COMBAT:

	if ai, ok := target.(*AI); ok { // attacked to ai

		if ai == nil || ai.HP <= 0 || ai.IsDead || ai.Faction == c.Faction {
			goto OUT
		}

		pos := GetNPCPosByID(ai.PosID)
		if pos == nil {
			goto OUT
		}
		npc, _ := GetNpcInfo(pos.NPCID)
		if npc == nil {
			goto OUT
		}

		if ai.PseudoID != uint16(c.Selection) {
			goto OUT
		}

		if skillID == 90002 || skillID == 90012 { //Move of Extortion
			st := c.Socket.Stats
			drain := ai.HP / 100 * (9 + 4*int(plus))
			st.HP += drain
			if st.HP > st.MaxHP {
				st.HP = st.MaxHP
			}
			c.Socket.Write(c.GetHPandChi())
		}

		if skillID == 41201 || skillID == 41301 || skillID == 90060 || skillID == 90070 { // howl of tame
			c.TamingAI = ai
			goto OUT

		}
		if pos.Attackable { // target is attackable
			castLocation := ConvertPointToLocation(c.Coordinate)
			if skillInfo.AreaCenter == 1 || skillInfo.AreaCenter == 2 {
				castLocation = ConvertPointToLocation(ai.Coordinate)
			}
			skillSlots, err := c.Socket.Skills.GetSkills()
			if err != nil {
				return nil, err
			}
			plusRange := 0.0
			divinePlus := 0
			plusDamage := 0
			for _, slot := range skillSlots.Slots {
				if slot.BookID == skillInfo.BookID {
					for _, points := range slot.DivinePoints {
						if points.DivineID == 2 && points.DivinePlus > 0 {
							divinePlus = points.DivinePlus
							plusRange = 0.5
						}
						if points.DivineID == 1 && points.DivinePlus > 0 {
							divinePlus = points.DivinePlus
							plusDamage = 100
						}
					}
				}
			}
			castRange := skillInfo.BaseRadius + skillInfo.AdditionalRadius*float64(plus+1) + (float64(plusRange) * float64(divinePlus)) + c.Socket.Stats.AdditionalSkillRadius
			candidates := AIsByMap[ai.Server][ai.Map]
			candidates = funk.Filter(candidates, func(cand *AI) bool {
				nPos := GetNPCPosByID(cand.PosID)
				if nPos == nil {
					return false
				}

				aiCoordinate := ConvertPointToLocation(cand.Coordinate)
				return (cand.PseudoID == ai.PseudoID || (utils.CalculateDistance(aiCoordinate, castLocation) < castRange)) && cand.HP > 0 && nPos.Attackable
			}).([]*AI)

			for _, mob := range candidates {
				dmg, _ := c.CalculateDamage(mob, true)
				dmg += plusDamage * divinePlus
				if skillInfo.InfectionID != 0 && skillInfo.Target == 1 {
					c.Targets = append(c.Targets, &Target{Damage: dmg, AI: mob, SkillId: skillID})
				}
				c.Targets = append(c.Targets, &Target{Damage: dmg, AI: mob, SkillId: skillID})
			}

			ai.PlayersMutex.RLock()
			ids := funk.Keys(ai.OnSightPlayers).([]int)
			ai.PlayersMutex.RUnlock()

			for _, id := range ids {
				enemy, err := FindCharacterByID(id)
				if err != nil || enemy == nil || !enemy.IsOnline || enemy.Map != ai.Map {
					continue
				}
				if enemy == c || !c.CanAttack(character) {
					continue
				}
				enemyCoord := ConvertPointToLocation(ai.Coordinate)
				candidateCoord := ConvertPointToLocation(enemy.Coordinate)

				plusRange := 0.0
				divinePlus := 0
				plusDamage := 0
				for _, slot := range skillSlots.Slots {
					if slot.BookID == skillInfo.BookID {
						for _, points := range slot.DivinePoints {
							if points.DivineID == 2 && points.DivinePlus > 0 {
								divinePlus = points.DivinePlus
								plusRange = 0.5
							}
							if points.DivineID == 1 && points.DivinePlus > 0 {
								divinePlus = points.DivinePlus
								plusDamage = 100
							}
						}
					}
				}

				distance := utils.CalculateDistance(enemyCoord, candidateCoord)
				castRange := skillInfo.BaseRadius + skillInfo.AdditionalRadius*float64(plus+1) + (float64(plusRange) * float64(divinePlus))
				if distance < castRange && enemy.IsActive && !skillInfo.Passive && !enemy.Invisible {
					dmg, _ := c.CalculateDamageToPlayer(enemy, true)
					dmg += plusDamage * divinePlus
					c.PlayerTargets = append(c.PlayerTargets, &PlayerTarget{Damage: dmg, Enemy: enemy, SkillId: skillID})
				}

			}

		} else { // target is not attackable
			if funk.Contains(miningSkills, skillID) { // mining skill
				c.Targets = []*Target{{Damage: 10, AI: ai}}
			}
		}
	} else if character == ch && c.CanAttack(character) { //targeting player in pvp
		if ch != nil && ch.IsActive && !skillInfo.Passive {
			dmg, _ := c.CalculateDamageToPlayer(ch, true)
			c.PlayerTargets = append(c.PlayerTargets, &PlayerTarget{Damage: dmg, Enemy: ch, SkillId: skillID})
			candidates := c.OnSight.Players
			for _, candidate := range candidates {
				enemyPseudoID := candidate.(uint16)
				enemy := FindCharacterByPseudoID(c.Socket.User.ConnectedServer, enemyPseudoID)
				if enemy == nil {
					continue
				}
				if enemy == ch || enemy == c || !c.CanAttack(character) {
					continue
				}
				enemyCoord := ConvertPointToLocation(ch.Coordinate)
				candidateCoord := ConvertPointToLocation(enemy.Coordinate)

				plusRange := 0.0
				divinePlus := 0
				plusDamage := 0
				for _, slot := range skillSlots.Slots {
					if slot.BookID == skillInfo.BookID {
						for _, points := range slot.DivinePoints {
							if points.DivineID == 2 && points.DivinePlus > 0 {
								divinePlus = points.DivinePlus
								plusRange = 0.5
							}
							if points.DivineID == 1 && points.DivinePlus > 0 {
								divinePlus = points.DivinePlus
								plusDamage = 100
							}
						}
					}
				}

				distance := utils.CalculateDistance(enemyCoord, candidateCoord)
				castRange := skillInfo.BaseRadius + skillInfo.AdditionalRadius*float64(plus+1) + (float64(plusRange) * float64(divinePlus))
				if distance < castRange && enemy.IsActive && !skillInfo.Passive && !enemy.Invisible {
					dmg, _ := c.CalculateDamageToPlayer(enemy, true)
					dmg += plusDamage * divinePlus
					c.PlayerTargets = append(c.PlayerTargets, &PlayerTarget{Damage: dmg, Enemy: enemy, SkillId: skillID})
				}

			}

			if skillID == 90002 || skillID == 90012 { //Move of Extortion
				st := c.Socket.Stats
				drain := ch.Socket.Stats.HP / 100 * (9 + 4*int(plus))
				st.HP += drain
				if st.HP > st.MaxHP {
					st.HP = st.MaxHP
				}
				c.Socket.Write(c.GetHPandChi())
			}
		}
		c.HealingSkill(skillID, plus)
	} else if character == ch && !c.CanAttack(character) { // targeting player in non-pvp
		character.HealingSkill(skillID, plus)
	} else { //targetting itself of other shit
		c.HealingSkill(skillID, plus)
	}

OUT:
	r := SKILL_CASTED
	index := 7
	r.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), index) // character pseudo id
	index += 2
	r[index] = byte(attackCounter)
	index++
	r.Insert(utils.IntToBytes(uint64(skillID), 4, true), index) // skill id
	index += 4
	r.Insert(utils.FloatToBytes(cX, 4, true), index) // coordinate-x
	index += 4
	r.Insert(utils.FloatToBytes(cY, 4, true), index) // coordinate-y
	index += 4
	r.Insert(utils.FloatToBytes(cZ, 4, true), index) // coordinate-z
	index += 5
	r.Insert(utils.IntToBytes(uint64(targetID), 2, true), index) // target id
	index += 3
	r.Insert(utils.IntToBytes(uint64(targetID), 2, true), index) // target id
	//index += 2

	p := nats.CastPacket{CastNear: true, CharacterID: c.ID, Type: nats.CAST_SKILL, Data: r}
	if err = p.Cast(); err != nil {
		return nil, err
	}

	resp.Concat(r)
	resp.Concat(c.GetHPandChi())

	return resp, nil
}

func (c *Character) DoAnimation(skillID int) {

	r := SKILL_CASTED
	index := 7
	r.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), index) // character pseudo id
	index += 2
	r[index] = byte(0)
	index++
	r.Insert(utils.IntToBytes(uint64(skillID), 4, true), index) // skill id
	index += 4
	r.Insert(utils.FloatToBytes(0, 4, true), index) // coordinate-x
	index += 4
	r.Insert(utils.FloatToBytes(0, 4, true), index) // coordinate-y
	index += 4
	r.Insert(utils.FloatToBytes(0, 4, true), index) // coordinate-z
	index += 5
	r.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), index) // target id
	index += 3
	r.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), index) // target id
	//index += 2

	p := nats.CastPacket{CastNear: true, CharacterID: c.ID, Type: nats.CAST_SKILL, Data: r}
	if err := p.Cast(); err != nil {
		return
	}
}
func (c *Character) HealingSkill(skillID int, plus byte) {
	skillInfo := SkillInfos[skillID]
	if skillInfo.PassiveType == 14 { //HEAL SKILL
		st := c.Socket.Stats
		st.HP += skillInfo.BaseMaxHP + skillInfo.AdditionalMaxHP*int(plus)
		if st.HP > st.MaxHP {
			st.HP = st.MaxHP
		}
		c.Socket.Write(c.GetHPandChi())
	}
	if skillID == 90033 || skillID == 90031 || skillID == 90027 { //REMOVE PARA AND POISON BUFF
		buffs, err := FindBuffsByCharacterID(c.ID)
		if err == nil {
			for _, bf := range buffs {
				if skillID == 90033 || skillID == 90027 {
					if bf.ID == 259 || bf.ID == 257 || bf.ID == 56 || bf.ID == 66 {
						bf.Duration = 0
						err := bf.Update()
						if err != nil {
							log.Print(err)
							return
						}
					}
				} else if skillID == 90031 || skillID == 90027 {
					if bf.ID == 257 {
						bf.Duration = 0
						err := bf.Update()
						if err != nil {
							log.Print(err)
							return
						}
					}
				}
			}
		}
	}
}

func (c *Character) DealInfection(ai *AI, character *Character, skillID int) {
	skillInfo := SkillInfos[skillID]
	if skillInfo.InfectionID == 0 {
		return
	}
	infection := BuffInfections[skillInfo.InfectionID]

	skills := c.Socket.Skills
	if skills == nil {
		return
	}
	plus, err := skills.GetPlus(skillID)
	if err != nil {
		return
	}

	duration := (skillInfo.BaseTime + skillInfo.AdditionalTime*int(plus)) / 10

	if ai != nil { //AI BUFF

	} else if character != nil { //PLAYER BUFF ADD
		if c.Level < 101 && !(character.Level < 101) {
			return
		} else if (c.Level > 100 && c.Level < 201) && !(character.Level > 100 && character.Level < 201) {
			return
		} else if c.Level > 200 && !(character.Level > 200) {
			return
		}

		if infection.ID == 66 {
			statEnemy := character.Socket.Stats
			stat := c.Socket.Stats

			seed := utils.RandInt(0, int64(stat.ParalysisATK+skillInfo.ParaDamage))
			if seed <= int64(statEnemy.ParalysisDEF) {
				buffs, err := FindBuffsByCharacterID(character.ID)

				r := DEAL_DAMAGE
				index := 5
				r.Insert(utils.IntToBytes(uint64(character.PseudoID), 2, true), index) // ai pseudo id
				index += 2
				r.Insert(utils.IntToBytes(uint64(ai.PseudoID), 2, true), index) // ai pseudo id
				index += 2
				r.Insert(utils.IntToBytes(uint64(statEnemy.HP), 4, true), index) // ai current hp
				index += 4
				r.Insert(utils.IntToBytes(uint64(statEnemy.CHI), 4, true), index) // ai current chi
				index += 4
				if err == nil {
					r.Overwrite(utils.IntToBytes(uint64(len(buffs)), 1, true), 21) //BUFF ID
					index = 22
					//r.Insert(utils.IntToBytes(uint64(18), 4, true), index) //BUFF ID
					//index += 4
					count := 0
					for _, buff := range buffs {
						if buff.ID == 10100 || buff.ID == 90098 {
							continue
						}
						r.Insert(utils.IntToBytes(uint64(buff.ID), 4, true), index) //BUFF ID
						index += 4
						if count < len(buffs)-1 {

							r.Insert(utils.IntToBytes(uint64(0), 2, true), index) //BUFF ID
							index += 2
						}
						count++
					}
					index += 4
				}
				index += 3
				r.Insert([]byte{0x00, 0x00, 0x00}, index) // INJURY
				index--
				r.SetLength(int16(index))
				r.Concat(character.GetHPandChi())
				p := &nats.CastPacket{CastNear: true, CharacterID: character.ID, Data: r, Type: nats.PLAYER_ATTACK}
				if err := p.Cast(); err != nil {
					log.Println("deal damage broadcast error:", err)
					return
				}
				return
			}
		}
		expire := true
		if c.Type == BEAST_KING || c.Type == DIVINE_BEAST_KING || c.Type == DARKNESS_BEAST_KING {
			ok := funk.Contains(Beast_King_Infections, int16(skillInfo.InfectionID))
			if ok {
				for _, buffid := range Beast_King_Infections {
					buff, err := FindBuffByID(int(buffid), character.ID)
					if buff != nil && err == nil {
						buff.CanExpire = true
						buff.Duration = 0
						err := buff.Update()
						if err != nil {
							log.Print(err)
							return
						}
						time.Sleep(time.Second * 2)
					}
				}

			}
		} else if c.Type == EMPRESS || c.Type == DIVINE_EMPRESS || c.Type == DARKNESS_EMPRESS {
			ok := funk.Contains(Empress_Infections, int16(skillInfo.InfectionID))
			if ok {
				for _, buffid := range Empress_Infections {
					buff, err := FindBuffByID(int(buffid), character.ID)
					if buff != nil && err == nil {
						buff.CanExpire = true
						buff.Duration = 0
						err := buff.Update()
						if err != nil {
							log.Print(err)
							return
						}
						time.Sleep(time.Second * 2)
					}
				}

			}
		}
		if skillInfo.InfectionID != 0 && duration == 0 {
			expire = false
		}
		buff, err := FindBuffByID(infection.ID, character.ID)
		if err != nil {
			return
		} else if buff != nil {
			buff.StartedAt = character.Epoch
			buff.Update()
		} else if buff == nil {

			if infection.IsPercent == 0 {
				buff = &Buff{ID: infection.ID, CharacterID: character.ID, StartedAt: character.Epoch, Duration: int64(duration), Name: skillInfo.Name,
					ATK: infection.BaseATK + infection.AdditionalATK*int(plus), ArtsATK: infection.BaseArtsATK + infection.AdditionalArtsATK*int(plus),
					ArtsDEF: infection.ArtsDEF + infection.AdditionalArtsDEF*int(plus), ConfusionDEF: infection.ConfusionDef + infection.AdditionalConfusionDef*int(plus),
					DEF: infection.BaseDef + infection.AdditionalDEF*int(plus), DEX: infection.DEX + infection.AdditionalDEX*int(plus), HPRecoveryRate: infection.HPRecoveryRate + infection.AdditionalHPRecovery*int(plus), INT: infection.INT + infection.AdditionalINT*int(plus),
					MaxHP: infection.MaxHP + infection.AdditionalHP*int(plus), ParalysisDEF: infection.ParalysisDef + infection.AdditionalParalysisDef*int(plus), PoisonDEF: infection.ParalysisDef + infection.AdditionalPoisonDef*int(plus), STR: infection.STR + infection.AdditionalSTR*int(plus),
					Accuracy: infection.Accuracy + infection.AdditionalAccuracy*int(plus), Dodge: infection.DodgeRate + infection.AdditionalDodgeRate*int(plus), RunningSpeed: infection.MovSpeed + infection.AdditionalMovSpeed*float64(plus), SkillPlus: int(plus), CanExpire: expire, DropMultiplier: infection.DropRate, EXPMultiplier: infection.ExpRate,
					EnhancedProbabilitiesBuff: infection.EnchancedProb, SyntheticCompositeBuff: infection.SyntheticComposite, AdvancedCompositeBuff: infection.AdvancedComposite, HyeolgongCost: infection.HyeolgongCost, PetExpMultiplier: infection.PetExpMultiplier, Wind: infection.Wind, Fire: infection.Fire, Water: infection.Water, CriticalRate: infection.CriticalRate + infection.AdditionalCritRate*int(plus),
					AttackSpeed: infection.AttackSpeed,
				}

				if skillInfo.PassiveType == 41 && skillInfo.IsIncreasing {
					buff.ATK = skillInfo.BasePassive + int(skillInfo.AdditionalPassive*float64(plus))
				}

			} else {
				percentArtsDEF := int(float64(character.Socket.Stats.ArtsDEF) * (float64(infection.ArtsDEF+infection.AdditionalArtsDEF*int(plus)) / 1000))
				percentDEF := int(float64(character.Socket.Stats.DEF) * (float64(infection.BaseDef+infection.AdditionalDEF*int(plus)) / 1000))
				percentATK := (infection.BaseATK + infection.AdditionalATK*int(plus)) / 10
				percentArtsATK := (infection.BaseArtsATK + infection.AdditionalArtsATK*int(plus)) / 10

				buff = &Buff{ID: infection.ID, CharacterID: character.ID, StartedAt: character.Epoch, Duration: int64(duration), Name: skillInfo.Name,
					ATKRate: percentATK, ArtsATKRate: percentArtsATK, ArtsDEF: percentArtsDEF, DEF: percentDEF,
					ConfusionDEF: infection.ConfusionDef + infection.AdditionalConfusionDef*int(plus),
					DEX:          infection.DEX + infection.AdditionalDEX*int(plus), HPRecoveryRate: infection.HPRecoveryRate + infection.AdditionalHPRecovery*int(plus), INT: infection.INT + infection.AdditionalINT*int(plus),
					MaxHP: infection.MaxHP + infection.AdditionalHP*int(plus), ParalysisDEF: infection.ParalysisDef + infection.AdditionalParalysisDef*int(plus), PoisonDEF: infection.ParalysisDef + infection.AdditionalPoisonDef*int(plus), STR: infection.STR + infection.AdditionalSTR*int(plus),
					Accuracy: infection.Accuracy + infection.AdditionalAccuracy*int(plus), Dodge: infection.DodgeRate + infection.AdditionalDodgeRate*int(plus), SkillPlus: int(plus), CanExpire: expire, DropMultiplier: infection.DropRate, EXPMultiplier: infection.ExpRate,
					EnhancedProbabilitiesBuff: infection.EnchancedProb, SyntheticCompositeBuff: infection.SyntheticComposite, AdvancedCompositeBuff: infection.AdvancedComposite, HyeolgongCost: infection.HyeolgongCost, PetExpMultiplier: infection.PetExpMultiplier}
			}
			buff.Create()
		}
		if funk.Contains(InvisibilitySkillIDs, buff.ID) {
			character.Invisible = true
		} else if buff.ID == 242 || buff.ID == 245 { // detection arts
			character.DetectionMode = true
		}

		statData, _ := character.GetStats()
		character.Socket.Write(statData)

		p := &nats.CastPacket{CastNear: true, CharacterID: character.ID, Data: character.GetHPandChi()}
		p.Cast()

		if buff.Name == "Floating" {
			c.Stunned = true
		}
	}
}

func (c *Character) CalculateDamage(ai *AI, isSkill bool) (int, error) {

	st := c.Socket.Stats

	npcPos := GetNPCPosByID(ai.PosID)
	if npcPos == nil {
		return 0, errors.New("npc position not found")
	}
	npc, ok := GetNpcInfo(npcPos.NPCID)
	if !ok || npc == nil {
		return 0, errors.New("npc not found")
	}

	def, min, max := npc.DEF, st.MinATK, st.MaxATK
	reborns := c.Reborns
	if c.Level > 100 {
		reborns = 0
	}
	if isSkill {
		cons := 50
		def = int(npc.Level) * cons * int(reborns)
		def += npc.SkillDEF
		min, max = st.MinArtsATK, st.MaxArtsATK
	} else {
		cons := 5
		def = int(npc.Level) * cons * int(reborns)
	}
	dmg := int(utils.RandInt(int64(min), int64(max))) - def
	if dmg < 3 {
		dmg = 3
	} else if dmg > ai.HP {
		dmg = ai.HP
	}

	server := c.Socket.User.ConnectedServer
	//-------------------------------------YINGYANG Dungeon-------------------------------------
	if c.Map == 243 {
		if YingYangMobsCounter[c.Map] == nil {
			YingYangMobsCounter[c.Map] = &DungeonMobsCounter{
				BlackBandits: 50,
				Rogues:       50,
				Ghosts:       50,
				Animals:      50,
			}
		}
		counter := YingYangMobsCounter[c.Map]
		if npcPos.NPCID == 60003 && counter.BlackBandits > 5 {
			dmg = 0
		} else if npcPos.NPCID == 60005 && counter.Rogues > 10 {
			dmg = 0
		} else if npcPos.NPCID == 60008 && counter.Ghosts > 5 {
			dmg = 0
		} else if (npcPos.NPCID == 60013 || npcPos.NPCID == 60014) && counter.Animals > 10 {
			dmg = 0
		}
	} else if c.Map == 215 {
		if YingYangMobsCounter[c.Map] == nil {
			YingYangMobsCounter[c.Map] = &DungeonMobsCounter{
				BlackBandits: 50,
				Rogues:       50,
				Ghosts:       50,
				Animals:      50,
			}
		}
		counter := YingYangMobsCounter[c.Map]
		if npcPos.NPCID == 60017 && counter.BlackBandits > 5 {
			dmg = 0
		} else if npcPos.NPCID == 60019 && counter.Rogues > 10 {
			dmg = 0
		} else if npcPos.NPCID == 60022 && counter.Ghosts > 5 {
			dmg = 0
		} else if (npcPos.NPCID == 60027 || npcPos.NPCID == 60028) && counter.Animals > 10 {
			dmg = 0
		}
	} else if c.Map == 212 {
		if npcPos.NPCID == 45009 && (SeasonCaveMobsCounter[server].Bats > 5 || SeasonCaveMobsCounter[server].Spiders > 5) {
			dmg = 0
		} else if npcPos.NPCID == 45008 && (SeasonCaveMobsCounter[server].Centipede != 0 || SeasonCaveMobsCounter[server].Snakes > 5) {
			dmg = 0
		}
	}
	if diff := int(npc.Level) - c.Level; diff > 0 {
		reqAcc := utils.SigmaFunc(float64(diff))
		if float64(st.Accuracy) < reqAcc {
			probability := float64(st.Accuracy) * 1000 / reqAcc
			if utils.RandInt(0, 1000) > int64(probability) {
				dmg = 0
			}
		}
	}

	return dmg, nil
}

func (c *Character) CalculateDamageToPlayer(enemy *Character, isSkill bool) (int, error) {
	st := c.Socket.Stats
	enemySt := enemy.Socket.Stats

	def, min, max := enemySt.PVPdef, st.MinATK, st.MaxATK
	if isSkill {
		def, min, max = enemySt.PVPsdef, st.MinArtsATK, st.MaxArtsATK
	}

	def = utils.PvPFunc(def)

	dmg := int(utils.RandInt(int64(min), int64(max))) - def
	dmg = int(float32(dmg) - float32(dmg)*enemySt.DEXDamageReduction/100)
	dmg += int(float32(dmg) * 0.3)

	if dmg < 0 {
		dmg = 3
	} else if dmg > enemySt.HP {
		dmg = enemySt.HP
	}

	reqAcc := float64(enemySt.Dodge) - float64(st.Accuracy)
	if utils.RandInt(0, 1200) < int64(reqAcc) {
		dmg = 0
	}

	return dmg, nil
}

func (c *Character) CancelTrade() {

	trade := FindTrade(c)
	if trade == nil {
		return
	}

	receiver, sender := trade.Receiver.Character, trade.Sender.Character
	trade.Delete()

	resp := TRADE_CANCELLED
	sender.Socket.Write(resp)
	receiver.Socket.Write(resp)
}

func (c *Character) OpenSale(name string, slotIDs []int16, prices []uint64) ([]byte, error) {

	slots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}
	if c.TradeID != "" {
		text := "Name: " + c.Name + "(" + c.UserID + ") tried to open sale while trading."
		utils.NewLog("logs/cheat_alert.txt", text)
		return messaging.SystemMessage(10053), nil //Cannot do that while trading
	}

	sale := &Sale{ID: c.PseudoID, Seller: c, Name: name}
	for i := 0; i < len(slotIDs); i++ {
		slotID := slotIDs[i]
		price := prices[i]
		item := slots[slotID]

		info, ok := GetItemInfo(item.ItemID)
		if !ok {
			continue
		}

		if slotID == 0 || price == 0 || item == nil || item.ItemID == 0 || info.Tradable != 1 {
			continue
		}
		if item.Activated || item.InUse {
			return nil, nil
		}

		saleItem := &SaleItem{SlotID: slotID, Price: price, IsSold: false}
		sale.Items = append(sale.Items, saleItem)
	}

	sale.Data, err = sale.SaleData()
	if err != nil {
		return nil, err
	}

	sale.Create()

	resp := OPEN_SALE
	spawnData, err := c.SpawnCharacter()
	if err == nil {
		p := nats.CastPacket{CastNear: true, CharacterID: c.ID, Type: nats.PLAYER_SPAWN, Data: spawnData}
		p.Cast()
		resp.Concat(spawnData)
	}

	text := fmt.Sprintf("Character: " + c.Name + "(" + c.UserID + ") Opened shop with items :\n")
	for _, saleItem := range sale.Items {
		slotID := saleItem.SlotID
		item := slots[slotID]
		info, _ := GetItemInfo(item.ItemID)
		text += fmt.Sprintf("%d		%s			Price: %d 			Qty: %d\n", info.ID, info.Name, saleItem.Price, item.Quantity)
	}

	utils.NewLog("logs/shops_logs.txt", text)

	return resp, nil
}

func FindSaleVisitors(saleID uint16) []*Character {

	characterMutex.RLock()
	allChars := funk.Values(characters)
	characterMutex.RUnlock()

	return funk.Filter(allChars, func(c *Character) bool {
		return c.IsOnline && c.VisitedSaleID == saleID
	}).([]*Character)
}

func (c *Character) CloseSale() ([]byte, error) {
	sale := FindSale(c.PseudoID)
	if sale != nil {
		sale.Delete()
		resp := CLOSE_SALE

		spawnData, err := c.SpawnCharacter()
		if err == nil {
			p := nats.CastPacket{CastNear: true, CharacterID: c.ID, Type: nats.PLAYER_SPAWN, Data: spawnData}
			p.Cast()
			resp.Concat(spawnData)
		}

		return resp, nil
	}

	return nil, nil
}

func (c *Character) BuySaleItem(saleID uint16, saleSlotID, inventorySlotID int16) ([]byte, error) {
	sale := FindSale(saleID)
	if sale == nil {
		return nil, nil
	}
	if c.TradeID != "" {
		text := "Name: " + c.Name + "(" + c.UserID + ") tried to buy sale items while trading."
		utils.NewLog("logs/cheat_alert.txt", text)
		return messaging.SystemMessage(10053), nil //Cannot do that while trading
	}
	mySlots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}

	seller := sale.Seller
	slots, err := seller.InventorySlots()
	if err != nil {
		return nil, err
	}

	saleItem := sale.Items[saleSlotID]
	if saleItem == nil || saleItem.IsSold {
		return nil, nil
	}

	item := slots[saleItem.SlotID]
	if item == nil || item.ItemID == 0 || c.Gold < saleItem.Price {
		return nil, nil
	}

	if !c.SubtractGold(saleItem.Price) {
		return nil, nil
	}
	seller.Socket.Write(seller.LootGold(saleItem.Price))

	resp := BOUGHT_SALE_ITEM
	resp.Insert(utils.IntToBytes(c.Gold, 8, true), 8)                   // buyer gold
	resp.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), 17)     // sale item id
	resp.Insert(utils.IntToBytes(uint64(item.Quantity), 2, true), 23)   // sale item quantity
	resp.Insert(utils.IntToBytes(uint64(inventorySlotID), 2, true), 25) // inventory slot id
	resp.Insert(item.GetUpgrades(), 27)                                 // sale item upgrades
	resp[42] = byte(item.SocketCount)                                   // item socket count
	resp.Insert(item.GetSockets(), 43)                                  // sale item sockets

	myItem := NewSlot()
	*myItem = *item
	myItem.CharacterID = null.IntFrom(int64(c.ID))
	myItem.UserID = null.StringFrom(c.UserID)
	myItem.SlotID = int16(inventorySlotID)
	mySlots[inventorySlotID] = myItem
	myItem.Update()
	InventoryItems.Add(myItem.ID, myItem)

	saleItem.IsSold = true

	sellerResp := SOLD_SALE_ITEM
	sellerResp.Insert(utils.IntToBytes(uint64(saleSlotID), 2, true), 8)  // sale slot id
	sellerResp.Insert(utils.IntToBytes(seller.Gold, 8, true), 10)        // seller gold
	sellerResp.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), 18) // buyer pseudo id

	*item = *NewSlot()
	sellerResp.Concat(item.GetData(saleItem.SlotID))

	remainingCount := len(funk.Filter(sale.Items, func(i *SaleItem) bool {
		return !i.IsSold
	}).([]*SaleItem))

	if remainingCount > 0 {
		sale.Data, _ = sale.SaleData()
		resp.Concat(sale.Data)

	} else {
		close, err := seller.CloseSale()
		if err != nil {
			return nil, err
		}
		seller.Socket.Write(close)
	}

	seller.Socket.Write(sellerResp)
	return resp, nil
}

func (c *Character) UpdatePartyStatus() {

	user := c.Socket.User
	stat := c.Socket.Stats

	party := FindParty(c)
	if party == nil {
		return
	}

	coordinate := ConvertPointToLocation(c.Coordinate)

	resp := PARTY_STATUS
	resp.Insert(utils.IntToBytes(uint64(c.ID), 4, true), 6)
	resp.Insert(utils.IntToBytes(uint64(stat.HP), 4, true), 10)
	resp.Insert(utils.IntToBytes(uint64(stat.MaxHP), 4, true), 14)
	resp.Insert(utils.FloatToBytes(float64(coordinate.X), 4, true), 19)
	resp.Insert(utils.FloatToBytes(float64(coordinate.Y), 4, true), 23)
	resp.Insert(utils.IntToBytes(uint64(stat.CHI), 4, true), 27)
	resp.Insert(utils.IntToBytes(uint64(stat.MaxCHI), 4, true), 31)
	resp.Insert(utils.IntToBytes(uint64(c.Level), 4, true), 35)
	resp[39] = byte(c.Type)
	resp[41] = byte(user.ConnectedServer - 1)

	members := party.GetMembers()
	members = funk.Filter(members, func(m *PartyMember) bool {
		return m.Accepted
	}).([]*PartyMember)

	party.Leader.Socket.Write(resp)
	for _, m := range members {
		m.Socket.Write(resp)
	}
}

func (c *Character) LeaveParty() {

	party := FindParty(c)
	if party == nil {
		return
	}

	c.PartyID = ""

	members := party.GetMembers()
	members = funk.Filter(members, func(m *PartyMember) bool {
		return m.Accepted
	}).([]*PartyMember)

	resp := utils.Packet{}
	if c.ID == party.Leader.ID { // disband party
		resp = PARTY_DISBANDED
		party.Leader.Socket.Write(resp)

		for _, member := range members {
			member.PartyID = ""
			member.Socket.Write(resp)
		}

		party.Delete()

	} else { // leave party
		member := party.GetMember(c.ID)
		party.RemoveMember(member)

		resp = LEFT_PARTY
		resp.Insert(utils.IntToBytes(uint64(c.ID), 4, true), 8)

		leader := party.Leader
		if len(party.GetMembers()) == 0 {
			leader.PartyID = ""
			resp.Concat(PARTY_DISBANDED)
			party.Delete()

		}

		leader.Socket.Write(resp)
		for _, m := range members {
			m.Socket.Write(resp)
		}

	}
}

func (c *Character) GetGuildData() ([]byte, error) {

	if c.GuildID > 0 {
		guild, err := FindGuildByID(c.GuildID)
		if err != nil {
			return nil, err
		} else if guild == nil {
			return nil, nil
		}

		return guild.GetData(c)
	}

	return nil, nil
}
func (c *Character) JobPassives(stat *Stat) error {

	//stat := c.Socket.Stats
	skills, err := FindSkillsByID(c.ID)
	if err != nil {
		return err
	}
	if skills == nil {
		skills := &Skills{ID: c.ID}
		err = skills.Create(c)
		if err != nil {
			return err
		}
	}

	skillSlots, err := skills.GetSkills()
	if err != nil {
		return err
	}

	if passive := skillSlots.Slots[5]; passive.BookID > 0 {
		info := JobPassives[int64(passive.BookID)]
		if info != nil {
			plus := passive.Skills[0].Plus
			stat.MaxHP += info.BaseHp + info.AdditionalHp*plus
			stat.MaxCHI += info.BaseChi + info.AdditionalChi*plus
			stat.MinATK += info.BaseMinDmg + info.AdditionalMinDmg*plus
			stat.MaxATK += info.BaseMaxDmg + info.AdditionalMaxDmg*plus
			stat.MinArtsATK += info.BaseArtsATK + info.AdditionalArtsATK*plus
			stat.MinArtsATK += info.BaseArtsATK + info.AdditionalArtsATK*plus
			stat.DEF += info.BaseDEF + info.AdditionalDEF*plus
			stat.ArtsDEF += info.BaseArtsDef + info.AdditionalArtsDef*plus
			stat.Accuracy += info.BaseAccuracy + info.AdditionalAccuracy*plus
			stat.Dodge += info.BaseDodge + info.AdditionalDodge*plus
			stat.ConfusionDEF += info.BaseConfusionDEF + info.AdditionalConfusionDEF*plus
			stat.PoisonDEF += info.BasePoisonDEF + info.AdditionalPoisonDEF*plus
			stat.ParalysisDEF += info.BaseParalysisDEF + info.AddtitionalParalysisDEF*plus
			stat.HPRecoveryRate += info.BaseHPRecoveryRate + info.AdditionalHPRecoveryRate*plus
			stat.CHIRecoveryRate += info.BaseChiRecoveryRate + info.AdditionalChiRecoveryRate*plus
			stat.AdditionalRunningSpeed += info.RunningSpeed + info.AdditionalRunningSpeed*float64(plus)

		}
	}

	slots := funk.Filter(skillSlots.Slots, func(slot *SkillSet) bool { // get 2nd job passive book

		return slot.BookID == 16100200 || slot.BookID == 16100300 || slot.BookID == 100030021 || slot.BookID == 100030023 || slot.BookID == 100030025 || slot.BookID == 30000040 || slot.BookID == 30000041 || slot.BookID == 30000042 || slot.BookID == 30000043 || slot.BookID == 30000044 || slot.BookID == 30000045
	}).([]*SkillSet)

	for _, slot := range slots {
		for _, skill := range slot.Skills {
			info := SkillInfos[skill.SkillID]
			if info == nil {
				continue
			}
			if skill.Plus == 0 {
				continue
			}
			amount := info.BasePassive + int(info.AdditionalPassive*float64(skill.Plus))
			switch info.PassiveType {
			case 1: // passive hp
				stat.MaxHP += amount
			case 2: // passive chi
				stat.MaxCHI += amount
			case 3: // passive arts defense
				stat.ArtsDEF += amount
			case 4: // passive defense
				stat.DEF += amount
			case 5: // passive accuracy
				stat.Accuracy += amount
			case 6: // passive dodge
				stat.Dodge += amount
			case 7: // passive arts atk
				stat.MinArtsATK += amount
				stat.MaxArtsATK += amount
			case 8: // passive atk
				stat.MinATK += amount
				stat.MaxATK += amount
			case 9: //HP AND CHI
				stat.MaxHP += amount
				stat.MaxCHI += amount
			case 10:
				stat.ArtsDEF += amount
				stat.DEF += amount
			case 11: //Dodge RAte AND ACCURACY
				stat.Accuracy += amount
				stat.Dodge += amount
			case 12: //EXTERNAL ATK AND INTERNAL ATK
				stat.MinArtsATK += amount
				stat.MaxArtsATK += amount
				stat.MinATK += amount
				stat.MaxATK += amount
			case 13: //INTERNAL ATTACK AND INTERNAL DEF
				stat.MinATK += amount
				stat.MaxATK += amount
				stat.DEF += amount
			case 14: //EXTERNAL ATK MINUS AND HP +
				stat.MaxHP += amount
				stat.MinArtsATK -= amount
				stat.MaxArtsATK -= amount
			case 15: //DAMAGE + HP
				stat.MaxHP += amount
				stat.MinATK += amount
				stat.MaxATK += amount
			case 16: //MINUS HP AND PLUS DEFENSE
				stat.MaxHP -= 15 //
				stat.DEF += amount
			case 17: //HP
				stat.MaxHP += amount
			case 18: // passive defense
				stat.DEF += amount
			}
		}
	}

	return nil
}

func (c *Character) BuffEffects(stat *Stat) error {

	buffs, err := FindBuffsByCharacterID(c.ID)
	if err != nil {
		return err
	}

	//stat := c.Socket.Stats

	for _, buff := range buffs {
		if buff.Duration == 0 && buff.CanExpire {
			buff.Delete()
			continue
		}
		if (!buff.IsServerEpoch && buff.StartedAt+buff.Duration > c.Epoch) || (buff.IsServerEpoch && buff.StartedAt+buff.Duration > GetServerEpoch()) || !buff.CanExpire {

			if buff.ID == 257 { //Poison
				continue
			}

			stat.MinATK += buff.ATK
			stat.MaxATK += buff.ATK
			stat.ATKRate += buff.ATKRate
			stat.Accuracy += buff.Accuracy
			stat.MinArtsATK += buff.ArtsATK
			stat.MaxArtsATK += buff.ArtsATK
			stat.ArtsATKRate += buff.ArtsATKRate
			stat.ArtsDEF += buff.ArtsDEF
			stat.ArtsDEFRate += buff.ArtsDEFRate
			stat.HPRecoveryRate += buff.HPRecoveryRate
			stat.CHIRecoveryRate += buff.CHIRecoveryRate
			stat.ConfusionDEF += buff.ConfusionDEF
			stat.DEF += buff.DEF
			stat.DefRate += buff.DEFRate
			stat.DEXBuff += buff.DEX
			stat.Dodge += buff.Dodge
			stat.INTBuff += buff.INT
			stat.MaxCHI += buff.MaxCHI
			stat.MaxHP += buff.MaxHP
			stat.ParalysisDEF += buff.ParalysisDEF
			stat.PoisonDEF += buff.PoisonDEF
			stat.STRBuff += buff.STR

			stat.MinArtsATK += buff.MinArtsAtk
			stat.MaxArtsATK += buff.MaxArtsAtk

			stat.GoldMultiplier += buff.GoldMultiplier / 100
			stat.Npc_gold_multiplier += buff.Npc_gold_multiplier / 100
			stat.ExpMultiplier += float64(buff.EXPMultiplier) / 1000
			stat.DropMultiplier += float64(buff.DropMultiplier) / 1000

			stat.EnhancedProbabilitiesBuff += buff.EnhancedProbabilitiesBuff
			stat.SyntheticCompositeBuff += buff.SyntheticCompositeBuff
			stat.AdvancedCompositeBuff += buff.AdvancedCompositeBuff
			stat.HyeolgongCost += buff.HyeolgongCost

			stat.PetExpMultiplier += buff.PetExpMultiplier / 1000
		}
	}

	return nil
}

func (c *Character) GetLevelText() string {
	if c.Reborns == 0 {
		if c.Level < 10 {
			return fmt.Sprintf("%dKyu", c.Level)
		} else if c.Level <= 100 {
			return fmt.Sprintf("%dDan %dKyu", c.Level/10, c.Level%10)
		} else if c.Level < 110 {
			return fmt.Sprintf("Divine %dKyu", c.Level%100)
		} else if c.Level <= 200 {
			return fmt.Sprintf("Divine %dDan %dKyu", (c.Level-100)/10, c.Level%100)
		}
	} else {
		if c.Level < 10 {
			return fmt.Sprintf("%dKyu Reb%d", c.Level, c.Reborns)
		} else if c.Level <= 100 {
			return fmt.Sprintf("%dDan %dKyu Reb%d", c.Level/10, c.Level%10, c.Reborns)
		} else if c.Level < 110 {
			return fmt.Sprintf("Divine %dKyu Reb%d", c.Level%100, c.Reborns)
		} else if c.Level <= 200 {
			return fmt.Sprintf("Divine %dDan %dKyu Reb%d", (c.Level-100)/10, c.Level%100, c.Reborns)
		}
	}

	return ""
}

func (c *Character) RelicDrop(itemID int64) []byte {
	iteminfo, ok := GetItemInfo(itemID)
	if !ok || iteminfo == nil {
		return nil
	}

	msg := fmt.Sprintf("%s has acquired [%s].", c.Name, iteminfo.Name)
	length := int16(len(msg) + 3)

	resp := RELIC_DROP
	resp.SetLength(length)
	resp[6] = byte(len(msg))
	resp.Insert([]byte(msg), 7)

	return resp
}

func (c *Character) AidStatus() []byte {

	resp := utils.Packet{}
	if c.AidMode {
		resp = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0xFA, 0x01, 0x55, 0xAA}
		resp.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), 5)
		r2 := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x43, 0x01, 0x55, 0xAA}
		r2.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), 5)

		resp.Concat(r2)

	} else {
		resp = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0xFA, 0x00, 0x55, 0xAA}
		resp.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), 5)
		r2 := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x43, 0x00, 0x55, 0xAA}
		r2.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), 5)
		resp.Concat(r2)
	}

	return resp
}

func (c *Character) PickaxeActivated() bool {

	slots, err := c.InventorySlots()
	if err != nil {
		return false
	}

	pickaxeIDs := []int64{17200219, 17300005, 17501009, 17502536, 17502537, 17502538}

	return len(funk.Filter(slots, func(slot *InventorySlot) bool {
		return slot.Activated && funk.Contains(pickaxeIDs, slot.ItemID)
	}).([]*InventorySlot)) > 0
}

func (c *Character) TogglePet() []byte {
	slots, err := c.InventorySlots()
	if err != nil {
		return nil
	}
	if c.Map == 10 || c.Map == 233 || c.Map == 240 || c.Map == 239 || c.Map == 238 || c.Map == 237 || c.Map == 236 || c.Map == 235 {
		return nil
	}

	petSlot := slots[0x0A]
	pet := petSlot.Pet
	if pet == nil {
		return nil
	}
	spawnData, _ := c.SpawnCharacter()
	pet.PetOwner = c
	petInfo := Pets[petSlot.ItemID]
	if petInfo.Combat || !petInfo.Combat {
		location := ConvertPointToLocation(c.Coordinate)
		pet.Coordinate = utils.Location{X: location.X + 3, Y: location.Y}
		pet.IsOnline = !pet.IsOnline

		if pet.IsOnline {
			GeneratePetID(c, pet)
			pet.PetCombatMode = 0
			pet.CombatPet = true
			c.PetHandlerCB = c.PetHandler
			go c.PetHandlerCB()

			resp := utils.Packet{
				0xAA, 0x55, 0x0B, 0x00, 0x75, 0x00, 0x01, 0x00, 0x80, 0xa1, 0x43, 0x00, 0x00, 0x3d, 0x43, 0x55, 0xAA,
				0xAA, 0x55, 0x05, 0x00, 0x51, 0x01, 0x0A, 0x00, 0x00, 0x55, 0xAA,
				0xAA, 0x55, 0x06, 0x00, 0x51, 0x05, 0x0A, 0x00, 0x00, 0x00, 0x55, 0xAA,
				0xAA, 0x55, 0x05, 0x00, 0x51, 0x06, 0x0A, 0x00, 0x00, 0x55, 0xAA,
				0xAA, 0x55, 0x05, 0x00, 0x51, 0x07, 0x0A, 0x00, 0x00, 0x55, 0xAA,
				0xaa, 0x55, 0x12, 0x00, 0x51, 0x08, 0x0a, 0x00, 0x03, 0x01, 0x3b, 0x00, 0x3b, 0x00, 0x26, 0x00, 0x00, 0x00, 0x8d, 0x00, 0x00, 0x00, 0x55, 0xaa,
			}

			resp.Concat(spawnData)
			return resp
		}
	} else {
		return nil
	}
	pet.Target = 0
	pet.Casting = false
	pet.IsMoving = false
	c.PetHandlerCB = nil
	c.IsMounting = false
	RemovePetFromRegister(c)
	resp := DISMISS_PET
	return resp
}
func (c *Character) DismissPet() error {
	slots, err := c.InventorySlots()
	if err != nil {
		return err
	}
	petSlot := slots[0x0A]
	pet := petSlot.Pet
	if pet != nil || pet.PseudoID == 0 {

		pet.Target = 0
		pet.Casting = false
		pet.IsMoving = false
		c.PetHandlerCB = nil
		c.IsMounting = false
		RemovePetFromRegister(c)
		resp := DISMISS_PET
		p := nats.CastPacket{CastNear: true, CharacterID: c.ID, Data: resp}
		if err := p.Cast(); err == nil {
			c.Socket.Write(resp)
		}
	}
	return nil
}

func RemoveIndex(s []*AI, index int) []*AI {
	return append(s[:index], s[index+1:]...)
}

func (c *Character) DealDamageToPlayer(char *Character, dmg int) {
	r := DEAL_DAMAGE
	if c == nil {
		log.Println("character is nil")
		return
	} else if char.Socket.Stats.HP <= 0 {
		return
	}

	if dmg > char.Socket.Stats.HP {
		dmg = char.Socket.Stats.HP
	}
	stat := char.Socket.Stats

	//reflected := false
	if stat.DamageReflectedRate > 0 && stat.DamageReflectedProbabilty > 0 {
		seed := utils.RandInt(0, 1000)
		if seed <= int64(stat.DamageReflectedProbabilty) {
			dmg -= int(float32(dmg) * float32(stat.DamageReflectedRate) / 1000)
			//reflected = true
		}
	}

	stat.HP -= dmg
	if stat.HP <= 0 {
		stat.HP = 0
	}

	buffs, err := FindBuffsByCharacterID(char.ID)

	index := 5
	r.Insert(utils.IntToBytes(uint64(char.PseudoID), 2, true), index) // ai pseudo id
	index += 2
	r.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), index) // ai pseudo id
	index += 2
	r.Insert(utils.IntToBytes(uint64(stat.HP), 4, true), index) // ai current hp
	index += 4
	r.Insert(utils.IntToBytes(uint64(stat.CHI), 4, true), index) // ai current chi
	index += 4
	if err == nil {
		r.Overwrite(utils.IntToBytes(uint64(len(buffs)), 1, true), 21) //BUFF ID
		index = 22
		r.Insert(utils.IntToBytes(uint64(18), 4, true), index) //BUFF ID
		index += 4
		count := 0
		for _, buff := range buffs {
			if buff.ID == 10100 || buff.ID == 90098 {
				continue
			}
			r.Insert(utils.IntToBytes(uint64(buff.ID), 4, true), index) //BUFF ID
			index += 4
			if count < len(buffs)-1 {
				r.Insert(utils.IntToBytes(uint64(0), 2, true), index) //BUFF ID
				index += 2
			}
			count++
		}
		index += 4
	}
	index += 3
	r.Insert([]byte{0x00, 0x00, 0x00}, index) // INJURY
	index--
	r.SetLength(int16(index))

	p := &nats.CastPacket{CastNear: true, CharacterID: c.ID, Data: r, Type: nats.PLAYER_ATTACK}
	if err := p.Cast(); err != nil {
		log.Println("deal damage broadcast error:", err)
		return
	}
}
func (c *Character) DealDamage(ai *AI, dmg int, isSkill bool) {

	if ai == nil {
		return
	}
	if ai.Faction == 3 {
		return
	}
	r := DEAL_DAMAGE
	if isSkill {
		r = DEAL_SKILL_DAMAGE
	}

	if c == nil {
		log.Println("character is nil")
		return
	} else if ai.HP <= 0 {
		return
	}

	npcPos := GetNPCPosByID(ai.PosID)
	if npcPos == nil {
		log.Println("npc pos is nil")
		return
	}

	npcInfo, _ := GetNpcInfo(npcPos.NPCID)
	if npcInfo == nil {
		log.Println("npc is nil")
		return
	}

	characterCoordinate := ConvertPointToLocation(c.Coordinate)
	enemyCoordinate := ConvertPointToLocation(c.Coordinate)
	distance := utils.CalculateDistance(characterCoordinate, enemyCoordinate)
	if !isSkill {
		slots, err := c.InventorySlots()
		if err != nil {
			return
		}
		weapon := slots[c.WeaponSlot]
		info, ok := GetItemInfo(weapon.ItemID)
		if !ok && distance > 13 {
			return
		}
		if ok && float64(info.Range) < distance {
			return
		}
	} else {
		if distance > 13 {
			return
		}
	}

	for _, factionNPC := range ZhuangFactionMobs {
		if factionNPC == npcInfo.ID && c.Faction == 1 {
			return
		}
	}
	for _, factionNPC := range ShaoFactionMobs {
		if factionNPC == npcInfo.ID && c.Faction == 2 {
			return
		}
	}
	s := c.Socket

	stat := c.Socket.Stats
	seed := utils.RandInt(0, 1000) ///CRITICAL CHANCE
	if seed <= int64(stat.CriticalProbability) {
		critical := dmg + int(float32(dmg)*(float32(stat.CriticalRate)/1000))
		dmg = critical
		if isSkill {
			r = DEAL_SKILL_CRITICAL_DAMAGE
		} else {
			r = DEAL_NORMAL_CRITICAL_DAMAGE
		}
	}

	if c.Invisible {
		critical := dmg + int(float32(dmg)*(float32(stat.CriticalRate)/1000))
		if isSkill {
			r = DEAL_SKILL_CRITICAL_DAMAGE
		} else {
			r = DEAL_NORMAL_CRITICAL_DAMAGE
		}
		dmg = critical

		for _, invskillID := range InvisibilitySkillIDs {
			buff, _ := FindBuffByID(invskillID, c.ID)
			if buff != nil {
				buff.Duration = 0
				err := buff.Update()
				if err != nil {
					log.Print(err)
					return
				}
			}
		}

		if c.DuelID > 0 {
			opponent, _ := FindCharacterByID(c.DuelID)
			spawnData, _ := c.SpawnCharacter()

			r := utils.Packet{}
			r.Concat(spawnData)
			r.Overwrite(utils.IntToBytes(500, 2, true), 13) // duel state

			sock := GetSocket(opponent.UserID)
			if sock != nil {
				sock.Write(r)
			}
		}
	}
	if ai.Map == 233 {

		if isSkill && dmg > 100 {
			dmg = 150
		} else if !isSkill && dmg > 30 {
			dmg = 50
		}

	}

	if npcInfo.ID == 100080280 || npcInfo.ID == 100080281 || npcInfo.ID == 100080282 || npcInfo.ID == 100080283 || npcInfo.ID == 100080284 { //NCASH BOSSES CENTRAL ISLE
		if dmg > 20000 {
			dmg = 20000
		}
		if c.Level > int(npcInfo.Level)+20 || c.Level > 100 && npcInfo.Level <= 100 {
			dmg = 0
		}
	}
	if npcInfo.ID == 420104 {
		if dmg > 500 {
			dmg = 500
		}
	}

	if npcInfo.ID == 50009 { //RED DRAGON CENTRAL ISLE
		if dmg > 20000 {
			dmg = 20000
		}
		if c.Level > 100 && npcInfo.Level <= 100 {
			dmg = 0
		}
	} else if npcInfo.ID == 50010 { //RAKMA CENTRAL ISLE
		if dmg > 100000 {
			dmg = 100000
		}
		if c.Level > 200 {
			dmg = 1
		}
	}

	if npcInfo.ID == 424901 || npcInfo.ID == 424902 || npcInfo.ID == 424903 { //Flag Kingdom Statues
		slots, err := c.InventorySlots()
		if err == nil {
			if slots[11].ItemID != 0 {
				c.Socket.Write(messaging.InfoMessage("Clear first slot of the invenotry!"))
				return
			}
		}
		dmg = 10
	}

	if dmg > ai.HP {
		dmg = ai.HP
	}

	ai.HP -= dmg
	if ai.HP <= 0 {
		ai.HP = 0
	}

	d := ai.DamageDealers.Get(c.ID)
	if d == nil {
		ai.DamageDealers.Add(c.ID, &Damage{Damage: dmg, DealerID: c.ID})
	} else {
		d.(*Damage).Damage += dmg
		ai.DamageDealers.Add(c.ID, d)
	}

	buffs, err := FindBuffsByAiPseudoID(ai.PseudoID)
	chi := ai.CHI
	if npcInfo.Type == 29 {
		chi = (npcInfo.MaxHp / 2) / 10
	}
	index := 5
	r.Insert(utils.IntToBytes(uint64(ai.PseudoID), 2, true), index) // ai pseudo id
	index += 2
	r.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), index) // ai pseudo id
	index += 2
	r.Insert(utils.IntToBytes(uint64(ai.HP), 4, true), index) // ai current hp
	index += 4
	r.Insert(utils.IntToBytes(uint64(chi), 4, true), index) // ai current chi
	index += 4
	if err == nil {
		r.Overwrite(utils.IntToBytes(uint64(len(buffs)), 1, true), 21) //BUFF ID
		index = 22
		count := 0
		for _, buff := range buffs {
			r.Insert(utils.IntToBytes(uint64(buff.ID), 4, true), index) //BUFF ID
			index += 4
			if count < len(buffs)-1 {
				r.Insert(utils.IntToBytes(uint64(0), 2, true), index) //BUFF ID
				index += 2
			}
			count++
		}
		index += 4
	}
	index += 3
	r.Insert([]byte{0x00, 0x00, 0x00}, index) // INJURY
	index--
	r.SetLength(int16(index))
	p := &nats.CastPacket{CastNear: true, MobID: ai.ID, Data: r, Type: nats.MOB_ATTACK}
	if err := p.Cast(); err != nil {
		log.Println("deal damage broadcast error:", err)
		return
	}

	if !npcPos.Attackable {
		go ai.DropHandler(c)
	}

	if ai.HP <= 0 { // ai died

		if ai.Map == 233 {
			clan, err := FindGuildByID(c.GuildID)
			if err != nil {
				return
			}
			if clan != nil {
				temple := GetTempleDataByStatue(npcInfo.ID)
				if temple != nil {
					c.WinClanBatte(temple)
				}
			}
		}

		//-----------------YingYang Dungeon---------------------

		server := c.Socket.User.ConnectedServer
		if c.Map == 243 || c.Map == 215 {

			if YingYangMobsCounter[c.Map] == nil {
				YingYangMobsCounter[c.Map] = &DungeonMobsCounter{
					BlackBandits: 0,
					Rogues:       0,
					Ghosts:       0,
					Animals:      0,
				}
			}

			counter := YingYangMobsCounter[c.Map]

			if npcPos.NPCID == 60001 || npcPos.NPCID == 60002 || npcPos.NPCID == 60015 || npcPos.NPCID == 60016 {
				counter.BlackBandits--
				if counter.BlackBandits < 0 {
					counter.BlackBandits = 0
				}
				c.Socket.Write(messaging.InfoMessage(fmt.Sprintf("You have %d BlackBandits left to kill", counter.BlackBandits)))
			} else if npcPos.NPCID == 60004 || npcPos.NPCID == 60018 {
				counter.Rogues--
				if counter.Rogues < 0 {
					counter.Rogues = 0
				}
				c.Socket.Write(messaging.InfoMessage(fmt.Sprintf("You have %d Rogues left to kill", counter.Rogues)))
			} else if npcPos.NPCID == 60006 || npcPos.NPCID == 60007 || npcPos.NPCID == 60020 || npcPos.NPCID == 60021 {
				counter.Ghosts--
				if counter.Ghosts < 0 {
					counter.Ghosts = 0
				}
				c.Socket.Write(messaging.InfoMessage(fmt.Sprintf("You have %d Ghosts left to kill", counter.Ghosts)))
			} else if npcPos.NPCID == 60009 || npcPos.NPCID == 60010 || npcPos.NPCID == 60011 || npcPos.NPCID == 60012 ||
				npcPos.NPCID == 60023 || npcPos.NPCID == 60024 || npcPos.NPCID == 60025 || npcPos.NPCID == 60026 {
				counter.Animals--
				if counter.Animals < 0 {
					counter.Animals = 0
				}
				c.Socket.Write(messaging.InfoMessage(fmt.Sprintf("You have %d animals left to kill", counter.Animals)))
			}

		}
		//----------------------------------------------
		//-----------------Season Dungeon---------------------
		if SeasonCaveMobsCounter[server] == nil && c.Map == 212 {

			gomap, _ := c.ChangeMap(1, nil)
			s.Write(gomap)
		} else {
			if npcPos.NPCID == 45003 {
				SeasonCaveMobsCounter[server].Bats--
				if SeasonCaveMobsCounter[server].Bats < 0 {
					SeasonCaveMobsCounter[server].Bats = 0
				}
				c.Socket.Write(messaging.InfoMessage(fmt.Sprintf("You have %d Bats left to kill", SeasonCaveMobsCounter[server].Bats)))
			} else if npcPos.NPCID == 45004 {
				SeasonCaveMobsCounter[server].Spiders--
				if SeasonCaveMobsCounter[server].Spiders < 0 {
					SeasonCaveMobsCounter[server].Spiders = 0
				}
				c.Socket.Write(messaging.InfoMessage(fmt.Sprintf("You have %d Spiders left to kill", SeasonCaveMobsCounter[server].Spiders)))
			} else if npcPos.NPCID == 45005 {
				SeasonCaveMobsCounter[server].Snakes--
				if SeasonCaveMobsCounter[server].Snakes < 0 {
					SeasonCaveMobsCounter[server].Snakes = 0
				}
				c.Socket.Write(messaging.InfoMessage(fmt.Sprintf("You have %d Snakes left to kill", SeasonCaveMobsCounter[server].Snakes)))
			}
		}
		if npcPos.NPCID == 45008 {

			resp := utils.Packet{}
			slots, err := c.InventorySlots()
			if err == nil {
				reward := NewSlot()
				reward.ItemID = int64(200000038)
				reward.Quantity = 1440
				_, slot, _ := c.AddItem(reward, -1, true)
				resp.Concat(slots[slot].GetData(slot))
				s.Write(resp)
			}

		} else if npcPos.NPCID == 45009 {
			resp := utils.Packet{}
			slots, err := c.InventorySlots()
			if err == nil {
				reward := NewSlot()
				reward.ItemID = int64(200000039)
				reward.Quantity = 720
				_, slot, _ := c.AddItem(reward, -1, true)
				resp.Concat(slots[slot].GetData(slot))
				s.Write(resp)
			}

		}
		//----------------------------------------------
		if c.Map == 255 && IsFactionWarStarted() { //faction war
			if c.Faction == 1 {
				if npcPos.NPCID == 425506 {
					AddPointsToFactionWarFaction(5, 1)
					c.WarContribution += 5
				}
				if npcPos.NPCID == 425505 {
					AddPointsToFactionWarFaction(50, 1)
					c.WarContribution += 50
				}
				if npcPos.NPCID == 425507 {
					AddPointsToFactionWarFaction(7, 1)
					c.WarContribution += 7
				}
				if npcPos.NPCID == 425508 {
					AddPointsToFactionWarFaction(500, 1)
					c.WarContribution += 500
				}
			}
			if c.Faction == 2 {
				if npcPos.NPCID == 425501 {
					AddPointsToFactionWarFaction(5, 2)
					c.WarContribution += 5
				}
				if npcPos.NPCID == 425502 {
					AddPointsToFactionWarFaction(50, 2)
					c.WarContribution += 50
				}
				if npcPos.NPCID == 425503 {
					AddPointsToFactionWarFaction(7, 2)
					c.WarContribution += 7
				}
				if npcPos.NPCID == 425504 {
					AddPointsToFactionWarFaction(500, 2)
					c.WarContribution += 500
				}
			}

			if npcInfo.ID == 424201 && WarStarted {
				OrderPoints -= 200
			} else if npcInfo.ID == 424202 && WarStarted {
				ShaoPoints -= 200
			}
		}
		if ai.Once {
			ai.Handler = nil
		} else {
			ai.IsDead = true
			respawnTimeX := npcPos.RespawnTime - int(float32(npcPos.RespawnTime)*0.15)
			respawnTimeY := npcPos.RespawnTime + int(float32(npcPos.RespawnTime)*0.15)
			respawnRange := utils.RandInt(int64(respawnTimeX), int64(respawnTimeY))
			if ai.Map == 243 {
				respawnRange = 480
			}
			time.AfterFunc(time.Duration(respawnRange)*time.Second, func() { // respawn mob n secs later
				curCoordinate := ConvertPointToLocation(ai.Coordinate)
				minCoordinate := ConvertPointToLocation(npcPos.MinLocation)
				maxCoordinate := ConvertPointToLocation(npcPos.MaxLocation)

				X := utils.RandFloat(minCoordinate.X, maxCoordinate.X)
				Y := utils.RandFloat(minCoordinate.Y, maxCoordinate.Y)

				X = (X / 3) + 2*curCoordinate.X/3
				Y = (Y / 3) + 2*curCoordinate.Y/3

				coordinate := &utils.Location{X: X, Y: Y}
				ai.TargetLocation = *coordinate
				ai.SetCoordinate(coordinate)

				ai.HP = npcInfo.MaxHp
				ai.IsDead = false
			})
		}

		exp := int64(0)
		if c.Level <= 100 {
			exp = npcInfo.Exp
		} else if c.Level <= 200 {
			exp = npcInfo.DivineExp
		} else {
			exp = npcInfo.DarknessExp
		}
		if c.Relaxation > 0 && exp > 0 {
			exp *= 2
			c.Relaxation--
			c.HousingDetails()
		}

		// EXP gain for party members
		party := FindParty(c)
		if party == nil || (party != nil && c.GroupSettings.ExperienceSharingMethod == 1) {
			// EXP gained
			r, levelUp := c.AddExp(exp)
			if levelUp {
				statData, err := c.GetStats()
				if err == nil {
					s.Write(statData)
				}
			}
			s.Write(r)
		} else {
			members := funk.Filter(party.GetMembers(), func(m *PartyMember) bool {
				return m.Accepted || m.ID == c.ID
			}).([]*PartyMember)
			members = append(members, &PartyMember{Character: party.Leader, Accepted: true})
			coordinate := ConvertPointToLocation(c.Coordinate)
			for _, m := range members {
				user, err := FindUserByID(m.UserID)
				if err != nil || user == nil || (c.Level-m.Level) > 20 {
					continue
				}
				memberCoordinate := ConvertPointToLocation(m.Coordinate)
				if m.ID == c.ID && !m.Accepted {
					break
				}
				if m.Map != c.Map || s.User.ConnectedServer != user.ConnectedServer ||
					utils.CalculateDistance(coordinate, memberCoordinate) > 100 || m.Socket.Stats.HP <= 0 {
					continue
				}

				exp := int64(0)
				if m.Level <= 100 {
					exp = npcInfo.Exp
				} else if m.Level <= 200 {
					exp = npcInfo.DivineExp
				} else {
					exp = npcInfo.DarknessExp
				}

				exp /= int64(len(members))

				r, levelUp := m.AddExp(exp)
				if levelUp {
					statData, err := m.GetStats()
					if err == nil {
						m.Socket.Write(statData)
					}
				}
				m.Socket.Write(r)
			}
		}
		//QUEST ITEMS DROP
		/*	if funk.Contains(c.questMobsIDs, npcInfo.ID) {

			resp := utils.Packet{}
			item, itemcount, questID := c.GetQuestItemsDrop(npcInfo.ID)
			itemData, slotID, err := c.AddItem(&InventorySlot{ItemID: item, Quantity: 1}, -1, true)
			if err != nil {
				log.Print("Quest item add error!")
				return
			}
			qslots, err := c.InventorySlots()
			if err != nil {
				log.Print("Quest item add error 2!")
				return
			}
			qitem := qslots[slotID]
			if qitem.Quantity <= uint(itemcount) {
				resp.Concat(*itemData)
				iteminfo, _ := GetItemInfo(item)
				itemName := iteminfo.Name
				mess := messaging.InfoMessage(fmt.Sprintf("Acquired the %s", itemName))
				resp.Concat(mess)
				c.Socket.Write(resp)
			}
			if qitem.Quantity >= uint(itemcount) {
				quest, _ := FindPlayerQuestByID(questID, c.ID)
				quest.QuestState = 4
				quest.Update()
				test, _ := c.LoadReturnQuests(quest.ID, quest.QuestState)
				c.Socket.Write(test)
			}

		}*/
		if npcInfo.ID == 43401 {
			if c.Exp >= 544951059310 && c.Level == 200 {
				resp := utils.Packet{}
				slots, err := c.InventorySlots()
				if err != nil {
					log.Print("Kill wyrm no slot in inverntory")
					return
				}
				reward := NewSlot()
				reward.ItemID = int64(90000304)
				reward.Quantity = 1
				_, slot, _ := c.AddItem(reward, -1, true)
				resp.Concat(slots[slot].GetData(slot))
				c.Socket.Write(resp)
				c.Socket.Write(messaging.InfoMessage("You kill the Wyrm, now you can make the transformation."))
			}
		}

		if npcInfo.ID == 420108 || npcInfo.ID == 41941 || npcInfo.ID == 430108 {
			if c.Exp >= 233332051410 && c.Level == 100 {
				itemData, _, err := c.AddItem(&InventorySlot{ItemID: 90000190, Quantity: uint(5000)}, -1, false)
				if err != nil {
					return
				} else if itemData == nil {
					return
				}
				c.Socket.Write(*itemData)
				c.Socket.Write(messaging.InfoMessage("You killed the Dragon, now you can make the transformation."))
			}
			if c.Level <= 100 { //Red Dragon pill
				prob := 20
				rand := int(utils.RandInt(0, 100))
				if rand < prob {
					itemData, _, err := c.AddItem(&InventorySlot{ItemID: 99059998, Quantity: 5000}, -1, false)
					if err != nil {
						return
					} else if itemData == nil {
						return
					}
					c.Socket.Write(*itemData)
				}
			}
		}
		if npcInfo.ID == 42451 || npcInfo.ID == 42452 {
			if c.Level <= 200 { //1000 Centipede Pills
				prob := 20
				rand := int(utils.RandInt(0, 100))
				if rand < prob {
					itemData, _, err := c.AddItem(&InventorySlot{ItemID: 99059995, Quantity: 5000}, -1, false)
					if err != nil {
						return
					} else if itemData == nil {
						return
					}
					c.Socket.Write(*itemData)
				}

				prob = 5
				rand = int(utils.RandInt(0, 100))
				if rand < prob {
					itemData, _, err := c.AddItem(&InventorySlot{ItemID: 90000240, Quantity: 1}, -1, false)
					if err != nil {
						return
					} else if itemData == nil {
						return
					}
					c.Socket.Write(*itemData)
				}
			}
		}

		if npcInfo.ID == 424901 || npcInfo.ID == 424902 || npcInfo.ID == 424903 { //Flag Kingdom Statues
			slots, err := c.InventorySlots()
			if err == nil {
				if slots[11].ItemID != 0 {
					c.Socket.Write(messaging.InfoMessage("Clear first invenotry slot!"))
					return
				}
			}
			var itemid int64
			itemid = 99059990
			if npcInfo.ID == 424902 {
				itemid = 99059991
			} else if npcInfo.ID == 424903 {
				itemid = 99059992
			}
			itemData, _, err := c.AddItem(&InventorySlot{ItemID: itemid, Quantity: 1}, -1, false)
			if err != nil {
				return
			} else if itemData == nil {
				return
			}
			FactionCapturedFlagNotification()
			c.Socket.Write(*itemData)
		}

		if npcInfo.ID == 42562 { //Evil Spirit`s Essence
			prob := 10
			rand := int(utils.RandInt(0, 100))
			if rand < prob {
				itemData, _, err := c.AddItem(&InventorySlot{ItemID: 99059989, Quantity: 5000}, -1, false)
				if err != nil {
					return
				} else if itemData == nil {
					return
				}
				c.Socket.Write(*itemData)
			}

			prob = 5
			rand = int(utils.RandInt(0, 100))
			if rand < prob {
				itemData, _, err := c.AddItem(&InventorySlot{ItemID: 90000241, Quantity: 1}, -1, false)
				if err != nil {
					return
				} else if itemData == nil {
					return
				}
				c.Socket.Write(*itemData)
			}

		}
		if npcInfo.ID == 42716 || npcInfo.ID == 42717 { //Moonwind lord
			prob := 5
			rand := int(utils.RandInt(0, 100))
			if rand < prob {
				itemData, _, err := c.AddItem(&InventorySlot{ItemID: 90000242, Quantity: 1}, -1, false)
				if err != nil {
					return
				} else if itemData == nil {
					return
				}
				c.Socket.Write(*itemData)
			}
		}
		if npcInfo.ID == 42910 { //Kunlun Snowmaster lord
			prob := 5
			rand := int(utils.RandInt(0, 100))
			if rand < prob {
				itemData, _, err := c.AddItem(&InventorySlot{ItemID: 90000243, Quantity: 1}, -1, false)
				if err != nil {
					return
				} else if itemData == nil {
					return
				}
				c.Socket.Write(*itemData)
			}
		}
		if npcInfo.ID == 60123 { //Rudolf
			prob := 100
			rand := int(utils.RandInt(0, 100))
			if rand < prob {
				itemData, _, err := c.AddItem(&InventorySlot{ItemID: 13003133, Quantity: 1}, -1, false)
				if err != nil {
					return
				} else if itemData == nil {
					return
				}
				c.Socket.Write(*itemData)
			}
		}
		if c.Map == 243 {

			if npcInfo.ID == 60003 || npcInfo.ID == 60005 || npcInfo.ID == 60008 || npcInfo.ID == 60013 || npcInfo.ID == 60014 { //Dungeon Bosses Black Plates

				dealers := ai.DamageDealers.Values()

				sort.Slice(dealers, func(i, j int) bool {
					di := dealers[i].(*Damage)
					dj := dealers[j].(*Damage)
					return di.Damage > dj.Damage
				})
				for _, dealer := range dealers {

					char, err := FindCharacterByID(dealer.(*Damage).DealerID)
					if err != nil {
						continue
					}
					if !char.IsActive || !char.IsOnline {
						continue
					}

					if char.Level <= 100 {
						itemData, _, err := char.AddItem(&InventorySlot{ItemID: 99002478, Quantity: 500}, -1, false)
						if err != nil {
							return
						} else if itemData == nil {
							return
						}
						char.Socket.Write(*itemData)
					}
				}
			}

		} else if c.Map == 215 {

			if npcInfo.ID == 60017 || npcInfo.ID == 60019 || npcInfo.ID == 60022 || npcInfo.ID == 60027 || npcInfo.ID == 60028 { //Dungeon Bosses Black Plates

				dealers := ai.DamageDealers.Values()

				sort.Slice(dealers, func(i, j int) bool {
					di := dealers[i].(*Damage)
					dj := dealers[j].(*Damage)
					return di.Damage > dj.Damage
				})
				for _, dealer := range dealers {

					char, err := FindCharacterByID(dealer.(*Damage).DealerID)
					if err != nil {
						continue
					}
					if !char.IsActive || !char.IsOnline {
						continue
					}

					if char.Level > 100 && char.Level < 201 {
						itemData, _, err := char.AddItem(&InventorySlot{ItemID: 99002479, Quantity: 500}, -1, false)
						if err != nil {
							return
						} else if itemData == nil {
							return
						}
						char.Socket.Write(*itemData)
					}
				}
			}

		}
		if npcInfo.ID == 44499 {
			resp := utils.Packet{}
			slots, err := c.InventorySlots()
			if err == nil {
				reward := NewSlot()
				reward.ItemID = int64(17502418)
				reward.Quantity = 1
				_, slot, _ := c.AddItem(reward, -1, true)
				resp.Concat(slots[slot].GetData(slot))
				s.Write(resp)
			}
		}
		if npcInfo.ID == 42451 || npcInfo.ID == 42452 || npcInfo.ID == 420308 || npcInfo.ID == 420309 {
			rand := utils.RandInt(0, 1000)
			if rand < 100 {
				amount := uint64(1000)
				s.User.NCash += 1000
				s.User.Update()
				s.Write(messaging.InfoMessage(fmt.Sprintf("You earned %d nC from boss hunting.", amount)))
			}
		}

		// PTS gained LOOT
		c.PTS++
		if c.PTS%100 == 0 {
			r = c.GetPTS()
			c.HasLot = true
			s.Write(r)
		}
		pvpGoldBonus := 0.0
		if funk.Contains(PVPServers, int16(c.Socket.User.ConnectedServer)) {
			pvpGoldBonus = 0.05
		}
		goldMmultiplier := (GOLD_RATE * stat.GoldMultiplier) + pvpGoldBonus
		if c.Level < 50 {
			goldMmultiplier *= 2.5
		} else if c.Level <= 100 {
			goldMmultiplier *= 1.75
		} else {
			goldMmultiplier *= 1.50
		}
		goldDrop := int64(npcInfo.GoldDrop)
		goldDrop = int64(float64(goldDrop) * goldMmultiplier)
		amount := uint64(utils.RandInt(goldDrop/2, goldDrop))

		lootOwner := c

		if party == nil || (party != nil && c.GroupSettings.LootDistriburionMethod == 1) {
			if goldDrop > 0 {
				r = c.LootGold(amount)
				s.Write(r)
			}
		} else { //GOLD gain for party members

			members := funk.Filter(party.GetMembers(), func(m *PartyMember) bool {
				return m.Accepted || m.ID == c.ID
			}).([]*PartyMember)
			members = append(members, &PartyMember{Character: party.Leader, Accepted: true})

			coordinate := ConvertPointToLocation(c.Coordinate)
			if c.GroupSettings.LootDistriburionMethod == 2 {
				rand := utils.RandInt(0, int64(len(members)))
				m := members[rand]
				r = m.LootGold(uint64(amount))
				m.Socket.Write(r)
				lootOwner = m.Character

			} else if c.GroupSettings.LootDistriburionMethod == 3 {
				m := party.Leader
				r = m.LootGold(uint64(amount))
				m.Socket.Write(r)
				lootOwner = m
			} else if c.GroupSettings.LootDistriburionMethod == 4 {
				for _, m := range members {
					user, err := FindUserByID(m.UserID)
					if err != nil || user == nil || (c.Level-m.Level) > 20 {
						continue
					}

					memberCoordinate := ConvertPointToLocation(m.Coordinate)

					if m.ID == c.ID && !m.Accepted {
						break
					}

					if c == m.Character || m.Map != c.Map || s.User.ConnectedServer != user.ConnectedServer ||
						utils.CalculateDistance(coordinate, memberCoordinate) > 100 || m.Socket.Stats.HP <= 0 {
						continue
					}

					if goldDrop > 0 {
						amount /= uint64(len(members))
						r = m.LootGold(uint64(amount))
						m.Socket.Write(r)
					}
				}
				rand := utils.RandInt(0, int64(len(members)))
				lootOwner = members[rand].Character
				if lootOwner == party.Leader && members[rand-1] != nil {
					lootOwner = members[rand-1].Character
				}
			}

		}

		//Item dropped
		go func() {
			claimer, err := ai.FindClaimer()
			if err != nil || claimer == nil {
				return
			}
			if claimer == c && lootOwner != c {
				claimer = lootOwner
			}

			dropMaxLevel := int(npcInfo.Level + 25)
			if claimer.Level <= dropMaxLevel {
				claimer.KilledMobs++
				ai.DropHandler(claimer)
			}

			if npcInfo.ID == 50009 || npcInfo.ID == 50010 {
				ai.RewardNcash()
			}
			time.AfterFunc(time.Second, func() {
				ai.DamageDealers.Clear()
			})
		}()

		time.AfterFunc(time.Second, func() { // disappear mob 1 sec later

			ai.TargetPlayerID = 0
			ai.TargetPetID = 0
			ai.IsDead = true
		})
		if npcInfo.StageLink != 0 {
			st := BossStages[npcInfo.StageLink]
			if st != nil {
				for index, nextStage := range st.Stages {
					if nextStage == npcInfo.ID && index < 4 {
						if st.Stages[index+1] != 0 {
							if npcPos.IsNPC {
								continue
							}
							npcinfo, ok := GetNpcInfo(st.Stages[index+1])
							if !ok || npcinfo == nil {
								continue
							}
							newnpcPos := &NpcPosition{
								ID:          len(GetNPCPostions()),
								NPCID:       st.Stages[index+1],
								MapID:       c.Map,
								Min_X:       npcPos.Min_X,
								Min_Y:       npcPos.Min_Y,
								Max_X:       npcPos.Max_X,
								Max_Y:       npcPos.Max_Y,
								Count:       1,
								RespawnTime: npcPos.RespawnTime,
								IsNPC:       false,
								Attackable:  true,
								Rotation:    npcPos.Rotation,
								MinLocation: npcPos.MinLocation,
								MaxLocation: npcPos.MaxLocation,
							}
							GenerateIDForNPC(newnpcPos)
							SetNPCPos(newnpcPos.ID, newnpcPos)

							newai := &AI{
								ID:             len(AIs),
								HP:             npcInfo.MaxHp,
								Map:            ai.Map,
								PosID:          newnpcPos.ID,
								RunningSpeed:   float64(npcInfo.RunningSpeed),
								Server:         ai.Server,
								WalkingSpeed:   float64(npcInfo.WalkingSpeed),
								Once:           true,
								CanAttack:      ai.CanAttack,
								Faction:        ai.Faction,
								IsDead:         false,
								OnSightPlayers: ai.OnSightPlayers,
								Coordinate:     ai.Coordinate,
								TargetLocation: ai.TargetLocation,
								NPCpos:         newnpcPos,
							}
							GenerateIDForAI(newai)
							AIs[newai.ID] = newai
							err := SpawnMob(newnpcPos, newai)
							if err != nil {
								print(err)
							}
							c.OnSight.MobMutex.RLock()
							_, ok = c.OnSight.Mobs[newai.ID]
							c.OnSight.MobMutex.RUnlock()

							if ok {
								c.OnSight.MobMutex.Lock()
								delete(c.OnSight.Mobs, newai.ID)
								c.OnSight.MobMutex.Unlock()
							}
							break
						}

					}
				}
			}
		}
	} else if ai.TargetPlayerID == 0 {
		ai.IsMoving = false
		ai.MovementToken = 0
		ai.TargetPlayerID = c.ID
	} else {
		ai.IsMoving = false
		ai.MovementToken = 0
	}

}

func (c *Character) GetPetStats() []byte {

	slots, err := c.InventorySlots()
	if err != nil {
		return nil
	}

	petSlot := slots[0x0A]
	pet := petSlot.Pet
	if pet == nil {
		return nil
	}

	resp := utils.Packet{}
	resp = petSlot.GetPetStats(c)
	resp.Concat(petSlot.GetData(0x0A))
	return resp
}

func (c *Character) StartPvP(timeLeft int) {

	info, resp := "", utils.Packet{}
	if timeLeft > 0 {
		info = fmt.Sprintf("Duel will start %d seconds later.", timeLeft)
		time.AfterFunc(time.Second, func() {
			c.StartPvP(timeLeft - 1)
		})

	} else if c.DuelID > 0 {
		info = "Duel has started."
		resp.Concat(c.OnDuelStarted())
	}

	resp.Concat(messaging.InfoMessage(info))
	c.Socket.Write(resp)
}

func (c *Character) CanAttack(enemy *Character) bool {
	if funk.Contains(PVPServers, int16(c.Socket.User.ConnectedServer)) {
		return true
	}

	if (c.Map == 255) && c.Faction == enemy.Faction {
		return false
	}
	rr := (c.DuelID == enemy.ID && c.DuelStarted) || funk.Contains(PvPZones, c.Map)
	return rr
}

func (c *Character) OnDuelStarted() []byte {

	c.DuelStarted = true
	statData, _ := c.GetStats()

	opponent, err := FindCharacterByID(c.DuelID)
	if err != nil || opponent == nil {
		return nil
	}

	opData, err := opponent.SpawnCharacter()
	if err != nil || opData == nil || len(opData) < 13 {
		return nil
	}

	r := utils.Packet{}
	r.Concat(opData)
	r.Overwrite(utils.IntToBytes(500, 2, true), 13) // duel state

	resp := utils.Packet{}
	resp.Concat(opponent.GetHPandChi())
	resp.Concat(r)
	resp.Concat(statData)
	resp.Concat([]byte{0xAA, 0x55, 0x02, 0x00, 0x2A, 0x04, 0x55, 0xAA})
	return resp
}

func (c *Character) PoisonDamage(damage int) {
	if damage < 3 {
		damage = 3
	}
	enemySt := c.Socket.Stats
	enemySt.HP -= damage
	if enemySt.HP < 0 {
		enemySt.HP = 0
	}

	if c.Meditating { //STOP MEDITATION
		c.Meditating = false
		med := MEDITATION_MODE
		med.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), 6) // character pseudo id
		med[8] = 0

		p := nats.CastPacket{CastNear: true, CharacterID: c.ID, Type: nats.MEDITATION_MODE, Data: med}
		if err := p.Cast(); err == nil {
			c.Socket.Write(med)
		}
	}

	r := DEAL_POISON_DAMAGE
	r.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), 5)
	r.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), 7)
	r.Insert(utils.IntToBytes(uint64(enemySt.HP), 4, true), 9)
	r.Insert(utils.IntToBytes(uint64(enemySt.CHI), 4, true), 13)

	r.Concat(c.GetHPandChi())
	p := &nats.CastPacket{CastNear: true, CharacterID: c.ID, Data: r, Type: nats.PLAYER_ATTACK}
	if err := p.Cast(); err != nil {
		log.Println("deal damage broadcast error:", err)
	}

}

func (c *Character) Paralysis(npc *NPC) {
	atk := npc.ParalysisATK
	rand := rand.Intn(1000)
	seconds := npc.ParaInflictTime / 1000
	def := c.Socket.Stats.ParalysisDEF
	probabilty := atk - def

	if rand <= probabilty {
		if !c.Paralised {

			c.AddBuff(BuffInfections[259], int64(seconds))
			c.Paralised = true

		}
	}
}

func (c *Character) Confusion(npc *NPC) {
	atk := npc.ConfusionATK
	rand := rand.Intn(1000)
	seconds := npc.ConfInflictTime / 1000
	def := c.Socket.Stats.ConfusionDEF
	probabilty := atk - def

	if rand <= probabilty {
		if !c.Confused {

			c.AddBuff(BuffInfections[258], int64(seconds))
			c.Confused = true
		}
	}

}
func (c *Character) ResetPlayerSkillBook() {

	//resp := utils.Packet{}
	var BookIDs []int64
	skills, err := FindSkillsByID(c.ID)
	if err != nil || skills == nil {
		return
	}

	skillSlots, err := skills.GetSkills()
	if err != nil || skillSlots == nil {
		return
	}
	for j := 0; j < 5; j++ {
		if skillSlots.Slots[j].BookID != 0 {
			BookIDs = append(BookIDs, skillSlots.Slots[j].BookID)
		}
	}
	for j := 5; j <= 7; j++ {
		if skillSlots.Slots[j].BookID != 0 {
			set := skillSlots.Slots[j]
			if len(set.Skills) == 0 || set.Skills[0].Plus <= 0 {
				continue
			}
			set.Skills[0].Plus = 0
			skills.SetSkills(skillSlots)
		}
	}

	for k := 0; k < len(BookIDs); k++ {

		skillInfos := SkillBooks[BookIDs[k]].SkillTree
		set := &SkillSet{BookID: BookIDs[k]}
		c := 0
		for i := 1; i <= 24; i++ { // there should be 24 skills with empty ones
			if len(skillInfos) <= c {
				set.Skills = append(set.Skills, &SkillTuple{SkillID: 0, Plus: 0})
			} else if si := skillInfos[c]; si.Slot == i {
				//log.Print(fmt.Sprintf("skillID: %d"), si.ID)
				tuple := &SkillTuple{SkillID: si.ID, Plus: 0}
				set.Skills = append(set.Skills, tuple)
				c++
			} else {
				set.Skills = append(set.Skills, &SkillTuple{SkillID: 0, Plus: 0})
			}
		}
		skillSlots.Slots[k] = set
		skills.SetSkills(skillSlots)
		err := skills.Update()
		if err != nil {
			log.Print(err)
			return
		}
	}

	spIndex := utils.SearchUInt64(SkillPoints, uint64(1))
	spIndex2 := utils.SearchUInt64(SkillPoints, uint64(c.Exp))
	skPts := spIndex2 - spIndex
	skPts += c.Level * int(c.Reborns) * 3
	skills.SkillPoints = skPts
	if c.Level > 100 {
		for i := 101; i <= c.Level; i++ {
			skills.SkillPoints += EXPs[int16(i)].SkillPoints
		}
	}

	skills.Update()

}
func (c *Character) CalculateInjury() []int {
	remaining := c.Injury
	divCount := []int{0, 0, 0, 0}
	divNumbers := []float64{0.1, 0.7, 1.09, 17.48}
	for i := len(divNumbers) - 1; i >= 0; i-- {
		if remaining < divNumbers[i] || remaining == 0 {
			continue
		}
		test := remaining / divNumbers[i]
		if test > 15 {
			test = 15
		}
		divCount[i] = int(test)
		test2 := test * divNumbers[i]
		remaining -= test2
	}
	return divCount
}
func (enemy *Character) LosePlayerExp(dealer *Character) (int64, error) {
	percent := float64(enemy.Level) - float64(dealer.Level)*0.1
	level := int16(enemy.Level)
	expminus := int64(0)
	if level >= 10 {
		oldExp := EXPs[level-1].Exp
		resp := EXP_SKILL_PT_CHANGED
		if oldExp <= enemy.Exp {
			per := float64(percent) / 100
			expLose := float64(enemy.Exp) * float64(1-per)
			if int64(expLose) >= oldExp {
				exp := enemy.Exp - int64(expLose)
				expminus = int64(float64(exp) * float64(1-0.30))
				enemy.Exp = int64(expLose)
			} else {
				exp := enemy.Exp - oldExp
				expminus = int64(float64(exp) * float64(1-0.30))
				enemy.Exp = oldExp
			}
		}
		resp.Insert(utils.IntToBytes(uint64(enemy.Exp), 8, true), 5)                        // character exp
		resp.Insert(utils.IntToBytes(uint64(enemy.Socket.Skills.SkillPoints), 4, true), 13) // character skill points
		enemy.Update()
		enemy.Socket.Skills.Update()
		enemy.Socket.Write(resp)
	}
	return expminus, nil
}

func (c *Character) AddRepurchaseItem(slot int) ([]byte, error) {
	slots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}

	item := slots[slot]
	info, ok := GetItemInfo(item.ItemID)
	if !ok {
		return nil, errors.New("AddRepurchaseItem:item not found")
	}
	if info.TimerType > 0 { //don't add timed items
		return nil, nil
	}

	c.RepurchaseList.Push(*item)

	count := len(c.RepurchaseList.Slots)
	data := item.GetData(int16(count - 1))
	data = data[6 : len(data)-2]

	resp := utils.Packet{}
	if count < 12 {
		resp = REPURCHASE_LIST
		resp.Insert(data, 7) // item data

	} else {
		resp = c.RepurchaseList.Data(REPURCHASE_LIST)
	}

	return resp, nil
}

func (c *Character) GoDivine() {

	s := c.Socket

	c.Level = 101
	c.Type += 10
	c.Exp = 233332051411
	s.Character.Type = c.Type

	skills, err := FindSkillsByID(c.ID)
	if err != nil {
		return
	}
	skillSlots, err := skills.GetSkills()
	if err != nil {
		return
	}

	sd := utils.Packet{}

	for i := 0; i <= 7; i++ {
		if skillSlots.Slots[i].BookID != 0 {
			if i < 5 {
				if skillSlots.Slots[i].BookID != 0 {
					skillData, _ := c.RemoveSkill(byte(i), skillSlots.Slots[i].BookID)
					sd.Concat(skillData)
				}
			} else {
				skillIndex := byte(0)
				if i == 0 {
					skillIndex = 5
				} else if i == 1 {
					skillIndex = 6
				} else if i == 7 {
					skillIndex = 8
				}
				skillData, _ := c.RemovePassiveSkill(skillIndex, byte(i), skillSlots.Slots[i].BookID)
				sd.Concat(skillData)
			}
		}
	}

	spIndex := utils.SearchUInt64(SkillPoints, uint64(1))
	spIndex2 := utils.SearchUInt64(SkillPoints, uint64(c.Exp))
	skPts := spIndex2 - spIndex
	skPts += c.Level * int(c.Reborns) * 3
	skills.SkillPoints = skPts
	if c.Level > 100 {
		for i := 101; i <= c.Level; i++ {
			skills.SkillPoints += EXPs[int16(i)].SkillPoints
		}
	}

	c.Socket.Skills.Update()

	c.Class = 0
	c.Update()
	s.User.Update()
	c.Update()

	ATARAXIA := utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x21, 0xE3, 0x55, 0xAA, 0xaa, 0x55, 0x0b, 0x00, 0x75, 0x00, 0x01, 0x00, 0x80, 0xa1, 0x43, 0x00, 0x00, 0x3d, 0x43, 0x55, 0xaa}
	resp := ATARAXIA
	resp[6] = byte(c.Type) // character type

	ANNOUNCEMENT := utils.Packet{0xAA, 0x55, 0x54, 0x00, 0x71, 0x14, 0x51, 0x55, 0xAA}
	msg := "At this moment I mark my name on list of Top master in Strong HERO."
	announce := ANNOUNCEMENT
	index := 6
	announce[index] = byte(len(c.Name) + len(msg))
	index++
	announce.Insert([]byte("["+c.Name+"]"), index) // character name
	index += len(c.Name) + 2
	announce.Insert([]byte(msg), index) // character name
	announce.SetLength(int16(binary.Size(announce) - 6))
	p := nats.CastPacket{CastNear: false, Data: announce}
	p.Cast()

	statData, _ := c.GetStats()
	resp.Concat(statData)
	resp.Concat(sd)

	skillsData, err := s.Skills.GetSkillsData()
	if err != nil {
		return
	}
	s.Write(skillsData)
	s.Write(resp)

}

func (c *Character) Reborn() {
	if c.Exp >= 233332051410 && c.Level == 100 && c.Reborns < 4 {
		s := c.Socket
		stat := c.Socket.Stats

		newstat := startingStats[c.Type]
		stat.STR = newstat.STR
		stat.DEX = newstat.DEX
		stat.INT = newstat.INT

		stat.StatPoints = 5

		c.Level = 1
		c.Exp = 1
		c.Class = 0
		c.Socket.Skills.SkillPoints = 0
		c.Reborns++

		skills, err := FindSkillsByID(c.ID)
		if err != nil {
			return
		}
		skillSlots, err := skills.GetSkills()
		if err != nil {
			return
		}

		sd := utils.Packet{}

		for i := 0; i <= 7; i++ {
			if skillSlots.Slots[i].BookID != 0 {
				if i < 5 {
					if skillSlots.Slots[i].BookID != 0 {
						skillData, _ := c.RemoveSkill(byte(i), skillSlots.Slots[i].BookID)
						sd.Concat(skillData)
					}
				} else {
					skillIndex := byte(0)
					if i == 0 {
						skillIndex = 5
					} else if i == 1 {
						skillIndex = 6
					} else if i == 7 {
						skillIndex = 8
					}
					skillData, _ := c.RemovePassiveSkill(skillIndex, byte(i), skillSlots.Slots[i].BookID)
					sd.Concat(skillData)
				}
			}
		}
		c.Update()
		s.User.Update()

		ATARAXIA := utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x21, 0xE3, 0x55, 0xAA, 0xaa, 0x55, 0x0b, 0x00, 0x75, 0x00, 0x01, 0x00, 0x80, 0xa1, 0x43, 0x00, 0x00, 0x3d, 0x43, 0x55, 0xaa}
		resp := ATARAXIA
		resp[6] = byte(c.Type) // character type

		ANNOUNCEMENT := utils.Packet{0xAA, 0x55, 0x54, 0x00, 0x71, 0x14, 0x51, 0x55, 0xAA}
		msg := "At this moment I mark my name on the list of Reborns."
		announce := ANNOUNCEMENT
		index := 6
		announce[index] = byte(len(c.Name) + len(msg))
		index++
		announce.Insert([]byte("["+c.Name+"]"), index) // character name
		index += len(c.Name) + 2
		announce.Insert([]byte(msg), index) // character name
		announce.SetLength(int16(binary.Size(announce) - 6))
		p := nats.CastPacket{CastNear: false, Data: announce}
		p.Cast()

		statData, _ := c.GetStats()
		resp.Concat(statData)
		resp.Concat(sd)

		skillsData, err := s.Skills.GetSkillsData()
		if err != nil {
			return
		}
		s.Write(skillsData)
		s.Write(resp)
	}
}
func (c *Character) IsAllowedInMap(Map int16) bool {
	s := c.Socket
	if s.User.UserType > 2 {
		return true
	}

	if Map == 33 && c.Level != 200 && c.Exp < 544951059310 {
		return false
	}

	if Map == 72 || Map == 73 || Map == 74 || Map == 75 {
		f := func(item *InventorySlot) bool {
			return item.Activated
		}
		_, item, err := s.Character.FindItemInInventory(f, 200000038, 200000039)
		if err != nil {
			return false
		} else if item == nil {
			return false
		}
	} else if maps, ok := DKMaps[Map]; ok {
		if maps[1] == Map || maps[2] == Map {
			f := func(item *InventorySlot) bool {
				return item.Activated
			}
			_, item, err := s.Character.FindItemInInventory(f, 15700040, 15710087)
			if err != nil {
				return false
			} else if item == nil {
				return false
			}
		}
	}
	if (Map == 14 && c.Faction == 2) || (Map == 15 && c.Faction == 1) {
		s.Write((messaging.InfoMessage("You don't meet the requirements to enter this region.")))
		return false
	}
	if SavePoints[int(Map)] != nil {
		if c.Level < SavePoints[int(Map)].MinLevel {
			s.Write((messaging.InfoMessage("You don't meet the requirement level to enter this region.")))
			return false
		}
	}

	return true
}

func (c *Character) WinClanBatte(temple *TempleData) {
	guild, err := FindGuildByID(c.GuildID)
	if err != nil {
		log.Print(err)
		return
	}
	if guild == nil {
		return
	}

	infection := BuffInfections[temple.BuffID]
	if infection == nil {
		return
	}
	members, err := guild.GetMembers()
	if err != nil {
		log.Print(err)
		return
	}
	for _, member := range members {
		char, err := FindCharacterByID(member.ID)
		if err != nil {
			log.Print(err)
			continue
		}
		if char == nil {
			continue
		}
		buff, err := FindBuffByID(infection.ID, c.ID)
		if err != nil {
			log.Print(err)
			continue
		} else {
			if buff == nil {
				buff = &Buff{ID: infection.ID, CharacterID: char.ID, StartedAt: GetServerEpoch(), Duration: 7200, Name: infection.Name, IsServerEpoch: true,
					ATK: infection.BaseATK + infection.AdditionalATK, ArtsATK: infection.BaseArtsATK + infection.AdditionalArtsATK,
					ArtsDEF: infection.ArtsDEF + infection.AdditionalArtsDEF, ConfusionDEF: infection.ConfusionDef + infection.AdditionalConfusionDef,
					DEF: infection.BaseDef + infection.AdditionalDEF, DEX: infection.DEX + infection.AdditionalDEX, HPRecoveryRate: infection.HPRecoveryRate + infection.AdditionalHPRecovery, INT: infection.INT + infection.AdditionalINT,
					MaxHP: infection.MaxHP + infection.AdditionalHP, ParalysisDEF: infection.ParalysisDef + infection.AdditionalParalysisDef, PoisonDEF: infection.ParalysisDef + infection.AdditionalPoisonDef, STR: infection.STR + infection.AdditionalSTR,
					Accuracy: infection.Accuracy + infection.AdditionalAccuracy, Dodge: infection.DodgeRate + infection.AdditionalDodgeRate, RunningSpeed: infection.MovSpeed + infection.AdditionalMovSpeed, EXPMultiplier: infection.ExpRate, DropMultiplier: infection.DropRate,
					CanExpire: true, MinArtsAtk: infection.MinArtsAtk, MaxArtsAtk: infection.MaxArtsAtk, EnhancedProbabilitiesBuff: infection.EnchancedProb, SyntheticCompositeBuff: infection.SyntheticComposite, AdvancedCompositeBuff: infection.AdvancedComposite, HyeolgongCost: infection.HyeolgongCost, PetExpMultiplier: infection.PetExpMultiplier}

				err := buff.Create()
				if err != nil {
					log.Print(err)
					continue
				}
			}
			data, _ := c.GetStats()
			if char.Socket != nil {
				char.Socket.Write(data)
			}
		}
	}
	FiveClans[temple.ID].ClanID = guild.ID
	FiveClans[temple.ID].Buff = int64(7200)
	FiveClans[temple.ID].Duration = int64(temple.Cooldown)
	FiveClans[temple.ID].Update()

	for _, ai := range AIsByMap[c.Socket.User.ConnectedServer][c.Map] {

		if ai.NPCpos.NPCID == temple.GateNpcID {
			ai.Faction = 3
		} else if ai.NPCpos.NPCID == temple.GuardNpcID {
			ai.Faction = 3
		} else if ai.NPCpos.NPCID == temple.StatueNpcID {
			ai.Faction = 3
		}

		c.OnSight.MobMutex.RLock()
		_, ok := c.OnSight.Mobs[ai.ID]
		c.OnSight.MobMutex.RUnlock()

		if ok {
			c.OnSight.MobMutex.Lock()
			delete(c.OnSight.Mobs, ai.ID)
			c.OnSight.MobMutex.Unlock()
		}

	}
	go time.AfterFunc(time.Hour, func() {
		for _, ai := range AIsByMap[1][233] {
			if ai.NPCpos.NPCID == temple.GateNpcID {
				ai.Faction = 0
			} else if ai.NPCpos.NPCID == temple.GuardNpcID {
				ai.Faction = 0
			} else if ai.NPCpos.NPCID == temple.StatueNpcID {
				ai.Faction = 0
			}
		}
	})
}

func (c *Character) VerifyAidKS() {
	//return
	if c.Map == 1 || c.Map == 236 || c.Map == 238 || c.Map == 240 {
		return
	}
	if c.AidMode {
		characters, err := c.GetNearbyCharacters()
		if err != nil {
			log.Println(err)
			return
		}
		ids := funk.Map(characters, func(c *Character) int {
			return c.ID
		}).([]int)

		party := FindParty(c)
		if c.Map != 236 && c.Map != 238 && c.Map != 240 {
			for _, id := range ids {
				onSightPlayer, err := FindCharacterByID(id)
				if err != nil {
					continue
				}
				if id == c.ID {
					continue
				}

				if party != nil {
					isparty := false
					for _, member := range party.Members {
						if onSightPlayer.ID == member.Character.ID {
							isparty = true
						}
					}
					if isparty {
						continue
					}
				}

				characterCoordinate := ConvertPointToLocation(c.Coordinate)
				enemyCoordinate := ConvertPointToLocation(onSightPlayer.Coordinate)
				distance := utils.CalculateDistance(characterCoordinate, enemyCoordinate)
				if distance < 35 && onSightPlayer.AidMode {
					teleport := c.Teleport(ConvertPointToLocation(c.AidStartingPosition))
					c.Socket.Write(teleport)
				}
			}
		}
	}
}

func (c *Character) AddBuff(infection *BuffInfection, duration int64) {

	if infection == nil || c == nil {
		return
	}
	expire := true
	if duration == 0 {
		expire = false
		duration = 10
	}
	buff, err := FindBuffByID(infection.ID, c.ID)
	if err != nil {
		return
	} else {
		if buff == nil {
			buff = &Buff{ID: infection.ID, CharacterID: c.ID, StartedAt: c.Epoch, Duration: duration, Name: infection.Name,
				ATK: infection.BaseATK + infection.AdditionalATK, ArtsATK: infection.BaseArtsATK + infection.AdditionalArtsATK,
				ArtsDEF: infection.ArtsDEF + infection.AdditionalArtsDEF, ConfusionDEF: infection.ConfusionDef + infection.AdditionalConfusionDef,
				DEF: infection.BaseDef + infection.AdditionalDEF, DEX: infection.DEX + infection.AdditionalDEX, HPRecoveryRate: infection.HPRecoveryRate + infection.AdditionalHPRecovery, INT: infection.INT + infection.AdditionalINT,
				MaxHP: infection.MaxHP + infection.AdditionalHP, ParalysisDEF: infection.ParalysisDef + infection.AdditionalParalysisDef, PoisonDEF: infection.ParalysisDef + infection.AdditionalPoisonDef, STR: infection.STR + infection.AdditionalSTR,
				Accuracy: infection.Accuracy + infection.AdditionalAccuracy, Dodge: infection.DodgeRate + infection.AdditionalDodgeRate, RunningSpeed: infection.MovSpeed + infection.AdditionalMovSpeed, EXPMultiplier: infection.ExpRate, DropMultiplier: infection.DropRate,
				CanExpire: expire, MinArtsAtk: infection.MinArtsAtk, MaxArtsAtk: infection.MaxArtsAtk, EnhancedProbabilitiesBuff: infection.EnchancedProb, SyntheticCompositeBuff: infection.SyntheticComposite, AdvancedCompositeBuff: infection.AdvancedComposite, HyeolgongCost: infection.HyeolgongCost, PetExpMultiplier: infection.PetExpMultiplier}

			if buff.ID >= 60001 && buff.ID <= 60025 {
				buff.StartedAt = GetServerEpoch()
				buff.IsServerEpoch = true
			}
			err := buff.Create()
			if err != nil {
				log.Print(err)
				return
			}
		} else {
			buff.StartedAt = c.Epoch
			buff.Update()
		}
	}
	data, _ := c.GetStats()
	c.Socket.Write(data)
}
func (c *Character) HandleItemsBuffs() {
	slots, err := c.InventorySlots()
	if err != nil {
		return
	}

	if slots[309] == nil || slots[309].SlotID == 0 {
		c.DeleteAura()
		return
	}

	ht := c.GetAppearingItemSlots()
	if ht[2] != 309 {
		c.DeleteAura()
		return
	}

	if slots[309].Buff != 0 {
		infection := BuffInfections[int(slots[309].Buff)]
		if infection != nil {
			c.AddBuff(infection, 99999)
		}
	}
}
func (c *Character) DeleteAura() {
	armorBuffs := []int{60026, 60027}
	for _, buff := range armorBuffs {
		buff, err := FindBuffByID(buff, c.ID)
		if err != nil {
			log.Print(err)
			return
		}
		if buff != nil {
			buff.Duration = 0
			buff.Update()
		}
	}
}
func RefreshYingYangKeys() error {
	query := `update characters SET ying_yang_tickets = 3`
	_, err := pgsql_DbMap.Exec(query)
	if err != nil {
		return err
	}

	characterMutex.RLock()
	allChars := funk.Values(characters).([]*Character)
	characterMutex.RUnlock()
	for _, c := range allChars {
		c.YingYangTicketsLeft = 3
	}

	return err
}

func (c *Character) GoDarkness() {
	s := c.Socket

	c.Level = 201
	c.Type += 10
	c.Exp = 544951059311

	skills := s.Skills
	skillSlots, err := skills.GetSkills()
	if err != nil {
		return
	}

	sd := utils.Packet{}

	for i := 0; i <= 7; i++ {
		if skillSlots.Slots[i].BookID != 0 {
			if i < 5 {
				if skillSlots.Slots[i].BookID != 0 {
					skillData, _ := c.RemoveSkill(byte(i), skillSlots.Slots[i].BookID)
					sd.Concat(skillData)
				}
			} else {
				skillIndex := byte(0)
				if i == 0 {
					skillIndex = 5
				} else if i == 1 {
					skillIndex = 6
				} else if i == 7 {
					skillIndex = 8
				}
				skillData, _ := c.RemovePassiveSkill(skillIndex, byte(i), skillSlots.Slots[i].BookID)
				sd.Concat(skillData)
			}
		}
	}

	spIndex := utils.SearchUInt64(SkillPoints, uint64(1))
	spIndex2 := utils.SearchUInt64(SkillPoints, uint64(c.Exp))
	skPts := spIndex2 - spIndex
	skPts += c.Level * int(c.Reborns) * 3
	skills.SkillPoints = skPts
	if c.Level > 100 {
		for i := 101; i <= c.Level; i++ {
			skills.SkillPoints += EXPs[int16(i)].SkillPoints
		}
	}

	c.Socket.Skills.Update()
	c.Class = 40
	c.Update()
	s.User.Update()

	ATARAXIA := utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x21, 0xE3, 0x55, 0xAA, 0xaa, 0x55, 0x0b, 0x00, 0x75, 0x00, 0x01, 0x00, 0x80, 0xa1, 0x43, 0x00, 0x00, 0x3d, 0x43, 0x55, 0xaa}
	resp := ATARAXIA
	resp[6] = byte(c.Type) // character type

	ANNOUNCEMENT := utils.Packet{0xAA, 0x55, 0x54, 0x00, 0x71, 0x14, 0x51, 0x55, 0xAA}
	msg := "At this moment I mark my name on list of Darklord Hero."
	announce := ANNOUNCEMENT
	index := 6
	announce[index] = byte(len(c.Name) + len(msg))
	index++
	announce.Insert([]byte("["+c.Name+"]"), index) // character name
	index += len(c.Name) + 2
	announce.Insert([]byte(msg), index) // character name
	announce.SetLength(int16(binary.Size(announce) - 6))
	p := nats.CastPacket{CastNear: false, Data: announce}
	p.Cast()

	statData, _ := c.GetStats()
	resp.Concat(statData)
	resp.Concat(sd)

	skillsData, err := s.Skills.GetSkillsData()
	if err != nil {
		return
	}
	rr, _, err := c.AddItem(&InventorySlot{ItemID: 17504477, Quantity: 1}, -1, false)
	if err != nil {
		return
	}
	s.Write(*rr)

	s.Write(skillsData)
	s.Write(resp)
}
func (c *Character) AddBoxToOpener(whereItem *InventorySlot) error {

	slots, _ := c.InventorySlots()

	slotID := int16(402)
	for ; slotID <= 414; slotID++ {
		slot := slots[slotID]
		if slot.ItemID == 0 {
			break
		}
	}
	toItem := slots[slotID]
	where := whereItem.SlotID

	if slots[slotID+20].ItemID == 0 {
		c.Socket.Write(messaging.SystemMessage(messaging.NO_OPENER_EXPANTION))
		return nil

	} else {
		if slots[slotID+20].Quantity >= whereItem.Quantity {
			whereItem.SlotID = slotID
			whereItem.CharacterID = null.IntFrom(int64(c.ID))
			*toItem = *whereItem
			*whereItem = *NewSlot()
			toItem.Update()
		} else {
			*toItem = *whereItem
			toItem.Quantity = slots[slotID+20].Quantity
			toItem.SlotID = slotID

			whereItem.Quantity -= slots[slotID+20].Quantity

			toItem.Insert()
			whereItem.Update()
		}
	}

	InventoryItems.Add(toItem.ID, toItem)

	c.Socket.Write(whereItem.GetData(where))

	return nil
}

func (c *Character) ShowBoxOpenerItems() []byte {

	rr := utils.Packet{}

	slots, _ := c.InventorySlots()

	slotID := int16(402)
	for ; slotID <= 414; slotID++ {
		var slot InventorySlot
		if slots[slotID].ItemID == 0 {
			slot = *slots[slotID+20]
		} else {
			slot = *slots[slotID]
		}

		data := slot.GetData(slotID - 402)
		data = data[6 : len(data)-2]

		resp := TRASH_LIST
		resp.Insert(data, 7) // item data
		rr.Concat(resp)
	}

	return rr
}

func (c *Character) Enchant(bookID int64, matsSlots []int16, matsIds []int64) ([]byte, error) {
	resp := utils.Packet{}
	bookSlotID, book, err := c.FindItemInInventory(nil, bookID)
	if err != nil || book == nil {
		return nil, err
	}

	enhancement := Enhancements[int(bookID)]
	if enhancement == nil {
		return ENCHANT_ERROR, nil
	}

	reqmats := []int64{enhancement.Material1, enhancement.Material2, enhancement.Material3}

	checked := 0
	for _, reqmat := range reqmats {
		if reqmat == 0 {
			checked++
			continue
		}
		for _, mat := range matsIds {
			if mat == 0 {
				continue
			}
			if reqmat == mat {
				mat = 0
				checked++
			}
		}

	}
	if checked != 3 {
		return nil, fmt.Errorf("Enchant: materialsAmount = %d", checked)
	}
	//purceed
	for _, mats := range matsSlots {
		data := c.DecrementItem(mats, 1)
		resp.Concat(*data)
	}
	data := c.DecrementItem(bookSlotID, 1)
	resp.Concat(*data)

	rand := utils.RandInt(0, 1000)
	if rand < int64(enhancement.Rate) {
		resp.Concat(ENCHANT_SUCCESS)
		additem, _, err := c.AddItem(&InventorySlot{ItemID: int64(enhancement.Result), Quantity: 1}, -1, false)
		if err != nil {
			return nil, err
		} else {
			resp.Concat(*additem)
		}
	} else {
		resp.Concat(ENCHANT_FAILED)
	}

	return resp, nil

}
func (c *Character) GetNumberOfAiAroundPlayer() int {

	var (
		// /distance = 7.0
		ids []int
	)

	user, err := FindUserByID(c.UserID)
	if err != nil {
		return 0
	} else if user == nil {
		return 0
	}

	candidates := AIsByMap[user.ConnectedServer][c.Map]
	filtered := funk.Filter(candidates, func(ai *AI) bool {

		return ai.TargetPlayerID == c.ID
	})

	for _, ai := range filtered.([]*AI) {
		ids = append(ids, ai.ID)
	}

	return len(ids)
}

func (c *Character) UnequipItemSlotID(slotID int16) error {

	c.AntiDupeMutex.Lock()
	defer c.AntiDupeMutex.Unlock()
	slot, err := c.FindFreeSlot()
	if err != nil {
		return err
	} else if slot == 0 {
		return nil
	}
	slots, err := c.InventorySlots()
	if err != nil {
		return err
	}
	info := slots[slotID]
	info.SlotID = slot

	if c.Socket != nil {
		c.Socket.Write(info.GetData(info.SlotID))
	}

	return nil
}

func (c *Character) CheckIn() {

	c.CheckInMutex.Lock()
	defer c.CheckInMutex.Unlock()

	if CheckIns[c.ID] == nil || time.Since(CheckIns[c.ID].LastCheckIn.Time.Add(time.Hour*time.Duration(24))) >= 0 {
		CheckIns[c.ID].TotalChecks++
		CheckIns[c.ID].LastCheckIn = null.NewTime(time.Now(), true)
		CheckIns[c.ID].Update()
		c.Socket.Write(messaging.InfoMessage("Sucessfully checked in!"))

		itemdata, _, err := c.AddItem(&InventorySlot{ItemID: 17502447, Quantity: 1}, -1, false)
		if err != nil {
			return
		}
		c.Socket.Write(*itemdata)
	} else {
		c.Socket.Write(messaging.InfoMessage("You have already checked in today!"))
	}

}

func (c *Character) ClaimCheckIn() ([]byte, error) {
	c.CheckInMutex.Lock()
	defer c.CheckInMutex.Unlock()

	if CheckIns[c.ID] == nil {
		return nil, nil
	}
	resp := utils.Packet{}

	for i := 1; i <= 31; i++ {
		checkin_rewards := CheckinRewards[i]
		if checkin_rewards != nil {
			if CheckIns[c.ID].TotalChecks < checkin_rewards.Day {
				c.Socket.Write(messaging.InfoMessage(fmt.Sprintf("You need to check in %d times to claim this reward!, You have %d days.", checkin_rewards.Day, CheckIns[c.ID].TotalChecks)))
				break
			}
			if checkin_rewards.Day <= CheckIns[c.ID].LastClaimed {
				c.Socket.Write(messaging.InfoMessage(fmt.Sprintf("You have already claimed this reward! Next reward in %d days.", int(math.Abs(float64(checkin_rewards.Day-CheckIns[c.ID].LastClaimed))))))
				break
			}

			_, err := c.FindFreeSlots(3)
			if err != nil {
				return messaging.SystemMessage(messaging.DISMINTLE_NOT_ENOUGH_SPACE), nil
			}

			for index, item := range checkin_rewards.ItemIdsArr {
				quantity := checkin_rewards.QuantitysArr[index]
				item := &InventorySlot{ItemID: item, Quantity: uint(quantity)}
				info, ok := GetItemInfo(item.ItemID)
				if !ok {
					continue
				}
				if info.GetType() == PET_TYPE {
					petInfo := Pets[item.ItemID]
					expInfo := PetExps[petInfo.Level-1]

					item.Pet = &PetSlot{
						Fullness: 100, Loyalty: 100,
						Exp:   float64(expInfo.ReqExpEvo1),
						HP:    petInfo.BaseHP,
						Level: byte(petInfo.Level),
						Name:  petInfo.Name,
						CHI:   petInfo.BaseChi}
				}
				itemdata, _, err := c.AddItem(item, -1, false)
				if err != nil {
					return nil, err
				}
				resp.Concat(*itemdata)
			}

			CheckIns[c.ID].LastClaimed = checkin_rewards.Day
			CheckIns[c.ID].Update()
			resp.Concat(messaging.InfoMessage(fmt.Sprintf("Sucessfully claimed checkin reward! Day: %d", checkin_rewards.Day)))

			return resp, nil
		}

	}

	return resp, nil
}
func (c *Character) BuyItem(quantity int64, itemID int64, slotID int16, shopID int) ([]byte, error) {

	c.AntiDupeMutex.Lock()
	defer c.AntiDupeMutex.Unlock()
	//item_buyed := utils.Packet{0xaa, 0x55, 0x3c, 0x00, 0x58, 0x01, 0x0a, 0x00, 0x15, 0x64, 0x64, 0x00, 0x00, 0x00, 0x00, 0x00, 0x20, 0x1c, 0x00, 0x00, 0x55, 0xaa}
	item_buyed := utils.Packet{0xaa, 0x55, 0x3c, 0x00, 0x58, 0x01, 0x0a, 0x00, 0x15, 0x64, 0x64, 0x00, 0x20, 0x1c, 0x00, 0x00, 0x55, 0xaa}

	itemindex := 8
	slots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}
	if quantity > 9999 {
		quantity = 9999
	}

	info, ok := GetItemInfo(itemID)
	if !ok {
		return nil, errors.New("BuyItem:Item not found")
	}

	if info.SpecialItem != 0 {
		canChange := true
		reqCoinCount := uint(info.BuyPrice) * uint(quantity)
		slotIDitem, _, _ := c.FindItemInInventory(nil, info.SpecialItem)
		slots, err := c.InventorySlots()
		if shopID == 27 { // gold coin shop
			if err != nil {
				return nil, err
			}
			if itemID == 15900001 {
				quantity *= 50
			}
			if itemID == 17100004 {
				quantity *= 40
			}
			if itemID == 17100005 {
				quantity *= 40
			}
			if itemID == 100080180 {
				quantity *= 1000
			}

		}
		if itemID == 10810001 || itemID == 10810002 || itemID == 10810003 || itemID == 10810004 {
			quantity *= 100
		}
		if info.Type == 55 { //
			quantity = info.SellPrice
		}
		items := slots[slotIDitem]

		if info.TimerType > 0 {
			quantity = int64(info.Timer)
		}
		specialiteminfo, ok := GetItemInfo(info.SpecialItem)
		if !ok {
			return nil, errors.New("BuyItem: Item not found")
		}
		if specialiteminfo.TimerType > 0 && items.Quantity != uint(specialiteminfo.Timer) {
			canChange = false
			return NOT_ENOUGH_GOLD, nil
		}
		if items.Quantity < reqCoinCount {
			canChange = false
			return NOT_ENOUGH_GOLD, nil
		}
		if items.Activated || items.InUse {
			canChange = false
			return NOT_ENOUGH_GOLD, nil
		}

		if canChange {
			item := &InventorySlot{ItemID: itemID, Quantity: uint(quantity)}
			if info.GetType() == PET_TYPE {
				petInfo := Pets[item.ItemID]
				expInfo := PetExps[petInfo.Level-1]

				item.Pet = &PetSlot{
					Fullness: 100, Loyalty: 100,
					Exp:   float64(expInfo.ReqExpEvo1),
					HP:    petInfo.BaseHP,
					Level: byte(petInfo.Level),
					Name:  petInfo.Name,
					CHI:   petInfo.BaseChi}
			}
			resp, _, err := c.AddItem(item, slotID, false)
			if err != nil {
				return nil, err
			} else if resp == nil {
				return nil, nil
			}

			if info.TimerType > 0 {
				itemData, _ := c.RemoveItem(slotIDitem)
				resp.Concat(itemData)
			} else {
				itemData := c.DecrementItem(slotIDitem, reqCoinCount)
				resp.Concat(*itemData)
			}

			item_buyed.Insert(utils.IntToBytes(uint64(itemID), 4, true), itemindex)
			itemindex += 8
			item_buyed.Insert(utils.IntToBytes(uint64(slotID), 2, true), itemindex)
			itemindex += 2
			item_buyed.Insert([]byte{0x00, 0x00, 0x00, 0x00}, itemindex) //PET HP
			item_buyed.Insert([]byte{0x00, 0x00, 0x00, 0x00}, itemindex) //PET EXP
			itemindex += 8
			item_buyed.Insert([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, itemindex)
			itemindex += 26
			item_buyed.Insert(utils.IntToBytes(uint64(c.Gold), 8, true), itemindex)
			item_buyed.SetLength(int16(binary.Size(item_buyed) - 6))
			resp.Concat(item_buyed)

			text := "Char: " + c.Name + "(" + c.UserID + ") bought: " + info.Name
			utils.NewLog("logs/regular_shops.txt", text)

			return *resp, nil
		}
		return nil, nil
	}
	cost := uint64(info.BuyPrice) * uint64(quantity)
	stat := c.Socket.Stats
	if stat == nil {
		return nil, nil
	}
	discount := uint64((float64(cost) * stat.Npc_gold_multiplier) - float64(cost))
	cost = cost - discount
	if slots[slotID].ItemID == 0 && cost <= c.Gold && quantity > 0 && info.SpecialItem == 0 { // slot is empty, player can afford and quantity is positive
		c.LootGold(-cost)
		if info.TimerType > 0 {
			quantity = int64(info.Timer)
		}
		item := &InventorySlot{ItemID: itemID, Quantity: uint(quantity)}

		if info.GetType() == PET_TYPE {
			petInfo := Pets[item.ItemID]
			expInfo := PetExps[petInfo.Level-1]

			item.Pet = &PetSlot{
				Fullness: 100, Loyalty: 100,
				Exp:   float64(expInfo.ReqExpEvo1),
				HP:    petInfo.BaseHP,
				Level: byte(petInfo.Level),
				Name:  "",
				CHI:   petInfo.BaseChi}
		}

		resp, _, err := c.AddItem(item, slotID, false)
		if err != nil {
			return nil, err
		} else if resp == nil {
			return nil, nil
		}
		item_buyed.Insert(utils.IntToBytes(uint64(itemID), 4, true), itemindex)
		itemindex += 8
		item_buyed.Insert(utils.IntToBytes(uint64(slotID), 2, true), itemindex)
		itemindex += 2
		item_buyed.Insert([]byte{0x00, 0x00, 0x00, 0x00}, itemindex) //PET HP
		item_buyed.Insert([]byte{0x00, 0x00, 0x00, 0x00}, itemindex) //PET EXP
		itemindex += 8
		item_buyed.Insert([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, itemindex)
		itemindex += 26
		item_buyed.Insert(utils.IntToBytes(uint64(c.Gold), 8, true), itemindex)
		item_buyed.SetLength(int16(binary.Size(item_buyed) - 6))

		resp.Concat(item_buyed)

		text := "Char: " + c.Name + "(" + c.UserID + ") bought: " + info.Name
		utils.NewLog("logs/regular_shops.txt", text)
		return *resp, nil
	}

	return nil, nil
}

func (c *Character) SellItem(itemID, slot, quantity int, unitPrice uint64) ([]byte, error) {
	c.AntiDupeMutex.Lock()
	defer c.AntiDupeMutex.Unlock()

	amount := (unitPrice * uint64(quantity))
	stat := c.Socket.Stats
	if stat == nil {
		return nil, nil
	}
	amount = uint64(float64(amount) * stat.Npc_gold_multiplier)
	amount = uint64(float64(amount))
	if c == nil || c.Socket == nil {
		return nil, nil
	}

	if c.TradeID != "" {
		return messaging.SystemMessage(10053), nil //Cannot do that while trading
	}

	repurchese, err := c.AddRepurchaseItem(int(slot))
	if err != nil {
		return nil, err
	}

	_, err = c.RemoveItem(int16(slot))
	if err != nil {
		return nil, err
	}
	c.LootGold(amount)

	resp := SELL_ITEM
	resp.Insert(utils.IntToBytes(uint64(itemID), 4, true), 8)  // item id
	resp.Insert(utils.IntToBytes(uint64(slot), 2, true), 12)   // slot id
	resp.Insert(utils.IntToBytes(uint64(c.Gold), 8, true), 14) // character gold

	resp.Concat(repurchese)

	return resp, nil

}
func (c *Character) ArrangeBankItems() ([]byte, error) {

	if c.ArrangeCooldown > 0 {
		c.Socket.Write(messaging.InfoMessage(fmt.Sprintf("Cooldown: %dsec", c.ArrangeCooldown)))
		time.Sleep(time.Duration(c.ArrangeCooldown) * time.Second)
	}

	slots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}

	slots = slots[0x43:0x133]
	resp := utils.Packet{}
	for page := 0; page < 4; page++ {

		newSlots := make([]InventorySlot, 60)
		for i := 0; i < 60; i++ {
			index := page*60 + i
			newSlots[i] = *slots[index]
			if newSlots[i].ItemID == 0 {
				newSlots[i].ItemID = math.MaxInt64
			}
			newSlots[i].RFU = int64(index + 0x43)
		}

		sort.SliceStable(newSlots, func(i, j int) bool {
			return newSlots[i].ItemID < newSlots[j].ItemID

		})

		for i := 0; i < 60; i++ {
			slot := &newSlots[i]
			r, r2 := ARRANGE_BANK_ITEM, utils.Packet{}

			if slot.ItemID == math.MaxInt64 {
				slot.ItemID = 0
			}

			slot.SlotID = int16(page*60 + i + 0x43)
			r.Insert(utils.IntToBytes(uint64(slot.ItemID), 4, true), 6) // item id

			info, _ := GetItemInfo(slot.ItemID)
			if info != nil && slot.Activated { // using state
				if info.TimerType == 1 {
					r[10] = 3
				} else if info.TimerType == 3 {
					r[10] = 5
					r2 = GREEN_ITEM_COUNT
					r2.Insert(utils.IntToBytes(uint64(slot.SlotID), 2, true), 8)    // slot id
					r2.Insert(utils.IntToBytes(uint64(slot.Quantity), 4, true), 10) // item quantity
				}
			} else {
				r[10] = 0
			}

			if slot.ItemID == 0 {
				r[11] = 0
			} else if slot.Plus > 0 || slot.SocketCount > 0 {
				r[11] = 0xA2
			}

			r.Insert(utils.IntToBytes(uint64(slot.Quantity), 2, true), 12) // item quantity
			r.Insert(utils.IntToBytes(uint64(slot.SlotID), 2, true), 14)   // slot id

			if slot.Plus > 0 || slot.SocketCount > 0 {
				r.Insert(slot.GetUpgrades(), 16) // slot upgrades
				r[31] = byte(slot.SocketCount)   // socket count
				r.Insert(slot.GetSockets(), 32)  // slot sockets
				r.SetLength(0x4D)
			}

			if i == 60 {
				r[47] = 1
			}
			r[48] = byte(page)

			r.Insert(utils.IntToBytes(uint64(slot.RFU.(int64)), 2, true), 49) // pre slot id

			if info != nil && info.GetType() == PET_TYPE {
				r2.Concat(slot.GetData(int16(slot.SlotID)))
			}

			slot.Update()
			resp.Concat(r)
			resp.Concat(r2)
			resp.Concat(slot.GetData(slot.SlotID))
		}

		for i := 0; i < 60; i++ {
			slotID := page*60 + i
			newSlots[i].RFU = 0
			*slots[slotID] = newSlots[i]
		}
	}

	c.ArrangeCooldown = 4
	return resp, nil
}
func (c *Character) ArrangeInventoryItems() ([]byte, error) {

	if c.ArrangeCooldown > 0 {
		c.Socket.Write(messaging.InfoMessage(fmt.Sprintf("Cooldown: %dsec", c.ArrangeCooldown)))
		time.Sleep(time.Duration(c.ArrangeCooldown) * time.Second)
	}

	slots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}
	newSlots := make([]InventorySlot, 56)
	for i := 0; i < 56; i++ {
		slotID := i + 0x0B
		newSlots[i] = *slots[slotID]
		if newSlots[i].ItemID == 0 {
			newSlots[i].ItemID = math.MaxInt64
		}
		newSlots[i].RFU = int64(slotID)
	}

	sort.SliceStable(newSlots, func(i, j int) bool {
		return newSlots[i].ItemID < newSlots[j].ItemID
	})

	resp := utils.Packet{}
	for i := 0; i < 56; i++ { // first page
		slot := &newSlots[i]
		r, r2 := ARRANGE_ITEM, utils.Packet{}

		if slot.ItemID == math.MaxInt64 {
			slot.ItemID = 0
		}

		slot.SlotID = int16(i + 0x0B)
		r.Insert(utils.IntToBytes(uint64(slot.ItemID), 4, true), 6) // item id

		info, _ := GetItemInfo(slot.ItemID)
		if info != nil && slot.Activated { // using state
			if info.TimerType == 1 {
				r[10] = 3
			} else if info.TimerType == 3 {
				r[10] = 5
				r2 = GREEN_ITEM_COUNT
				r2.Insert(utils.IntToBytes(uint64(slot.SlotID), 2, true), 8)    // slot id
				r2.Insert(utils.IntToBytes(uint64(slot.Quantity), 4, true), 10) // item quantity
			}
		} else {
			r[10] = 0
		}

		if slot.ItemID == 0 {
			r[11] = 0
		} else if slot.Plus > 0 {
			r[11] = 0xA2
		}

		r.Insert(utils.IntToBytes(uint64(slot.Quantity), 2, true), 12) // item quantity
		r.Insert(utils.IntToBytes(uint64(slot.SlotID), 2, true), 14)   // slot id
		r.Insert(slot.GetUpgrades(), 16)                               // slot upgrades
		r[31] = byte(slot.SocketCount)                                 // socket count
		r.Insert(slot.GetSockets(), 32)                                // slot sockets

		if i == 55 {
			r[50] = 1
		}

		r.Insert(utils.IntToBytes(uint64(slot.RFU.(int64)), 2, true), 52) // pre slot id

		if info != nil && info.GetType() == PET_TYPE {
			r2.Concat(slot.GetData(int16(slot.SlotID)))
		}

		slot.Update()
		resp.Concat(r)
		resp.Concat(r2)
		resp.Concat(slot.GetData(slot.SlotID, c.ID))
	}

	for i := 0; i < 56; i++ {
		slotID := i + 0x0B
		newSlots[i].RFU = 0
		*slots[slotID] = newSlots[i]
	}

	newSlots = make([]InventorySlot, 56)
	for i := 0; i < 56; i++ {
		slotID := i + 0x0155
		newSlots[i] = *slots[slotID]
		if newSlots[i].ItemID == 0 {
			newSlots[i].ItemID = math.MaxInt64
		}
		newSlots[i].RFU = int64(slotID)
	}

	sort.SliceStable(newSlots, func(i, j int) bool {
		return newSlots[i].ItemID < newSlots[j].ItemID
	})

	for i := 0; i < 56; i++ { // second page
		slot := &newSlots[i]
		r, r2 := ARRANGE_ITEM, utils.Packet{}

		if slot.ItemID == math.MaxInt64 {
			slot.ItemID = 0
		}

		slot.SlotID = int16(i + 0x0155)
		r.Insert(utils.IntToBytes(uint64(slot.ItemID), 4, true), 6) // item id

		info, _ := GetItemInfo(slot.ItemID)
		if info != nil && slot.Activated { // using state
			if info.TimerType == 1 {
				r[10] = 3
			} else if info.TimerType == 3 {
				r[10] = 5
				r2 = GREEN_ITEM_COUNT
				r2.Insert(utils.IntToBytes(uint64(slot.SlotID), 2, true), 8)    // slot id
				r2.Insert(utils.IntToBytes(uint64(slot.Quantity), 4, true), 10) // item quantity
			}
		} else {
			r[10] = 0
		}

		if slot.ItemID == 0 {
			r[11] = 0
		} else if slot.Plus > 0 {
			r[11] = 0xA2
		}

		r.Insert(utils.IntToBytes(uint64(slot.Quantity), 2, true), 12) // item quantity
		r.Insert(utils.IntToBytes(uint64(slot.SlotID), 2, true), 14)   // slot id
		r.Insert(slot.GetUpgrades(), 16)                               // slot upgrades
		r[31] = byte(slot.SocketCount)                                 // socket count
		r.Insert(slot.GetSockets(), 32)                                // slot sockets

		if i == 55 {
			r[50] = 1
		}

		r[51] = 1
		r.Insert(utils.IntToBytes(uint64(slot.RFU.(int64)), 2, true), 52) // pre slot id

		if info != nil && info.GetType() == PET_TYPE {
			r2.Concat(slot.GetData(int16(slot.SlotID)))
		}

		slot.Update()
		resp.Concat(r)
		resp.Concat(r2)
		resp.Concat(slot.GetData(slot.SlotID, c.ID))
	}

	for i := 0; i < 56; i++ {
		slotID := i + 0x0155
		newSlots[i].RFU = 0
		*slots[slotID] = newSlots[i]
	}

	c.ArrangeCooldown = 4
	return resp, nil

}

func (c *Character) GetNearbyBabyPetsIDs() ([]int, error) {

	var (
		distance = 50.0
		ids      []int
	)

	user, err := FindUserByID(c.UserID)
	if err != nil {
		return nil, err
	} else if user == nil {
		return nil, nil
	}

	if BabyPetsByMap[user.ConnectedServer] == nil {

		return ids, nil
	}

	candidates := BabyPetsByMap[user.ConnectedServer][c.Map]
	filtered := funk.Filter(candidates, func(bby *BabyPet) bool {

		characterCoordinate := ConvertPointToLocation(c.Coordinate)
		aiCoordinate := ConvertPointToLocation(bby.Coordinate)

		return utils.CalculateDistance(characterCoordinate, aiCoordinate) <= distance
	})

	for _, ai := range filtered.([]*BabyPet) {
		ids = append(ids, ai.ID)
	}

	return ids, nil
}
func (c *Character) HousingDetails() {

	if c.House == nil {
		return
	}

	pp := utils.Packet{0xaa, 0x55, 0x2d, 0x00, 0xac, 0x06, 0x0a, 0x00, 0xbe, 0x18, 0x01, 0x00, 0x90, 0x01, 0x00, 0x00, 0x00, 0x13, 0x32, 0x30, 0x32, 0x32, 0x2d, 0x30, 0x34, 0x2d, 0x32, 0x35,
		0x20, 0x31, 0x36, 0x3a, 0x31, 0x31, 0x3a, 0x35, 0x39, 0x00, 0x80, 0x81, 0x43, 0x15, 0x21, 0x18, 0x40, 0x00, 0x00, 0x35, 0x43, 0x55, 0xaa}

	formatdate := c.House.ExpirationDate.Time.Format("2006-01-02 15:04:05")

	pp.Overwrite(utils.IntToBytes(uint64(c.House.PseudoID), 4, true), 8)

	pp.Overwrite(utils.IntToBytes(uint64(c.House.HouseID), 4, true), 12)

	index := 16
	pp.Overwrite(utils.IntToBytes(uint64(len(formatdate)), 2, false), index)
	index += 2
	pp.Overwrite([]byte(formatdate), index)
	index += len(formatdate)

	pp.Overwrite(utils.FloatToBytes(c.House.PosX, 4, true), index)
	index += 4
	pp.Overwrite(utils.FloatToBytes(c.House.PosZ, 4, true), index)
	index += 4
	pp.Overwrite(utils.FloatToBytes(c.House.PosY, 4, true), index)
	index += 4

	pp.Overwrite(utils.IntToBytes(uint64(c.House.IsPublic), 1, true), 16) //ispublic

	c.Socket.Write(pp)

	c.CalculateMaxRelax()
	ss := RELAXATION_INFO
	ss.Overwrite(utils.IntToBytes(uint64(c.Relaxation), 4, true), 8)           //relaxation
	ss.Overwrite(utils.IntToBytes(uint64(c.House.MaxRelaxation), 4, true), 12) //max relaxation

	c.Socket.Write(ss)

}

func (c *Character) GetHouseItems() []*HousingItem {
	var itms []*HousingItem

	for _, house := range HousingItems {
		if house == nil {
			continue
		}
		if house.OwnerID == c.ID {
			itms = append(itms, house)
		}
	}

	return itms
}

func (c *Character) CalculateMaxRelax() {
	hitms := c.GetHouseItems()
	if c.House == nil {
		return
	}
	c.House.MaxRelaxation = 0
	for _, item := range hitms {
		info, ok := HouseItemsInfos[item.HouseID]
		if !ok {
			return
		}
		c.House.MaxRelaxation += info.Relaxetion
	}
	c.House.Update()
}
func (c *Character) OpenAdventurerMenu() []byte {
	resp := utils.Packet{0xaa, 0x55, 0xff, 0x00, 0xd7, 0x01, 0x55, 0xaa}
	index := 6
	for i := 0; i < 12; i++ {
		total_adv := 0
		hired := 0
		formatdate := ""
		for _, adv := range Adventurers {
			if adv.Index == i+1 && adv.CharID == c.ID {
				total_adv = adv.TotalAdventures
				hired = adv.Status
				formatdate = adv.FinishAt.Time.Format("2006-01-02 15:04:05") // expires at
				break
			}
		}
		resp.Insert(utils.IntToBytes(uint64(i+1), 4, true), index)
		index += 4
		resp.Insert(utils.IntToBytes(uint64(total_adv), 4, true), index) //level
		index += 4
		resp.Insert(utils.IntToBytes(uint64(hired), 1, true), index) //hired
		index += 1

		resp.Insert(utils.IntToBytes(uint64(len(formatdate)), 1, false), index)
		index++
		resp.Insert([]byte(formatdate), index)
		index += len(formatdate)
	}
	resp.SetLength(int16(binary.Size(resp) - 6))

	return resp
}
func (c *Character) CountPlayerOnlineHours() {

	time.AfterFunc(time.Minute*time.Duration(60), func() {
		if c == nil {
			return
		} else if !c.IsOnline {
			return
		}

		c.OnlineHours++
		if c.OnlineHours%6 == 0 {
			itemdata, slotid, err := c.AddItem(&InventorySlot{ItemID: 13003349, Quantity: 300}, -1, false)
			if err != nil {
				log.Print(err)
			} else if slotid == -1 || itemdata == nil {
				c.Socket.Write(messaging.InfoMessage("You have no space in your inventory to add reward."))
				return
			}
			resp := *itemdata
			resp.Concat(messaging.InfoMessage("You have received 6 hours spooky reward."))
			c.Socket.Write(resp)
			c.OnlineHours = 0

		}
		c.Update()
		c.CountPlayerOnlineHours()
	})

}

func (c *Character) GetNearbyHousesIDs() ([]int, error) {

	var (
		distance = 50.0
		ids      []int
	)

	user, err := FindUserByID(c.UserID)
	if err != nil {
		return nil, err
	} else if user == nil {
		return nil, nil
	}
	candidates := HousingItemsByMap[user.ConnectedServer][c.Map]
	filtered := funk.Filter(candidates, func(pos *HousingItem) bool {

		characterCoordinate := ConvertPointToLocation(c.Coordinate)

		itemLocation := ConvertPointToCoordinate(pos.PosX, pos.PosY)
		itemCoordinate := ConvertPointToLocation(itemLocation)

		return c.Map == pos.MapID && utils.CalculateDistance(characterCoordinate, itemCoordinate) <= distance
	})

	for _, pos := range filtered.([]*HousingItem) {
		ids = append(ids, pos.ID)
	}

	return ids, nil
}
func (c *Character) ClearInventory() {

	for i := 0; i <= 56; i++ {
		_, err := c.RemoveItem(int16(i + 11))
		if err != nil {
			continue
		}
		_, err = c.RemoveItem(int16(i + 341))
		if err != nil {
			continue
		}
	}
}

func (c *Character) TrickOrTreat() []byte {
	if c.ClanGoldDonation == 1 {
		return messaging.InfoMessage("You have already received your reward!")
	}
	if c.Level < 50 {
		return messaging.InfoMessage("You need to be at least level 50 to receive your reward!")
	}

	itemid := int64(13000148)
	resp := utils.Packet{}
	resp = messaging.InfoMessage("You received a spookie reward!")
	random := rand.Intn(3)
	if random == 0 {
		itemid = 13000148
	} else if random == 1 {
		itemid = 13000149
	} else if random == 2 {
		itemid = 13000150
	} else if random == 3 {
		itemid = 13000151
	}

	itemData, _, _ := c.AddItem(&InventorySlot{ItemID: itemid, Quantity: 20160}, -1, false)
	resp.Concat(*itemData)

	itemData, _, _ = c.AddItem(&InventorySlot{ItemID: 13000293, Quantity: 43800}, -1, false)
	resp.Concat(*itemData)

	c.ClanGoldDonation = 1
	c.Update()
	return resp
}

func (c *Character) LootGold(amount uint64) []byte {
	c.GoldMutex.Lock()
	defer c.GoldMutex.Unlock()

	if amount > c.Gold {
		resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x59, 0x08, 0xF9, 0x03, 0x55, 0xAA} // not enough gold
		return resp
	}
	c.Gold += amount
	/*	if c.Gold < 0 {
		c.Gold = 0
	}*/

	go c.Update()

	resp := GOLD_LOOTED
	resp.Insert(utils.IntToBytes(uint64(c.Gold), 8, true), 9) // character gold

	return resp
}

func (c *Character) SubtractGold(amount uint64) bool {

	c.GoldMutex.Lock()
	defer c.GoldMutex.Unlock()

	resp := utils.Packet{}
	if c.Gold < amount {
		resp.Concat(messaging.SystemMessage(messaging.INSUFFICIENT_GOLD))
		c.Socket.Write(resp)
		return false
	}
	c.Gold -= amount
	go c.Update()
	c.Socket.Write(c.GetGold())
	return true
}

func (c *Character) StrengthGrade(marble1slot int64, marble2slot int64, marble3slot int64, hammerslot int64) ([]byte, error) {
	//azure
	slots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}
	marble1 := slots[marble1slot]
	marble2 := slots[marble2slot]
	marble3 := slots[marble3slot]
	hammer := slots[hammerslot]
	if marble1.ItemID != marble2.ItemID && marble2.ItemID != marble3.ItemID {
		return nil, nil
	}
	if marble1 == nil || marble2 == nil || marble3 == nil {
		return nil, nil
	}
	luck := 1.0
	if hammer.ItemID == 15700053 { //Artisan's Hammer
		luck = 2.0
	}
	if hammer.ItemID == 15700068 { //Artisan's High Class Hammer
		luck = 3.0
	}
	if hammer.ItemID == 17502628 { //Premium Artisan's Hammer
		luck = 4.0
	}

	resp := &utils.Packet{}
	resp.Concat(*c.DecrementItem(int16(marble1.SlotID), 1))

	seed := int(utils.RandInt(0, 1000))
	if marble2.ItemID >= 17300449 && marble2.ItemID <= 17300478 { //Tree Normal marble
		if float64(seed) < float64(200)*luck { // Success
			resp, _, err = c.AddItem(&InventorySlot{ItemID: int64(17402802), Quantity: 1}, int16(marble1slot), false)
			if err != nil {
				return nil, err
			} else if resp == nil {
				return nil, nil
			}
			resp.Concat(PRODUCTION_SUCCESS)
		} else { // Faileds
			resp.Concat(PRODUCTION_FAILED)
		}
	}
	if marble2.ItemID >= 17300479 && marble2.ItemID <= 17300508 { //Tree Advanced marble
		if float64(seed) < float64(185)*luck { // Success
			resp, _, err = c.AddItem(&InventorySlot{ItemID: int64(17402803), Quantity: 1}, int16(marble1slot), false)
			if err != nil {
				return nil, err
			} else if resp == nil {
				return nil, nil
			}
			resp.Concat(PRODUCTION_SUCCESS)
		} else { // Faileds
			resp.Concat(PRODUCTION_FAILED)
		}
	}
	if marble2.ItemID >= 17300509 && marble2.ItemID <= 17300538 { //Tree Rare marble
		if float64(seed) < float64(175)*luck { // Success
			resp, _, err = c.AddItem(&InventorySlot{ItemID: int64(17402804), Quantity: 1}, int16(marble1slot), false)
			if err != nil {
				return nil, err
			} else if resp == nil {
				return nil, nil
			}
			resp.Concat(PRODUCTION_SUCCESS)
		} else { // Faileds
			resp.Concat(PRODUCTION_FAILED)
		}
	}
	if marble2.ItemID >= 17300539 && marble2.ItemID <= 17300568 { //Tree Legend marble
		if float64(seed) < float64(100)*luck { // Success
			resp, _, err = c.AddItem(&InventorySlot{ItemID: int64(17402805), Quantity: 1}, int16(marble1slot), false)
			if err != nil {
				return nil, err
			} else if resp == nil {
				return nil, nil
			}
			resp.Concat(PRODUCTION_SUCCESS)
		} else { // Faileds
			resp.Concat(PRODUCTION_FAILED)
		}
	}
	if marble2.ItemID >= 17300599 && marble2.ItemID <= 17300628 { //Fire Normal marble
		if float64(seed) < float64(200)*luck { // Success
			resp, _, err = c.AddItem(&InventorySlot{ItemID: int64(17402806), Quantity: 1}, int16(marble1slot), false)
			if err != nil {
				return nil, err
			} else if resp == nil {
				return nil, nil
			}
			resp.Concat(PRODUCTION_SUCCESS)
		} else { // Faileds
			resp.Concat(PRODUCTION_FAILED)
		}
	}
	if marble2.ItemID >= 17300629 && marble2.ItemID <= 17300658 { //Fire Advanced marble
		if float64(seed) < float64(185)*luck { // Success
			resp, _, err = c.AddItem(&InventorySlot{ItemID: int64(17402807), Quantity: 1}, int16(marble1slot), false)
			if err != nil {
				return nil, err
			} else if resp == nil {
				return nil, nil
			}
			resp.Concat(PRODUCTION_SUCCESS)
		} else { // Faileds
			resp.Concat(PRODUCTION_FAILED)
		}
	}
	if marble2.ItemID >= 17300659 && marble2.ItemID <= 17300688 { //Fire Rare marble
		if float64(seed) < float64(175)*luck { // Success
			resp, _, err = c.AddItem(&InventorySlot{ItemID: int64(17402808), Quantity: 1}, int16(marble1slot), false)
			if err != nil {
				return nil, err
			} else if resp == nil {
				return nil, nil
			}
			resp.Concat(PRODUCTION_SUCCESS)
		} else { // Faileds
			resp.Concat(PRODUCTION_FAILED)
		}
	}
	if marble2.ItemID >= 17300689 && marble2.ItemID <= 17300718 { //Fire Legend marble
		if float64(seed) < float64(100)*luck { // Success
			resp, _, err = c.AddItem(&InventorySlot{ItemID: int64(17402809), Quantity: 1}, int16(marble1slot), false)
			if err != nil {
				return nil, err
			} else if resp == nil {
				return nil, nil
			}
			resp.Concat(PRODUCTION_SUCCESS)
		} else { // Faileds
			resp.Concat(PRODUCTION_FAILED)
		}
	}
	if marble2.ItemID >= 17300749 && marble2.ItemID <= 17300778 { //Earth Normal marble
		if float64(seed) < float64(200)*luck { // Success
			resp, _, err = c.AddItem(&InventorySlot{ItemID: int64(17402810), Quantity: 1}, int16(marble1slot), false)
			if err != nil {
				return nil, err
			} else if resp == nil {
				return nil, nil
			}
			resp.Concat(PRODUCTION_SUCCESS)
		} else { // Faileds
			resp.Concat(PRODUCTION_FAILED)
		}
	}
	if marble2.ItemID >= 17300779 && marble2.ItemID <= 17300808 { //Earth Advanced marble
		if float64(seed) < float64(185)*luck { // Success
			resp, _, err = c.AddItem(&InventorySlot{ItemID: int64(17402811), Quantity: 1}, int16(marble1slot), false)
			if err != nil {
				return nil, err
			} else if resp == nil {
				return nil, nil
			}
			resp.Concat(PRODUCTION_SUCCESS)
		} else { // Faileds
			resp.Concat(PRODUCTION_FAILED)
		}
	}
	if marble2.ItemID >= 17300809 && marble2.ItemID <= 17300838 { //Earth Rare marble
		if float64(seed) < float64(175)*luck { // Success
			resp, _, err = c.AddItem(&InventorySlot{ItemID: int64(17402812), Quantity: 1}, int16(marble1slot), false)
			if err != nil {
				return nil, err
			} else if resp == nil {
				return nil, nil
			}
			resp.Concat(PRODUCTION_SUCCESS)
		} else { // Faileds
			resp.Concat(PRODUCTION_FAILED)
		}
	}
	if marble2.ItemID >= 17300839 && marble2.ItemID <= 17300868 { //Earth Legend marble
		if float64(seed) < float64(100)*luck { // Success
			resp, _, err = c.AddItem(&InventorySlot{ItemID: int64(17402813), Quantity: 1}, int16(marble1slot), false)
			if err != nil {
				return nil, err
			} else if resp == nil {
				return nil, nil
			}
			resp.Concat(PRODUCTION_SUCCESS)
		} else { // Faileds
			resp.Concat(PRODUCTION_FAILED)
		}
	}
	if marble2.ItemID >= 17300899 && marble2.ItemID <= 17300928 { //Steel Normal marble
		if float64(seed) < float64(200)*luck { // Success
			resp, _, err = c.AddItem(&InventorySlot{ItemID: int64(17402814), Quantity: 1}, int16(marble1slot), false)
			if err != nil {
				return nil, err
			} else if resp == nil {
				return nil, nil
			}
			resp.Concat(PRODUCTION_SUCCESS)
		} else { // Faileds
			resp.Concat(PRODUCTION_FAILED)
		}
	}
	if marble2.ItemID >= 17300929 && marble2.ItemID <= 17300958 { //Steel Advanced marble
		if float64(seed) < float64(185)*luck { // Success
			resp, _, err = c.AddItem(&InventorySlot{ItemID: int64(17402815), Quantity: 1}, int16(marble1slot), false)
			if err != nil {
				return nil, err
			} else if resp == nil {
				return nil, nil
			}
			resp.Concat(PRODUCTION_SUCCESS)
		} else { // Faileds
			resp.Concat(PRODUCTION_FAILED)
		}
	}
	if marble2.ItemID >= 17300959 && marble2.ItemID <= 17300988 { //Steel Rare marble
		if float64(seed) < float64(175)*luck { // Success
			resp, _, err = c.AddItem(&InventorySlot{ItemID: int64(17402816), Quantity: 1}, int16(marble1slot), false)
			if err != nil {
				return nil, err
			} else if resp == nil {
				return nil, nil
			}
			resp.Concat(PRODUCTION_SUCCESS)
		} else { // Faileds
			resp.Concat(PRODUCTION_FAILED)
		}
	}
	if marble2.ItemID >= 17300989 && marble2.ItemID <= 17301018 { //Steel Legend marble
		if float64(seed) < float64(100)*luck { // Success
			resp, _, err = c.AddItem(&InventorySlot{ItemID: int64(17402817), Quantity: 1}, int16(marble1slot), false)
			if err != nil {
				return nil, err
			} else if resp == nil {
				return nil, nil
			}
			resp.Concat(PRODUCTION_SUCCESS)
		} else { // Faileds
			resp.Concat(PRODUCTION_FAILED)
		}
	}
	if marble2.ItemID >= 17301049 && marble2.ItemID <= 17301078 { //water Normal marble
		if float64(seed) < float64(200)*luck { // Success
			resp, _, err = c.AddItem(&InventorySlot{ItemID: int64(17402818), Quantity: 1}, int16(marble1slot), false)
			if err != nil {
				return nil, err
			} else if resp == nil {
				return nil, nil
			}
			resp.Concat(PRODUCTION_SUCCESS)
		} else { // Faileds
			resp.Concat(PRODUCTION_FAILED)
		}
	}
	if marble2.ItemID >= 17301079 && marble2.ItemID <= 17301108 { //water Advanced marble
		if float64(seed) < float64(185)*luck { // Success
			resp, _, err = c.AddItem(&InventorySlot{ItemID: int64(17402819), Quantity: 1}, int16(marble1slot), false)
			if err != nil {
				return nil, err
			} else if resp == nil {
				return nil, nil
			}
			resp.Concat(PRODUCTION_SUCCESS)
		} else { // Faileds
			resp.Concat(PRODUCTION_FAILED)
		}
	}
	if marble2.ItemID >= 17301108 && marble2.ItemID <= 17301138 { //water Rare marble
		if float64(seed) < float64(175)*luck { // Success
			resp, _, err = c.AddItem(&InventorySlot{ItemID: int64(17402820), Quantity: 1}, int16(marble1slot), false)
			if err != nil {
				return nil, err
			} else if resp == nil {
				return nil, nil
			}
			resp.Concat(PRODUCTION_SUCCESS)
		} else { // Faileds
			resp.Concat(PRODUCTION_FAILED)
		}
	}
	if marble2.ItemID >= 17301139 && marble2.ItemID <= 17301168 { //water Legend marble
		if float64(seed) < float64(100)*luck { // Success
			resp, _, err = c.AddItem(&InventorySlot{ItemID: int64(17402821), Quantity: 1}, int16(marble1slot), false)
			if err != nil {
				return nil, err
			} else if resp == nil {
				return nil, nil
			}
			resp.Concat(PRODUCTION_SUCCESS)
		} else { // Faileds
			resp.Concat(PRODUCTION_FAILED)
		}
	}

	resp.Concat(*c.DecrementItem(int16(marble2.SlotID), 1))
	resp.Concat(*c.DecrementItem(int16(marble3.SlotID), 1))
	if luck != 1.0 {
		resp.Concat(*c.DecrementItem(int16(hammer.SlotID), 1))
	}
	cost := 500
	c.LootGold(-uint64(cost))
	resp.Concat(c.GetGold())
	marble2.Delete()

	err = marble1.Update()
	if err != nil {
		return nil, err
	}

	return *resp, nil
}

func (c *Character) Synthesis(marbleslot int64, material1slot int64, material2slot int64, hammerslot int64) ([]byte, error) {
	//azure
	resp := &utils.Packet{}
	slots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}
	marble := slots[marbleslot]
	material1 := slots[material1slot]
	material2 := slots[material2slot]
	hammer := slots[hammerslot]

	luck := 1.0
	if marble.ItemID < 17300449 || marble.ItemID >= 17301168 {
		return nil, nil
	}
	if hammer.ItemID == 15700006 || hammer.ItemID == 17502899 || hammer.ItemID == 13000139 {
		luck = 2
	}
	if hammer.ItemID == 15700035 {
		luck = 3
	}

	marbleID := marble.ItemID
	resp.Concat(*c.DecrementItem(int16(marble.SlotID), 1))

	seed := int(utils.RandInt(0, 1000))
	if float64(seed) < float64(200)*luck { // Success
		resp, _, err = c.AddItem(&InventorySlot{ItemID: int64(marbleID + 1), Quantity: 1}, int16(marbleslot), false)
		if err != nil {
			return nil, err
		} else if resp == nil {
			return nil, nil
		}
		resp.Concat(PRODUCTION_SUCCESS)
	} else { // Faileds
		resp.Concat(PRODUCTION_FAILED)
	}
	resp.Concat(*c.DecrementItem(int16(material1.SlotID), 1))
	resp.Concat(*c.DecrementItem(int16(material2.SlotID), 1))
	cost := 2000
	c.LootGold(-uint64(cost))
	resp.Concat(c.GetGold())

	err = marble.Update()
	if err != nil {
		return nil, err
	}

	return *resp, nil

}
