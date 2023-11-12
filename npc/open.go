package npc

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/thoas/go-funk"
	"github.com/tidwall/gjson"
	"github.com/twodragon/kore-server/database"
	"github.com/twodragon/kore-server/messaging"

	"github.com/twodragon/kore-server/utils"
)

type OpenHandler struct {
}

type PressButtonHandler struct {
}

var (
	shops = map[int]int{20002: 7, 20003: 2, 20004: 4, 20005: 1, 20009: 8, 20010: 10, 20011: 10, 20013: 25,
		20024: 6, 20025: 6, 20026: 11, 20033: 21, 20034: 22, 20035: 23, 20036: 24, 20044: 21, 20047: 21, 20082: 21,
		20413: 25, 20379: 25, 20253: 25, 20251: 25, 20414: 25, 23725: 25, 20337: 25, 20323: 25, 20316: 25, 20290: 25, 20236: 25,
		20083: 21, 20084: 21, 20085: 23, 20086: 22, 20087: 21, 20094: 103, 20095: 100, 20105: 21, 20133: 25,
		20146: 21, 20151: 6, 20173: 327, 20211: 25, 20202: 105, 20239: 21, 20361: 329, 20364: 306, 20415: 21,
		20293: 340, 20160: 27, 20352: 29, 20204: 100, 20207: 101, 20205: 102, 20206: 21, 20203: 104, 20416: 344,
		20418: 343, 20419: 342, 20420: 345, 20179: 346, 25004: 362, 25001: 6, 25003: 8,
		20099: 507, 23724: 318, 22353: 304, 22212: 307, 20295: 508, 23908: 339, 23714: 343, 23909: 341, 23762: 342, 20161: 346, 20336: 26,
	}

	COMPOSITION_MENU    = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x0F, 0x01, 0x55, 0xAA}
	OPEN_SHOP           = utils.Packet{0xAA, 0x55, 0x07, 0x00, 0x57, 0x03, 0x01, 0x55, 0xAA}
	NPC_MENU            = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x57, 0x47, 0x55, 0xAA}
	STRENGTHEN_MENU     = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x08, 0x01, 0x55, 0xAA}
	STRENGHTEN_PET_MENU = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x31, 0x01, 0x55, 0xAA}

	GUILD_MENU          = utils.Packet{0xAA, 0x55, 0x02, 0x00, 0x57, 0x0D, 0x55, 0xAA}
	DISMANTLE_MENU      = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x16, 0x01, 0x55, 0xAA}
	EXTRACTION_MENU     = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x17, 0x01, 0x55, 0xAA}
	ADV_FUSION_MENU     = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x32, 0x01, 0x55, 0xAA}
	TACTICAL_SPACE      = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x50, 0x01, 0x01, 0x55, 0xAA}
	CREATE_SOCKET_MENU  = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x39, 0x01, 0x55, 0xAA}
	UPGRADE_SOCKET_MENU = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x3A, 0x01, 0x55, 0xAA}
	CONSIGNMENT_MENU    = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x42, 0x01, 0x55, 0xAA}

	SYNTHESIS_MENU            = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x45, 0x01, 0x55, 0xAA}
	HIGH_SYNTHETIC_MENU       = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x46, 0x01, 0x55, 0xAA}
	APPEARANCE_MENU           = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x41, 0x01, 0x55, 0xAA}
	RESTORE_APPEARANCE_MENU   = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x43, 0x01, 0x55, 0xAA}
	BOUNTY_MENU               = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x18, 0x00, 0x55, 0xAA}
	DESTROY_ITEM_MENU         = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x27, 0x01, 0x55, 0xAA}
	MATERIAL_EXTRACTION_MENU  = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x54, 0x13, 0x08, 0x10, 0x00, 0x55, 0xAA}
	ENHANCEMENT_TRANSFER_MENU = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x38, 0x01, 0x55, 0xAA}
	COOKING_MENU              = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x3d, 0x01, 0x55, 0xAA}

	DISC_UPGRADE = utils.Packet{0xaa, 0x55, 0x03, 0x00, 0x57, 0x49, 0x01, 0x55, 0xaa}

	//productions
	CO_PRODUCTION_MENU = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x3B, 0x55, 0xAA}

	//MESAGEGES	a
	INSUFICIENT_CLAN_RANK = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x24, 0x01, 0x55, 0xAA}
	JOB_PROMOTED          = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x09, 0x00, 0x55, 0xAA}
	NOT_ENOUGH_LEVEL      = utils.Packet{0xAA, 0x55, 0x0B, 0x00, 0x57, 0x02, 0x38, 0x42, 0x0F, 0x00, 0x00, 0x55, 0xAA}
	INVALID_CLASS         = utils.Packet{0xAA, 0x55, 0x0B, 0x00, 0x57, 0x02, 0x49, 0x2F, 0x00, 0x00, 0x00, 0x55, 0xAA}
	INVALID_REQUIREMENT   = utils.Packet{0xAA, 0x55, 0x0B, 0x00, 0x57, 0x02, 0x4A, 0x2A, 0x00, 0x00, 0x00, 0x55, 0xAA}
	EVOLVE_PET_MESSAGE    = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x26, 0x01, 0x55, 0xAA}

	QUEST_REWARD_MENU = utils.Packet{0xaa, 0x55, 0x0a, 0x00, 0x57, 0x37, 0x55, 0xaa}
	ACCEPT_QUEST      = utils.Packet{0xaa, 0x55, 0x30, 0x00, 0x57, 0x40, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xaa}
)

func (h *OpenHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	c := s.Character
	if c == nil {
		return nil, nil
	}

	u := s.User
	if u == nil {
		return nil, nil
	}

	id := uint16(utils.BytesToInt(data[6:10], true))
	pos, ok := database.GetFromRegister(1, c.Map, id).(*database.NpcPosition)
	if !ok {
		return nil, nil
	}

	c.LastNPCAction = 0
	npc, ok := database.GetNpcInfo(pos.NPCID)
	if !ok {
		return nil, nil
	}

	charLocation := database.ConvertPointToLocation(c.Coordinate)
	npcCoordinate := database.ConvertPointToCoordinate(pos.Min_X, pos.Min_Y)
	npcLocation := database.ConvertPointToLocation(npcCoordinate)
	distance := utils.CalculateDistance(charLocation, npcLocation)

	if distance > 10 {
		return nil, nil
	}

	if npc.ID == 20147 { // Ice Palace Mistress Lord
		coordinate := &utils.Location{X: 163, Y: 350}
		return c.Teleport(coordinate), nil

	} else if npc.ID == 20055 { // Mysterious Tombstone
		coordinate := &utils.Location{X: 365, Y: 477}
		return c.Teleport(coordinate), nil

	} else if npc.ID == 20056 { // Mysterious Tombstone (R)
		coordinate := &utils.Location{X: 70, Y: 450}
		return c.Teleport(coordinate), nil

	} else if npc.ID == 22351 { // Golden Castle Teleport Tombstone
		return c.ChangeMap(235, nil)

	} else if npc.ID == 22357 { // 2nd FL Entrance
		return c.ChangeMap(237, nil)

	} else if npc.ID == 22358 { // 3rd FL Entrance
		return c.ChangeMap(239, nil)
	}

	npcScript := database.NPCScripts[npc.ID]

	if npcScript == nil {
		//	resp := GetNPCMenu(npc.ID, 999993, 0, []int{3060})
		//c.Socket.Write(resp)
		return nil, nil
	}

	script := string(npcScript.Script)
	textID := gjson.Get(script, "text").Int()
	actions := []int{}

	for _, action := range gjson.Get(script, "actions").Array() {
		ac := int(action.Int())
		if ac == 3330 {
			continue
		}
		actions = append(actions, ac)
	}

	resp := NPC_MENU
	resp.Insert(utils.IntToBytes(uint64(npc.ID), 4, true), 6)        // npc id
	resp.Insert(utils.IntToBytes(uint64(textID), 4, true), 10)       // text id
	resp.Insert(utils.IntToBytes(uint64(len(actions)), 1, true), 14) // action length

	index, length := 15, int16(11)
	for i, action := range actions {
		resp.Insert(utils.IntToBytes(uint64(action), 4, true), index) // action
		index += 4

		resp.Insert(utils.IntToBytes(uint64(npc.ID), 2, true), index) // npc id
		index += 2

		resp.Insert(utils.IntToBytes(uint64(i+1), 2, true), index) // action index
		index += 2

		length += 8
	}

	resp.SetLength(length)
	return resp, nil
}

func (h *PressButtonHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	c := s.Character
	if c == nil {
		return nil, nil
	}

	npcID := int(utils.BytesToInt(data[6:8], true))

	index := int(utils.BytesToInt(data[8:10], true))
	indexes := []int{index & 7, (index & 56) / 8, (index & 448) / 64, (index & 3584) / 512, (index & 28672) / 4096}
	indexes = funk.FilterInt(indexes, func(i int) bool {
		return i > 0
	})

	npcScript := database.NPCScripts[npcID]
	if npcScript == nil {
		return nil, nil
	}

	script := string(npcScript.Script)
	key := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(indexes)), "."), "[]")
	script = gjson.Get(script, key).String()

	if script != "" {
		textID := int(gjson.Get(script, "text").Int())
		actions := []int{}

		for _, action := range gjson.Get(script, "actions").Array() {
			actions = append(actions, int(action.Int()))
		}

		resp := GetNPCMenu(npcID, textID, index, actions)
		return resp, nil
	} else { // Action button

		key := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(indexes[:len(indexes)-1])), "."), "[]")
		script = string(npcScript.Script)
		if key != "" {
			script = gjson.Get(script, key).String()
		}
		actID := int64(0)

		actions := gjson.Get(script, "actions").Array()
		actIndex := indexes[len(indexes)-1] - 1
		actID = actions[actIndex].Int()

		if database.DEVLOG == 1 {
			log.Print(fmt.Sprintf("ActINdex: %d", actIndex))
			log.Print(fmt.Sprintf("ActID: %d", actID))
		}
		resp := utils.Packet{}
		//log.Printf("actID: %d", index)
		var err error
		book1, book2, job := 0, 0, 0

		switch actID {
		case 1: // Exchange
			shopNo := shops[npcID]
			resp = OPEN_SHOP
			resp.Insert(utils.IntToBytes(uint64(shopNo), 4, true), 7) // shop id

		case 827: // Relic Store

			resp = OPEN_SHOP
			resp.Insert(utils.IntToBytes(uint64(307), 4, true), 7) // shop id

		case 3060: //QUEST LOAD

		case 2: // Compositon
			resp = COMPOSITION_MENU

		case 4: // Strengthen
			c.ShowUpgradingRate = false
			resp = STRENGTHEN_MENU

		case 4410: // Strengthen Info
			c.ShowUpgradingRate = true
			resp = STRENGTHEN_MENU

		case 6: // Deposit
			resp = c.BankItems()

		case 30011:
			//trick or treat event
			resp = c.TrickOrTreat()

		case 4227:
			//move to dark valley
			resp, _ = c.ChangeMap(77, nil)

		case 3433:
			//move to wutai mountain
			coordinate := &utils.Location{X: 429, Y: 287}
			resp, _ = c.ChangeMap(40, coordinate)

		case 3153:
			if s.Character.Exp >= 544951059310 && c.Level == 200 {
				resp, _ = c.ChangeMap(33, nil)
			}

		case 13: // Accept
			if c.Type == database.BEAST_KING || c.Type == database.EMPRESS {
				return nil, nil
			}
			switch npcID {
			case 20006: // Hunter trainer
				book1, job = 16210003, 13
				resp, err = firstJobPromotion(c, book1, job, npcID)
				if err != nil {
					return nil, err
				}
			case 20020: // Warrior trainer
				book1, job = 16210001, 11
				resp, err = firstJobPromotion(c, book1, job, npcID)
				if err != nil {
					return nil, err
				}
			case 20021: // Physician trainer
				book1, job = 16210002, 12
				resp, err = firstJobPromotion(c, book1, job, npcID)
				if err != nil {
					return nil, err
				}
			case 20022: // Assassin trainer
				book1, job = 16210004, 14
				resp, err = firstJobPromotion(c, book1, job, npcID)
				if err != nil {
					return nil, err
				}

			case 20415: // RDL tavern
				resp, _ = c.ChangeMap(254, nil)
			}

		case 64: // Create Guild
			if c.GuildID == -1 {
				resp = GUILD_MENU
			}
		case 201:
			{
				resp, _ = c.ChangeMap(236, nil)
			}

		case 77: // Move to Souther Plains
			resp, _ = c.ChangeMap(7, nil)

		case 78: // Move to Dragon Castle
			resp, _ = c.ChangeMap(1, nil)

		case 86: // Move to Spirit Spire
			resp, _ = c.ChangeMap(5, nil)

		case 103: // Move to Highlands
			resp, _ = c.ChangeMap(2, nil)

		case 104: // Move to Venom Swamp
			resp, _ = c.ChangeMap(3, nil)

		case 106: // Move to Silent Valley
			resp, _ = c.ChangeMap(11, nil)

		case 4083:
			resp, _ = c.ChangeMap(65, nil)
		case 4084:
			resp, _ = c.ChangeMap(64, nil)
		case 783:
			if c.Map == 101 {
				resp, _ = c.ChangeMap(102, nil)
			} else if c.Map == 102 {
				resp, _ = c.ChangeMap(101, nil)
			}
		case 821:
			resp, _ = c.ChangeMap(221, nil)
		case 822:
			resp, _ = c.ChangeMap(224, nil)
		case 823:
			resp, _ = c.ChangeMap(222, nil)
		case 824:
			resp, _ = c.ChangeMap(225, nil)
		case 825:
			resp, _ = c.ChangeMap(223, nil)
		case 826:
			resp, _ = c.ChangeMap(226, nil)
		case 1022:
			resp, _ = c.ChangeMap(227, nil)
		case 1023:
			resp, _ = c.ChangeMap(228, nil)
		case 30176:

			g, err := database.FindGuildByID(c.GuildID)
			if err != nil {
				return nil, err
			}
			if g == nil {
				return messaging.InfoMessage("You are not in a guild."), nil
			}
			c.Socket.User.SelectedServerID = g.ID
			resp, _ = c.ChangeMap(121, nil)

		case 148: // Become a Champion

			book1, book2, job = 16100039, 16100200, 21
			if c.Type == database.BEAST_KING || c.Type == database.EMPRESS {
				return nil, nil
			}
			resp, err = secondJobPromotion(c, book1, book2, 11, job, npcID)
			if err != nil {
				return nil, err
			}
		case 149: // Become a Musa
			book1, book2, job = 16100040, 16100200, 22
			if c.Type == database.BEAST_KING || c.Type == database.EMPRESS {
				return nil, nil
			}
			resp, err = secondJobPromotion(c, book1, book2, 11, job, npcID)
			if err != nil {
				return nil, nil
			}
		case 151: // Become a Surgeon
			book1, book2, job = 16100041, 16100200, 23
			if c.Type == database.BEAST_KING || c.Type == database.EMPRESS {
				return nil, nil
			}
			resp, err = secondJobPromotion(c, book1, book2, 12, job, npcID)
			if err != nil {
				return nil, err
			}
		case 152: // Become a Combat Medic
			book1, book2, job = 16100042, 16100200, 24
			if c.Type == database.BEAST_KING || c.Type == database.EMPRESS {
				return nil, nil
			}
			resp, err = secondJobPromotion(c, book1, book2, 12, job, npcID)
			if err != nil {
				return nil, err
			}
		case 154: // Become a Slayer
			book1, book2, job = 16100043, 16100200, 27
			if c.Type == database.BEAST_KING || c.Type == database.EMPRESS {
				return nil, nil
			}
			resp, err = secondJobPromotion(c, book1, book2, 14, job, npcID)
			if err != nil {
				return nil, err
			}
		case 155: // Become a Shinobi
			book1, book2, job = 16100044, 16100200, 28
			if c.Type == database.BEAST_KING || c.Type == database.EMPRESS {
				return nil, nil
			}
			resp, err = secondJobPromotion(c, book1, book2, 14, job, npcID)
			if err != nil {
				return nil, err
			}
		case 157: // Become a Tracker
			book1, book2, job = 16100045, 16100200, 25
			if c.Type == database.BEAST_KING || c.Type == database.EMPRESS {
				return nil, nil
			}
			resp, err = secondJobPromotion(c, book1, book2, 13, job, npcID)
			if err != nil {
				return nil, err
			}
		case 158: // Become a Ranger
			book1, book2, job = 16100046, 16100200, 26
			if c.Type == database.BEAST_KING || c.Type == database.EMPRESS {
				return nil, nil
			}
			resp, err = secondJobPromotion(c, book1, book2, 13, job, npcID)
			if err != nil {
				return nil, err
			}

		case 194: // Dismantle
			resp = DISMANTLE_MENU

		case 195: // Extraction
			resp = EXTRACTION_MENU

		case 524: // Exit Paid Zone
			if maps, ok := database.DKMaps[c.Map]; ok {
				resp, err = c.ChangeMap(maps[0], nil)
				if err != nil {
					return nil, err
				}
			}

		case 525: // Enter Paid Zone
			f := func(item *database.InventorySlot) bool {
				return item.Activated
			}
			_, item, err := c.FindItemInInventoryByType(f, database.DEATHKING_CASTLE_TICKET) //DEATHKING CASTLE TICKET
			if err != nil {
				return nil, err
			} else if item == nil { // You don't have ticket
				resp := GetNPCMenu(npcID, 999993, 0, nil)
				return resp, nil
			}

			if maps, ok := database.DKMaps[c.Map]; ok {
				resp, err = c.ChangeMap(maps[1], nil)
				if err != nil {
					return nil, err
				}
			}

		case 559: // Advanced Fusion
			resp = ADV_FUSION_MENU
		case 526: //Get Divine Skills

		case 631: // Tactical Space
			resp = TACTICAL_SPACE

		case 732: // Flexible Castle Entry
			f := func(item *database.InventorySlot) bool {
				return item.Activated
			}
			_, item, err := c.FindItemInInventoryByType(f, database.PANDEMONIUM_ENTRY_TICKET)
			if err != nil {
				return nil, err
			} else if item == nil { // You don't have ticket
				resp := GetNPCMenu(npcID, 999993, 0, nil)
				return resp, nil
			}

			if maps, ok := database.DKMaps[c.Map]; ok {
				resp, err = c.ChangeMap(maps[2], nil)
				if err != nil {
					return nil, err
				}
			}

		case 633: //Scissors
			resp, err = rpsgame(c, actID)
			if err != nil {
				return nil, err
			}
		case 634: //Rock
			resp, err = rpsgame(c, actID)
			if err != nil {
				return nil, err
			}
		case 635: //Paper
			resp, err = rpsgame(c, actID)
			if err != nil {
				return nil, err
			}

		case 737: // Create Socket
			resp = CREATE_SOCKET_MENU

		case 738: // Upgrade Socket
			resp = UPGRADE_SOCKET_MENU

		case 739: // Co-production menu
			resp = CO_PRODUCTION_MENU
			resp.Insert(utils.IntToBytes(1, 1, true), 6)
			if npcID == 23714 {
				resp = CO_PRODUCTION_MENU
				resp.Insert(utils.IntToBytes(151, 1, true), 6)
			}
		case 3338:
			resp = CO_PRODUCTION_MENU
			resp.Insert(utils.IntToBytes(3, 1, true), 6)
		case 3296:
			resp = CO_PRODUCTION_MENU
			resp.Insert(utils.IntToBytes(22, 1, true), 6)
		case 3339:
			resp = CO_PRODUCTION_MENU
			resp.Insert(utils.IntToBytes(4, 1, true), 6)
		case 3340:
			resp = CO_PRODUCTION_MENU
			resp.Insert(utils.IntToBytes(16, 1, true), 6)
		case 3341:
			resp = CO_PRODUCTION_MENU
			resp.Insert(utils.IntToBytes(20, 1, true), 6)
		case 906: //APPEARANCE CHANGE
			resp = APPEARANCE_MENU

		case 985: //APPEARANCE RESTORE
			resp = RESTORE_APPEARANCE_MENU

		case 208: //Character appearance change
			slot, _, err := s.Character.FindItemInInventory(nil, 15830000, 15830001, 17502883)
			if err != nil {
				log.Println(err)
				return nil, err
			} else if slot == -1 {
				return nil, nil
			}
			resp = utils.Packet{0xAA, 0x55, 0x09, 0x00, 0x01, 0xB4, 0x0A, 0x00, 0xFF, 0xFF, 0xFF, 0xFF, 0xB4, 0x55, 0xAA}
		case 970: // Consignment
			resp = CONSIGNMENT_MENU
		case 3230: //High-grade synthetic
			resp = HIGH_SYNTHETIC_MENU
		case 3231: //High-grade synthetic
			resp = SYNTHESIS_MENU
		case 3337:
			_, item, err := c.FindItemInInventory(nil, 13003290)
			if err != nil {
				return nil, err
			}
			if item == nil || item.Quantity < 30 {
				resp.Concat(messaging.InfoMessage("You don't have enough Lights Pieces, come back with 30 pieces!"))
				return resp, nil
			}
			freeslot, err := c.FindFreeSlot()
			if err == nil {
				if freeslot != -1 {
					itemData, _, err := c.AddItem(&database.InventorySlot{ItemID: 13003291, Quantity: 120}, -1, false)
					if err != nil {
						return nil, err
					}
					data := c.DecrementItem(item.SlotID, 30)
					c.Socket.Write(*data)
					c.Socket.Write(*itemData)
					c.Update()
				} else {
					resp.Concat(messaging.InfoMessage("You don't have enough space in your invenotry"))
					return resp, nil
				}
			}
		case 3336:
			_, item, err := c.FindItemInInventory(nil, 13003290)
			if err != nil {
				return nil, err
			}
			if item == nil || item.Quantity < 30 {
				resp.Concat(messaging.InfoMessage("You don't have enough Lights Pieces, come back with 30 pieces!"))
				return resp, nil
			}
			freeslot, err := c.FindFreeSlot()
			if err == nil {
				if freeslot != -1 {
					itemData, _, err := c.AddItem(&database.InventorySlot{ItemID: 13003292, Quantity: 120}, -1, false)
					if err != nil {
						return nil, err
					}
					data := c.DecrementItem(item.SlotID, 30)
					c.Socket.Write(*data)
					c.Socket.Write(*itemData)
					c.Update()
				} else {
					resp.Concat(messaging.InfoMessage("You don't have enough space in your invenotry"))
					return resp, nil
				}
			}

		case 3335:
			_, item, err := c.FindItemInInventory(nil, 13003290)
			if err != nil {
				return nil, err
			}
			if item == nil || item.Quantity < 30 {
				resp.Concat(messaging.InfoMessage("You don't have enough Lights Pieces, come back with 30 pieces!"))
				return resp, nil
			}
			freeslot, err := c.FindFreeSlot()
			if err == nil {
				if freeslot != -1 {
					itemData, _, err := c.AddItem(&database.InventorySlot{ItemID: 13003293, Quantity: 120}, -1, false)
					if err != nil {
						return nil, err
					}
					data := c.DecrementItem(item.SlotID, 30)
					c.Socket.Write(*data)
					c.Socket.Write(*itemData)
					c.Update()
				} else {
					resp.Concat(messaging.InfoMessage("You don't have enough space in your invenotry"))
					return resp, nil
				}
			}

		case 3334:
			_, lights, err := c.FindItemInInventory(nil, 13003290)
			if err != nil {
				return nil, err
			}
			if lights == nil || lights.Quantity < 120 {
				resp.Concat(messaging.InfoMessage("You don't have enough Lights Pieces, come back with 120 pieces!"))
				return resp, nil
			}
			_, coins, err := c.FindItemInInventory(nil, 17502306)
			if err != nil {
				return nil, err
			}
			if coins == nil || coins.Quantity < 3 {
				resp.Concat(messaging.InfoMessage("You don't have enough Guardian Young Coins, come back with 3 pieces!"))
				return resp, nil
			}
			freeslot, err := c.FindFreeSlot()
			if err == nil {
				if freeslot != -1 {
					itemData, _, err := c.AddItem(&database.InventorySlot{ItemID: 13003294, Quantity: 7200}, -1, false)
					if err != nil {
						return nil, err
					}
					data := c.DecrementItem(lights.SlotID, 120)
					c.Socket.Write(*data)
					data = c.DecrementItem(coins.SlotID, 3)
					c.Socket.Write(*data)
					c.Socket.Write(*itemData)
					c.Update()
				} else {
					resp.Concat(messaging.InfoMessage("You don't have enough space in your invenotry"))
					return resp, nil
				}
			}

		case 3316:
			c.Reborn()
			return nil, nil
		case 3326:

			_, item, err := c.FindItemInInventory(nil, 17502418)
			if err != nil {
				return nil, err
			}
			if item == nil || item.Quantity < 10 {
				resp.Concat(messaging.InfoMessage("You don't have enough Lucky Coins, come back with 10 pieces!"))
				return resp, nil
			} else if item.Quantity >= 10 {
				freeslot, err := c.FindFreeSlot()
				if err == nil {
					if freeslot != -1 {
						itemData, _, err := c.AddItem(&database.InventorySlot{ItemID: 17502465, Quantity: 1}, -1, false)
						if err != nil {
							return nil, err
						}
						data := c.DecrementItem(item.SlotID, 10)
						c.Socket.Write(*data)
						c.Socket.Write(*itemData)
						c.Update()
					} else {
						resp.Concat(messaging.InfoMessage("You don't have enough space in your invenotry"))
						return resp, nil
					}
				}

			}
		case 640:
			_, item, err := c.FindItemInInventory(nil, 17502199)
			if err != nil {
				return nil, err
			}
			if item == nil || item.Quantity < 1 {
				resp.Concat(messaging.InfoMessage("You don't have enough decorations!"))
				return resp, nil
			} else if item.Quantity >= 1 {
				freeslot, err := c.FindFreeSlot()
				if err == nil {
					if freeslot != -1 {
						itemData, _, err := c.AddItem(&database.InventorySlot{ItemID: 99002840, Quantity: 1}, -1, false)
						if err != nil {
							return nil, err
						}
						data := c.DecrementItem(item.SlotID, 1)
						c.Socket.Write(*data)
						c.Socket.Write(*itemData)
						c.Update()
					} else {
						resp.Concat(messaging.InfoMessage("You don't have enough space in your invenotry"))
						return resp, nil
					}
				}

			}
		case 3327:
			_, item, err := c.FindItemInInventory(nil, 100080294)
			if err != nil {
				return nil, err
			}
			if item == nil || item.Quantity < 1 {
				resp.Concat(messaging.InfoMessage("You don't have (Exchange Charm)! It can be obtained from daily War."))
				return resp, nil
			}

			if c.TradeID != "" {
				return messaging.InfoMessage("cannot do that while trading"), nil
			}
			sale := database.FindSale(s.Character.PseudoID)
			if sale != nil {
				return messaging.InfoMessage("cannot do that while trading"), nil
			}
			if s.Character.Gold < 500000000 {
				return messaging.InfoMessage("You don't have enough gold."), nil
			}
			s.Character.Gold -= 500000000
			resp.Concat(c.GetGold())
			c.Socket.User.NCash += uint64(1000)
			c.Socket.User.Update()
			data := c.DecrementItem(item.SlotID, 1)
			c.Socket.Write(*data)
			c.Update()
			resp.Concat(messaging.InfoMessage("You exchaged 500mil gold into 1000 nC"))
			return resp, nil

		case 3333:
			resp := utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x3B, 0x96, 0x55, 0xAA}
			return resp, nil

		case 802:
			resp = COOKING_MENU

		case 207: //Bounty Hunt
			resp := BOUNTY_MENU

			return resp, nil
		case 854:
			if c.Type == database.BEAST_KING || c.Type == database.DIVINE_BEAST_KING || c.Type == database.DARKNESS_BEAST_KING || c.Type == database.EMPRESS || c.Type == database.DIVINE_EMPRESS || c.Type == database.DARKNESS_EMPRESS {
				resp, _ = c.ChangeMap(70, nil)
			}
		case 586: //GEUK MAPPOK
			if c.Level >= 150 {
				resp, _ = c.ChangeMap(30, nil)
			}
		case 197101: // Move to Marketplace
			resp, _ = c.ChangeMap(254, nil)
		//case 753:
		case 742: //GEUK MAPPOK
			if c.Level <= 100 {
				resp, _ = c.ChangeMap(93, nil)
			}
		case 743: //GEUK MAPPOK
			if c.Level <= 100 {
				resp, _ = c.ChangeMap(94, nil)
			}
		case 744: //GEUK MAPPOK
			if c.Level <= 100 {
				resp, _ = c.ChangeMap(95, nil)
			}
		case 745: //GEUK MAPPOK
			if c.Level > 100 {
				resp, _ = c.ChangeMap(96, nil)
			}
		case 746: //GEUK MAPPOK
			if c.Level > 100 {
				resp, _ = c.ChangeMap(97, nil)
			}
		case 747: //GEUK MAPPOK
			if c.Level > 100 {
				resp, _ = c.ChangeMap(98, nil)
			}
		case 748: //GEUK MAPPOK
			if c.Level > 100 {
				resp, _ = c.ChangeMap(99, nil)
			}
		case 3233: // NON DIVINE Sawangcheon
			if c.Level <= 100 {
				resp, _ = c.ChangeMap(100, nil)
			}
		case 3235: //DIVINE Sawangcheon
			if c.Level > 100 {
				resp, _ = c.ChangeMap(101, nil)
			}
		case 40:
			//resp = DISCRIMINATE_CRAFT

		case 92:
			job := 31
			book1 := 100031001
			book2 := 100030013
			book3 := 16100300
			jobName := "Warlord"
			resp, err = divineJobPromotion(c, book1, book2, book3, job, npcID, jobName)
			if err != nil {
				return nil, err
			}
		case 93:
			job := 33
			book1 := 100031002
			book2 := 100030014
			book3 := 16100300
			jobName := "Beastlord"
			resp, err = divineJobPromotion(c, book1, book2, book3, job, npcID, jobName)
			if err != nil {
				return nil, err
			}
		case 94:
			job := 32
			book1 := 100031003
			book2 := 100030015
			book3 := 16100300
			jobName := "HolyHand"
			resp, err = divineJobPromotion(c, book1, book2, book3, job, npcID, jobName)
			if err != nil {
				return nil, err
			}
		case 108:
			job := 34
			book1 := 100031004
			book2 := 100030016
			book3 := 16100300
			jobName := "ShadowLord"
			resp, err = divineJobPromotion(c, book1, book2, book3, job, npcID, jobName)
			if err != nil {
				return nil, err
			}
		case 3309: //Become God of War
			book1, book2, job = 100030020, 100030021, 41
			book3 := 100032001
			jobName := "God of War"
			resp, err = darknessJobPromotion(c, book1, book2, book3, job, npcID, jobName)
			if err != nil {
				return nil, err
			}

		case 3310: //Become God of Death
			book1, book2, job = 100030022, 100030023, 42
			book3 := 100032002
			jobName := "God of Death"
			resp, err = darknessJobPromotion(c, book1, book2, book3, job, npcID, jobName)
			if err != nil {
				return nil, err
			}
		case 3311: //Become God of Blade
			book1, book2, job = 100030024, 100030025, 43
			book3 := 100032003
			jobName := "God of Blade"
			resp, err = darknessJobPromotion(c, book1, book2, book3, job, npcID, jobName)
			if err != nil {
				return nil, err
			}
		case 10001:
			c.CheckIn()

		case 10002:
			resp, err = c.ClaimCheckIn()
			if err != nil {
				return nil, err
			}

		case 3087:
			if c.Map == 17 {
				if c.PartyID == "" {
					resp = GetNPCMenu(npcID, 2010000192, 0, nil)
				} else {
					party := database.FindParty(c)
					if party.Leader.ID != c.ID {
						resp = GetNPCMenu(npcID, 2010000192, 0, nil)
						return resp, nil
					}
					if len(party.Members) > 3 {
						msg := messaging.InfoMessage("Maximum 3 players can entry at once.")
						party.Leader.Socket.Write(msg)
						return nil, nil
					}

					if c.YingYangTicketsLeft <= 0 {
						resp = GetNPCMenu(npcID, 2010000190, 0, nil)
						return resp, nil
					} else if c.Level < 60 || c.Level > 100 {
						resp = GetNPCMenu(npcID, 2010000191, 0, nil)
						return resp, nil
					}

					for _, member := range party.Members {

						if member.Character.YingYangTicketsLeft <= 0 {
							resp = GetNPCMenu(npcID, 2010000190, 0, nil)
							return resp, nil
						} else if member.Character.Level < 60 || member.Character.Level > 100 {
							resp = GetNPCMenu(npcID, 2010000191, 0, nil)
							return resp, nil
						}

						member.Character.YingYangTicketsLeft--
						member.Character.Update()
					}
					c.YingYangTicketsLeft--
					c.Update()
					database.StartYingYang(party)
				}
			} else if c.Map == 24 {
				if c.PartyID == "" {
					resp = GetNPCMenu(npcID, 2010000192, 0, nil)
				} else {
					party := database.FindParty(c)
					if party.Leader.ID != c.ID {
						resp = GetNPCMenu(npcID, 2010000192, 0, nil)
						return resp, nil
					}
					if len(party.Members) > 3 {
						msg := messaging.InfoMessage("Maximum 3 players can entry at once.")
						party.Leader.Socket.Write(msg)
						return nil, nil
					}

					if c.YingYangTicketsLeft <= 0 {
						resp = GetNPCMenu(npcID, 2010000190, 0, nil)
						return resp, nil
					} else if c.Level < 101 || c.Level > 200 {
						resp = GetNPCMenu(npcID, 2010000191, 0, nil)
						return resp, nil
					}

					for _, member := range party.Members {

						if member.Character.YingYangTicketsLeft <= 0 {
							resp = GetNPCMenu(npcID, 2010000190, 0, nil)
							return resp, nil
						} else if member.Character.Level < 101 || member.Character.Level > 200 {
							resp = GetNPCMenu(npcID, 2010000191, 0, nil)
							return resp, nil
						}

						member.Character.YingYangTicketsLeft--
						member.Character.Update()
					}
					c.YingYangTicketsLeft--
					c.Update()
					go database.StartDivineYingYang(party)
				}
			}

		case 232: //Shin-Gang Region
			if c.Level > 100 {
				if c.Faction == 1 {
					resp, _ = c.ChangeMap(22, nil)
				} else if c.Faction == 2 {
					resp, _ = c.ChangeMap(23, nil)
				}
			}

		case 4403:
			resp = CO_PRODUCTION_MENU
			resp.Insert(utils.IntToBytes(6, 1, true), 6)
		case 4404:
			resp = CO_PRODUCTION_MENU
			resp.Insert(utils.IntToBytes(7, 1, true), 6)
		case 4405:
			resp = CO_PRODUCTION_MENU
			resp.Insert(utils.IntToBytes(8, 1, true), 6)
		case 4406:
			resp = CO_PRODUCTION_MENU
			resp.Insert(utils.IntToBytes(9, 1, true), 6)
		case 4407:
			resp = CO_PRODUCTION_MENU
			resp.Insert(utils.IntToBytes(10, 1, true), 6)
		case 4408:
			resp = CO_PRODUCTION_MENU
			resp.Insert(utils.IntToBytes(11, 1, true), 6)
		case 4409:
			resp = CO_PRODUCTION_MENU
			resp.Insert(utils.IntToBytes(12, 1, true), 6)
		case 4052:
			resp = CO_PRODUCTION_MENU
			resp.Insert(utils.IntToBytes(23, 1, true), 6)

		case 4125:

			resp = CO_PRODUCTION_MENU
			resp.Insert(utils.IntToBytes(50, 1, true), 6)
			if npcID == 25000 {
				resp = CO_PRODUCTION_MENU
				resp.Insert(utils.IntToBytes(70, 1, true), 6)
			}

		case 3426:
			c.ShowUpgradingRate = false
			resp = DISC_UPGRADE

		case 4411:
			c.ShowUpgradingRate = true
			resp = DISC_UPGRADE

		case 3321:
			if c.Level > 100 {
				msg := messaging.InfoMessage("You don't have the required level.")
				return msg, nil
			}
			database.StartSeasonDungeon(s)
		case 30177: //easter egg hunt
			_, item, err := c.FindItemInInventory(nil, 13000035)
			if err != nil {
				return nil, err
			}
			if item == nil || item.Quantity < 22 {
				resp.Concat(messaging.InfoMessage("You don't have enough eggs!"))
				return resp, nil
			} else if item.Quantity >= 22 {
				data := c.DecrementItem(item.SlotID, 22)
				c.Socket.Write(*data)
				itemData, _, _ := c.AddItem(&database.InventorySlot{ItemID: 13003191, Quantity: uint(1)}, -1, false)
				c.Socket.Write(*itemData)
			}
		case 30026: //easter blessing

			infection := database.BuffInfections[11141]
			c.AddBuff(infection, 7260)

		case 3322:
			f := func(item *database.InventorySlot) bool {
				return item.Activated
			}
			_, item, err := c.FindItemInInventory(f, 200000038, 200000039)
			if err != nil {
				return nil, err
			} else if item == nil { // You don't have ticket
				resp := GetNPCMenu(npcID, 999993, 0, nil)
				return resp, nil
			} else if c.Reborns < 1 { // You don't have right Reborn level
				resp := GetNPCMenu(npcID, 999993, 0, nil)
				return resp, nil
			}
			resp, err = c.ChangeMap(72, nil)
			if err != nil {
				return nil, err
			}
			return resp, nil
		case 3046:
			_, item, err := c.FindItemInInventory(nil, 99059990, 99059991, 99059992)
			if item == nil || err != nil || item.SlotID != 11 {
				return messaging.InfoMessage("You don't have the requested item."), err
			}
			itemid := item.ItemID
			resp, err = c.RemoveItem(item.SlotID)
			if err != nil {
				return nil, err
			}
			database.AddFlagToFlagKingdomFaction(c, itemid)

			return resp, nil

		case 3323:
			f := func(item *database.InventorySlot) bool {
				return item.Activated
			}
			_, item, err := c.FindItemInInventory(f, 200000038, 200000039)
			if err != nil {
				return nil, err
			} else if item == nil { // You don't have ticket
				resp := GetNPCMenu(npcID, 999993, 0, nil)
				return resp, nil
			} else if c.Reborns < 2 { // You don't have right Reborn level
				resp := GetNPCMenu(npcID, 999993, 0, nil)
				return resp, nil
			}
			resp, err = c.ChangeMap(73, nil)
			if err != nil {
				return nil, err
			}
			return resp, nil
		case 3324:
			f := func(item *database.InventorySlot) bool {
				return item.Activated
			}
			_, item, err := c.FindItemInInventory(f, 200000038, 200000039)
			if err != nil {
				return nil, err
			} else if item == nil { // You don't have ticket
				resp := GetNPCMenu(npcID, 999993, 0, nil)
				return resp, nil
			} else if c.Reborns < 3 { // You don't have right Reborn level
				resp := GetNPCMenu(npcID, 999993, 0, nil)
				return resp, nil
			}
			resp, err = c.ChangeMap(74, nil)
			if err != nil {
				return nil, err
			}
			return resp, nil
		case 3325:
			f := func(item *database.InventorySlot) bool {
				return item.Activated
			}
			_, item, err := c.FindItemInInventory(f, 200000038, 200000039)
			if err != nil {
				return nil, err
			} else if item == nil { // You don't have ticket
				resp := GetNPCMenu(npcID, 999993, 0, nil)
				return resp, nil
			} else if c.Reborns < 4 { // You don't have right Reborn level
				resp := GetNPCMenu(npcID, 999993, 0, nil)
				return resp, nil
			}
			resp, err = c.ChangeMap(75, nil)
			if err != nil {
				return nil, err
			}
			return resp, nil

		case 302: //Flag Kingdom
			database.AddMemberToFlagKingdom(c)
		case 116: //FACTION WAR
			database.AddMemberToFactionWar(c)
		case 542: //THE GREAT WAR
			database.AddPlayerToGreatWar(c)

		case 30007: //easter egg hunt
			freeslot, err := c.FindFreeSlot()
			if freeslot <= 0 || err != nil {
				return messaging.InfoMessage("You don't have enough space"), nil
			}
			_, item, err := c.FindItemInInventory(nil, 13000036)
			if err != nil {
				return nil, err
			}
			if item == nil || item.Quantity < 10 {
				resp.Concat(messaging.InfoMessage("You don't have enough eggs! You need 10 eggs."))
				return resp, nil
			} else if item.Quantity >= 10 {
				data := c.DecrementItem(item.SlotID, 10)
				c.Socket.Write(*data)
				itemData, _, _ := c.AddItem(&database.InventorySlot{ItemID: 13003191, Quantity: 1}, -1, false)
				c.Socket.Write(*itemData)
			}
		case 281:
			// resp = CO_PRODUCTION_MENU
			// resp.Insert(utils.IntToBytes(2, 1, true), 6)
			// return resp, nil
			if s.Character.Exp >= 233332051410 && c.Level == 100 {
				itemData, _, _ := c.AddItem(&database.InventorySlot{ItemID: 17504123, Quantity: 1}, -1, false)
				c.Socket.Write(*itemData)
			}
		case 3329:

		case 3315:
			return ENHANCEMENT_TRANSFER_MENU, nil

		case 3332:
			resp, _ = c.ChangeMap(33, nil)

		case 1020:
			buff, err := database.FindBuffByID(11219, c.ID)
			if err != nil {
				return nil, nil
			}
			if buff == nil {
				infection := database.BuffInfections[11219]
				c.AddBuff(infection, 86400)

				r, _, err := c.AddItem(&database.InventorySlot{ItemID: 92001033, Quantity: 20}, -1, false)
				if err != nil {
					return resp, err
				} else if r == nil {
					return nil, nil
				}

				resp.Concat(*r)
			}

		}
		return resp, nil
	}
}

func GetNPCMenu(npcID, textID, index int, actions []int) []byte {
	resp := NPC_MENU
	resp.Insert(utils.IntToBytes(uint64(npcID), 4, true), 6)         // npc id
	resp.Insert(utils.IntToBytes(uint64(textID), 4, true), 10)       // text id
	resp.Insert(utils.IntToBytes(uint64(len(actions)), 1, true), 14) // action length

	counter, length := 15, int16(11)
	for i, action := range actions {
		resp.Insert(utils.IntToBytes(uint64(action), 4, true), counter) // action
		counter += 4

		resp.Insert(utils.IntToBytes(uint64(npcID), 2, true), counter) // npc id
		counter += 2

		actIndex := int(index) + (i+1)<<(len(actions)*3)
		resp.Insert(utils.IntToBytes(uint64(actIndex), 2, true), counter) // action index
		counter += 2

		length += 8
	}

	resp.SetLength(length)
	return resp
}

func firstJobPromotion(c *database.Character, book, job, npcID int) (utils.Packet, error) {
	resp := utils.Packet{}
	if c.Class == 0 && c.Level >= 10 {
		c.Class = job
		resp = JOB_PROMOTED
		resp[6] = byte(job)

		r, _, err := c.AddItem(&database.InventorySlot{ItemID: int64(book), Quantity: 1}, -1, false)
		if err != nil {
			return resp, err
		} else if r == nil {
			return nil, nil
		}

		resp.Concat(*r)

	} else if c.Level < 10 {
		resp = NOT_ENOUGH_LEVEL
		resp.Insert(utils.IntToBytes(uint64(npcID), 4, true), 6) // npc id
	} else {
		resp = INVALID_CLASS
		resp.Insert(utils.IntToBytes(uint64(npcID), 4, true), 6) // npc id
	}

	return resp, nil
}

func secondJobPromotion(c *database.Character, book1, book2, preJob, job, npcID int) (utils.Packet, error) {
	resp := utils.Packet{}
	if c.Class == preJob && c.Level >= 50 {
		c.Class = job
		resp = JOB_PROMOTED
		resp[6] = byte(job)

		r, _, err := c.AddItem(&database.InventorySlot{ItemID: int64(book1), Quantity: 1}, -1, false)
		if err != nil {
			return resp, err
		} else if r == nil {
			return nil, nil
		}

		resp.Concat(*r)

		if book2 != 0 {
			r, _, err = c.AddItem(&database.InventorySlot{ItemID: int64(book2), Quantity: 1}, -1, false)
			if err != nil {
				return resp, err
			} else if r == nil {
				return nil, nil
			}

			resp.Concat(*r)
		}

	} else if c.Level < 50 {
		resp := NOT_ENOUGH_LEVEL
		resp.Insert(utils.IntToBytes(uint64(npcID), 4, true), 6) // npc id
	} else {
		resp = INVALID_CLASS
		resp.Insert(utils.IntToBytes(uint64(npcID), 4, true), 6) // npc id
	}

	return resp, nil
}

func divineJobPromotion(c *database.Character, book1, book2, book3, job, npcID int, jobName string) (utils.Packet, error) {
	resp := utils.Packet{}
	if c.Class == 0 {
		c.Class = job
		c.Update()
		resp = JOB_PROMOTED
		resp[6] = byte(c.Class)

		r, _, err := c.AddItem(&database.InventorySlot{ItemID: int64(book1), Quantity: 1}, -1, false)
		if err != nil {
			return resp, err
		} else if r == nil {
			return nil, nil
		}

		resp.Concat(*r)

		r, _, err = c.AddItem(&database.InventorySlot{ItemID: int64(book2), Quantity: 1}, -1, false)
		if err != nil {
			return resp, err
		} else if r == nil {
			return nil, nil
		}

		resp.Concat(*r)

		r, _, err = c.AddItem(&database.InventorySlot{ItemID: int64(book3), Quantity: 1}, -1, false)
		if err != nil {
			return resp, err
		} else if r == nil {
			return nil, nil
		}

		resp.Concat(*r)
		c.Update()
		resp.Concat(messaging.InfoMessage(fmt.Sprintf("Promoted as a %s.", jobName))) //NOTICE TO PROMOTE
		//r = c.ChangeMap(1, nil)
	} else {
		resp = INVALID_CLASS
		resp.Insert(utils.IntToBytes(uint64(npcID), 4, true), 6) // npc id
	}
	return resp, nil
}

func darknessJobPromotion(c *database.Character, book1, book2, book3, job, npcID int, jobName string) (utils.Packet, error) {
	resp := utils.Packet{}
	if c.Class == 40 {
		c.Class = job
		c.Update()
		resp = JOB_PROMOTED
		resp[6] = byte(c.Class)

		r, _, err := c.AddItem(&database.InventorySlot{ItemID: int64(book1), Quantity: 1}, -1, false)
		if err != nil {
			return resp, err
		} else if r == nil {
			return nil, nil
		}

		resp.Concat(*r)

		r, _, err = c.AddItem(&database.InventorySlot{ItemID: int64(book2), Quantity: 1}, -1, false)
		if err != nil {
			return resp, err
		} else if r == nil {
			return nil, nil
		}

		resp.Concat(*r)

		r, _, err = c.AddItem(&database.InventorySlot{ItemID: int64(book3), Quantity: 1}, -1, false)
		if err != nil {
			return resp, err
		} else if r == nil {
			return nil, nil
		}

		resp.Concat(*r)
		c.Update()
		resp.Concat(messaging.InfoMessage(fmt.Sprintf("Promoted as a %s.", jobName))) //NOTICE TO PROMOTE
		//r = c.ChangeMap(1, nil)
	} else {
		resp = INVALID_CLASS
		resp.Insert(utils.IntToBytes(uint64(npcID), 4, true), 6) // npc id
	}
	return resp, nil
}
func GoBackNPCMenu(NPCID int, c *database.Character) ([]byte, error) {

	c.LastNPCAction = 0
	npc, ok := database.GetNpcInfo(NPCID)
	if !ok {
		return nil, errors.New("NPC not found")
	}
	npcScript := database.NPCScripts[npc.ID]
	if npcScript == nil {
		resp := GetNPCMenu(npc.ID, 999993, 0, []int{30011})
		c.Socket.Write(resp)
		return nil, nil
	}

	script := string(npcScript.Script)
	textID := gjson.Get(script, "text").Int()
	actions := []int{}

	for _, action := range gjson.Get(script, "actions").Array() {
		actions = append(actions, int(action.Int()))
	}

	resp := NPC_MENU
	resp.Insert(utils.IntToBytes(uint64(npc.ID), 4, true), 6)        // npc id
	resp.Insert(utils.IntToBytes(uint64(textID), 4, true), 10)       // text id
	resp.Insert(utils.IntToBytes(uint64(len(actions)), 1, true), 14) // action length

	index, length := 15, int16(11)
	for i, action := range actions {
		resp.Insert(utils.IntToBytes(uint64(action), 4, true), index) // action
		index += 4

		resp.Insert(utils.IntToBytes(uint64(npc.ID), 2, true), index) // npc id
		index += 2

		resp.Insert(utils.IntToBytes(uint64(i+1), 2, true), index) // action index
		index += 2

		length += 8
	}

	resp.SetLength(length)
	return resp, nil
}

func rpsgame(c *database.Character, playerchoice int64) ([]byte, error) {
	resp := utils.Packet{}
	var textID int

	_, item, err := c.FindItemInInventory(nil, 17300116)
	if err != nil {
		return nil, err
	}
	if item == nil || item.Quantity < 1 {
		resp.Concat(messaging.InfoMessage("You don't have enough Coins to play!"))
		return resp, nil
	}
	compchoice := utils.RandInt(633, 635)
	if compchoice == playerchoice { //draw cases
		if playerchoice == 633 {
			textID = 1013910008
		} else if playerchoice == 634 {
			textID = 1013910011
		} else if playerchoice == 635 {
			textID = 1013910014
		}
	} else if compchoice == 633 && playerchoice == 634 || compchoice == 634 && playerchoice == 635 || compchoice == 635 && playerchoice == 633 { //win cases
		if playerchoice == 633 {
			textID = 1013910009
		} else if playerchoice == 634 {
			textID = 1013910012
		} else if playerchoice == 635 {
			textID = 1013910015
		}
		data := c.DecrementItem(item.SlotID, 1)
		resp.Concat(*data)
		itemData, _, _ := c.AddItem(&database.InventorySlot{ItemID: 100080299, Quantity: uint(1)}, -1, false)
		resp.Concat(*itemData)

	} else {
		if playerchoice == 633 {
			textID = 1013910010
		} else if playerchoice == 634 {
			textID = 1013910013
		} else if playerchoice == 635 {
			textID = 1013910016
		}
		data := c.DecrementItem(item.SlotID, 1)
		resp.Concat(*data)
	}

	resp.Concat(GetNPCMenu(20326, textID, 0, nil))
	return resp, nil
}
