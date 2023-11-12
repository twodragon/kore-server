package player

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/twodragon/kore-server/database"
	"github.com/twodragon/kore-server/messaging"
	"github.com/twodragon/kore-server/nats"
	"github.com/twodragon/kore-server/server"
	"github.com/twodragon/kore-server/utils"

	"github.com/thoas/go-funk"
	"gopkg.in/guregu/null.v3"
)

type (
	GetGoldHandler struct {
	}

	GetInventoryHandler              struct{}
	ReplaceItemHandler               struct{}
	SwitchWeaponHandler              struct{}
	SwapItemsHandler                 struct{}
	RemoveItemHandler                struct{}
	DestroyItemHandler               struct{}
	RemoveBoxFromOpener              struct{}
	CombineItemsHandler              struct{}
	ArrangeInventoryHandler          struct{}
	ArrangeBankHandler               struct{}
	DepositHandler                   struct{}
	WithdrawHandler                  struct{}
	OpenHTMenuHandler                struct{}
	CloseHTMenuHandler               struct{}
	BuyHTItemHandler                 struct{}
	ReplaceHTItemHandler             struct{}
	DiscriminateItemHandler          struct{}
	InitDiscrimination               struct{}
	InspectItemHandler               struct{}
	DressUpHandler                   struct{}
	SplitItemHandler                 struct{}
	HolyWaterUpgradeHandler          struct{}
	MaterialDestructionMenuHandler   struct{}
	MaterialDestructionHandler       struct{}
	MaterialDestructionHandlerCancel struct{}
	UseConsumableHandler             struct{}
	OpenBoxHandler                   struct{}
	OpenBoxHandler2                  struct{}
	ActivateTimeLimitedItemHandler   struct{}
	ActivateTimeLimitedItemHandler2  struct{}
	ToggleMountPetHandler            struct{}
	TogglePetHandler                 struct{}
	PetCombatModeHandler             struct{}
	FireworkHandler                  struct{}
	EnchantBookHandler               struct{}
	SaveMapBookHandler               struct{}
	TeleportMapBookHandler           struct{}
	PetMoveHandler                   struct{}
	PetSkillAttackHandler            struct{}
	PetAttackHandler                 struct{}
)

var (
	GET_GOLD       = utils.Packet{0xAA, 0x55, 0x0A, 0x00, 0x57, 0x0B, 0x55, 0xAA}
	ITEMS_COMBINED = utils.Packet{0xAA, 0x55, 0x10, 0x00, 0x59, 0x06, 0x0A, 0x00, 0x00, 0x00, 0x55, 0xAA}
	ARRANGE_ITEM   = utils.Packet{0xAA, 0x55, 0x32, 0x00, 0x78, 0x02, 0x00, 0xA2, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}

	ARRANGE_BANK_ITEM = utils.Packet{0xAA, 0x55, 0x2F, 0x00, 0x80, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}

	CLOSE_HT_MENU = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x64, 0x02, 0x0A, 0x00, 0x55, 0xAA}
	OPEN_HT_MENU  = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x64, 0x01, 0x0A, 0x00, 0x55, 0xAA}
	GET_CASH      = utils.Packet{0xAA, 0x55, 0x0C, 0x00, 0x64, 0x03, 0x0A, 0x00, 0x55, 0xAA}
	BUY_HT_ITEM   = utils.Packet{0xAA, 0x55, 0x38, 0x00, 0x64, 0x04, 0x0A, 0x00, 0x07, 0xA2, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}
	REPLACE_HT_ITEM = utils.Packet{0xAA, 0x55, 0x0A, 0x00, 0x59, 0x40, 0x0A, 0x00, 0x55, 0xAA}
	HT_VISIBILITY   = utils.Packet{0xAA, 0x55, 0x06, 0x00, 0x59, 0x11, 0x0A, 0x00, 0x01, 0x00, 0x55, 0xAA}
	PET_COMBAT      = utils.Packet{0xAA, 0x55, 0x06, 0x00, 0x51, 0x05, 0x0a, 0x00, 0x00, 0x55, 0xAA}
	htShopQuantites = map[int64]uint{17100004: 40, 17100005: 40, 15900001: 50}

	BuyHTitemMutex sync.Mutex
)

func (h *DestroyItemHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	index := 10
	slotID := int16(utils.BytesToInt(data[index:index+2], true)) // slot

	resp, err := s.Character.RemoveItem(slotID)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (ggh *GetGoldHandler) Handle(s *database.Socket) ([]byte, error) {

	resp := GET_GOLD
	resp.Insert(utils.IntToBytes(uint64(s.Character.Gold), 8, true), 6) // gold
	resp.Insert(utils.IntToBytes(uint64(s.User.BankGold), 8, true), 14) // bank gold
	return resp, nil
}
func (gih *GetInventoryHandler) Handle(s *database.Socket) ([]byte, error) {

	inventory, err := s.Character.InventorySlots()
	if err != nil {
		return nil, err
	}

	resp := utils.Packet{}
	for i := 0; i < len(inventory); i++ {
		if i >= 67 && i <= 306 {
			continue
		}

		slot := inventory[i]
		resp.Concat(slot.GetData(int16(i), s.Character.ID))
	}

	if inventory[0x0A].ItemID > 0 { // pet
		resp.Concat(database.SHOW_PET_BUTTON)
	}

	if s.Character.DoesInventoryExpanded() {
		buffs := s.Character.FindExtentionBagBuff()
		buff := buffs[0]
		remainingTime := buff.StartedAt + buff.Duration - s.Character.Epoch
		expiration := null.NewTime(time.Now().Add(time.Second*time.Duration(remainingTime)), true) //ADD HOURS*/
		expbag := database.BAG_EXPANDED
		expbag.Overwrite([]byte(utils.ParseDate(expiration)), 7) // bag expiration
		resp.Concat(expbag)
	}

	return resp, nil
}

func (h *ReplaceItemHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s == nil || s.Character == nil {
		return nil, nil
	}

	itemID := int(utils.BytesToInt(data[6:10], true))
	where := int16(utils.BytesToInt(data[10:12], true))
	to := int16(utils.BytesToInt(data[12:14], true))

	resp, err := s.Character.ReplaceItem(itemID, where, to)

	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (h *SwitchWeaponHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	slotID := data[6]
	s.Character.WeaponSlot = int(slotID)

	itemsData, err := s.Character.ShowItems()
	if err != nil {
		return nil, err
	}

	p := nats.CastPacket{CastNear: true, CharacterID: s.Character.ID, Type: nats.SHOW_ITEMS, Data: itemsData}
	if err = p.Cast(); err != nil {
		return nil, err
	}

	gsh := &GetStatsHandler{}
	statData, err := gsh.Handle(s)
	if err != nil {
		return nil, err
	}

	resp := utils.Packet{}
	resp.Concat(itemsData)
	resp.Concat(statData)

	return resp, nil
}

func (h *SwapItemsHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	index := 11
	where := int16(utils.BytesToInt(data[index:index+2], true))

	index += 2
	to := int16(utils.BytesToInt(data[index:index+2], true))
	if s.Character == nil { //
		return nil, nil
	}
	resp, err := s.Character.SwapItems(where, to)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (h *RemoveItemHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	index := 11
	slotID := int16(utils.BytesToInt(data[index:index+2], true)) // slot

	slots, err := s.Character.InventorySlots()
	if err != nil {
		return nil, err
	}
	item := slots[slotID]
	if item.ItemID == 99059990 || item.ItemID == 99059991 || item.ItemID == 99059992 || item.ItemID == 99059993 || item.ItemID == 99059994 {
		database.DropFlag(s.Character, item.ItemID)
	}

	resp, err := s.Character.RemoveItem(slotID)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (h *CombineItemsHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	c := s.Character
	if c == nil {
		return nil, nil
	}

	where := int16(utils.BytesToInt(data[6:8], true))
	to := int16(utils.BytesToInt(data[8:10], true))

	resp := ITEMS_COMBINED

	invSlots, _ := c.InventorySlots()

	whereItem := invSlots[where]
	toItem := invSlots[to]

	itemID, qty, _ := c.CombineItems(where, to)

	if whereItem.Plus >= 1 && whereItem.Plus != toItem.Plus {
		return resp, nil
	}

	resp.Insert(utils.IntToBytes(uint64(itemID), 4, true), 8) // item id
	resp.Insert(utils.IntToBytes(uint64(where), 2, true), 12) // where slot
	resp.Insert(utils.IntToBytes(uint64(to), 2, true), 16)    // to slot
	resp.Insert(utils.IntToBytes(uint64(qty), 2, true), 18)   // item quantity

	c.Socket.Write(resp)

	c.Socket.Write(toItem.GetData(toItem.SlotID, c.ID))
	c.Socket.Write(whereItem.GetData(whereItem.SlotID, c.ID))

	return nil, nil
}

func (h *ArrangeInventoryHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	if s.Character.TradeID != "" || database.FindSale(s.Character.PseudoID) != nil {
		return nil, nil
	}
	user := s.User
	if user == nil {
		return nil, nil
	}

	return s.Character.ArrangeInventoryItems()
}

func (h *ArrangeBankHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	c := s.Character
	if c.TradeID != "" {
		return nil, nil
	}
	if c.TypeOfBankOpened != 1 {
		return nil, nil
	}

	user := s.User
	if user == nil {
		return nil, nil
	}

	return s.Character.ArrangeBankItems()
}
func (h *DepositHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	u := s.User
	if u == nil {
		return nil, nil
	}

	c := s.Character
	if c == nil {
		return nil, nil
	}
	if c.TradeID != "" {
		text := "Name: " + c.Name + "(" + c.UserID + ") tried to deposit items while trading."
		utils.NewLog("logs/cheat_alert.txt", text)
		return nil, fmt.Errorf("Cannot deposit gold while trading")
	}

	gold := uint64(utils.BytesToInt(data[6:14], true))
	if s.Character.Gold >= gold {
		if gold < 0 {
			gold = 0
		}
		if !s.Character.SubtractGold(uint64(gold)) {
			return nil, nil
		}
		u.BankGold += gold
		u.Update()
		return c.GetGold(), nil
	}

	return nil, nil
}

func (h *WithdrawHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	u := s.User
	if u == nil {
		return nil, nil
	}

	c := s.Character
	if c == nil {
		return nil, nil
	}

	gold := uint64(utils.BytesToInt(data[6:14], true))
	if u.BankGold >= gold {
		if gold < 0 {
			gold = 0
		}
		c.LootGold(gold)
		u.BankGold -= gold

		u.Update()
		return c.GetGold(), nil
	}
	return nil, nil
}

func (h *OpenHTMenuHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	u := s.User
	if u == nil {
		return nil, nil
	}

	resp := OPEN_HT_MENU
	r := GET_CASH
	r.Insert(utils.IntToBytes(u.NCash, 8, true), 8) // user nCash

	resp.Concat(r)
	return resp, nil
}

func (h *CloseHTMenuHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	return CLOSE_HT_MENU, nil
}

func (h *BuyHTItemHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	BuyHTitemMutex.Lock()
	defer BuyHTitemMutex.Unlock()

	itemID := int64(utils.BytesToInt(data[6:10], true))
	slotID := utils.BytesToInt(data[12:14], true)

	if item, ok := database.TavernItems[int(itemID)]; ok && item.IsActive && s.User.NCash >= uint64(item.Price) {
		s.User.NCash -= uint64(item.Price)

		info, ok := database.GetItemInfo(itemID)
		if !ok {
			return nil, nil
		}
		quantity := uint(1)
		if info.Timer > 0 && info.TimerType > 0 {
			quantity = uint(info.Timer)
		} else if qty, ok := htShopQuantites[int64(info.ID)]; ok {
			quantity = qty
		}

		item := &database.InventorySlot{ItemID: itemID, Quantity: quantity}
		if info.GetType() == database.PET_TYPE {
			petInfo := database.Pets[itemID]
			petExpInfo := database.PetExps[petInfo.Level-1]
			targetExps := []int{petExpInfo.ReqExpEvo1, petExpInfo.ReqExpEvo2, petExpInfo.ReqExpEvo3, petExpInfo.ReqExpHt, petExpInfo.ReqExpDivEvo1, petExpInfo.ReqExpDivEvo2, petExpInfo.ReqExpDivEvo3, petExpInfo.ReqExpDivEvo4, petExpInfo.ReqExpDivEvo4}
			item.Pet = &database.PetSlot{
				Fullness: 100, Loyalty: 100,
				Exp:   float64(targetExps[petInfo.Evolution-1]),
				HP:    petInfo.BaseHP,
				Level: byte(petInfo.Level),
				Name:  petInfo.Name,
				CHI:   petInfo.BaseChi}
		}

		r, _, err := s.Character.AddItem(item, int16(slotID), false)
		if err != nil {
			return nil, err
		} else if r == nil {
			return nil, nil
		}

		resp := BUY_HT_ITEM
		resp.Insert(utils.IntToBytes(uint64(itemID), 4, true), 8)    // item id
		resp.Insert(utils.IntToBytes(uint64(quantity), 2, true), 14) // item quantity
		resp.Insert(utils.IntToBytes(uint64(slotID), 2, true), 16)   // slot id
		resp.Insert(utils.IntToBytes(s.User.NCash, 8, true), 52)     // user nCash

		resp.Concat(*r)

		iteminfo, _ := database.GetItemInfo(item.ItemID)
		text := fmt.Sprintf("Name: "+s.Character.Name+"("+s.Character.UserID+") bought: "+iteminfo.Name+" for (%d)ncash.", database.TavernItems[int(itemID)].Price)
		utils.NewLog("logs/ncash_store_log.txt", text)

		s.User.Update()
		return resp, nil
	}

	return nil, nil
}

func (h *ReplaceHTItemHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	s.Character.AntiDupeMutex.Lock()
	defer s.Character.AntiDupeMutex.Unlock()

	slots, err := s.Character.InventorySlots()
	if err != nil {
		return nil, err
	}

	itemID := int(utils.BytesToInt(data[6:10], true))
	where := int16(utils.BytesToInt(data[10:12], true))

	resp := REPLACE_HT_ITEM
	resp.Insert(utils.IntToBytes(uint64(itemID), 4, true), 8) // item id
	resp.Insert(utils.IntToBytes(uint64(where), 2, true), 12) // where slot id

	quantity := slots[where].Quantity

	r := database.ITEM_SLOT
	r.Insert(utils.IntToBytes(uint64(itemID), 4, true), 6)    // item id
	r.Insert(utils.IntToBytes(uint64(quantity), 2, true), 12) // item quantity
	r.Insert(utils.IntToBytes(uint64(where), 2, true), 14)    // where slot id
	resp.Concat(r)

	item := &database.InventorySlot{ItemID: slots[where].ItemID, Quantity: slots[where].Quantity}

	info, ok := database.GetItemInfo(item.ItemID)
	if !ok {
		return nil, nil
	}

	if info.GetType() == database.PET_TYPE {
		petInfo := database.Pets[item.ItemID]
		petExpInfo := database.PetExps[petInfo.Level-1]
		targetExps := []int{petExpInfo.ReqExpEvo1, petExpInfo.ReqExpEvo2, petExpInfo.ReqExpEvo3, petExpInfo.ReqExpHt, petExpInfo.ReqExpDivEvo1, petExpInfo.ReqExpDivEvo2, petExpInfo.ReqExpDivEvo3, petExpInfo.ReqExpDivEvo4, petExpInfo.ReqExpDivEvo4}
		item.Pet = &database.PetSlot{
			Fullness: 100, Loyalty: 100,
			Exp:   float64(targetExps[petInfo.Evolution-1]),
			HP:    petInfo.BaseHP,
			Level: byte(petInfo.Level),
			Name:  petInfo.Name,
			CHI:   petInfo.BaseChi}
	}

	slotID, err := s.Character.FindFreeSlot()
	if err != nil {
		return nil, err
	} else if slotID == -1 { // no free slot

		return messaging.InfoMessage("Inventory full!"), nil
	}

	removedata, err := s.Character.RemoveItem(where)
	if err != nil {
		return nil, err
	}

	additem, id, err := s.Character.AddItem(item, slotID, false)
	if err != nil {
		return nil, err
	}
	if id == -1 {
		return messaging.InfoMessage(fmt.Sprintf("You don't have enough space in inventory")), nil
	}

	resp.Concat(removedata)
	resp.Concat(*additem)
	resp.Concat(slots[id].GetData(id, s.Character.ID))
	return resp, nil
}

func (h *InspectItemHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	if s.User.UserType < server.GM_USER {
		return nil, nil
	}
	resp := utils.Packet{0xaa, 0x55, 0xf1, 0x02, 0x62, 0x01, 0x0a, 0x00, 0x11, 0x55, 0xaa}
	pseudoID := uint16(utils.BytesToInt(data[6:8], true))
	index := 9
	char := database.FindCharacterByPseudoID(s.User.ConnectedServer, pseudoID)
	slots := char.GetEquipedItemSlots()
	inventory, err := char.InventorySlots()
	if err != nil {
		return nil, err
	}

	for _, s := range slots {
		slot := inventory[s]
		//id := utils.IntToBytes(uint64(s), 2, true)
		plus := 161 + slot.Plus
		resp.Insert(utils.IntToBytes(uint64(slot.ItemID), 4, true), index) // item id
		index += 4
		resp.Insert(utils.IntToBytes(uint64(plus), 2, true), index)
		index += 2
		resp.Insert([]byte{0x01, 0x00}, index) //ELVILEG UGYAN AZ MINDIG
		index += 2
		resp.Insert(utils.IntToBytes(uint64(s), 2, true), index) // item slot
		index += 2
		resp.Insert(slot.GetUpgrades(), index) // item plus
		index += 15
		resp.Insert([]byte{byte(slot.SocketCount)}, index)
		index++
		resp.Insert(slot.GetSockets(), index) // SOCKET
		index += 15
		resp.Insert([]byte{0x00, 0x00, 0x00}, index)
		index += 3
	}

	return resp, nil
}

func (h *DressUpHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if len(data) < 6 {
		return nil, nil
	}
	isHT := data[6] == 1
	if isHT {
		s.Character.HTVisibility = int(data[7])
		resp := HT_VISIBILITY
		resp[9] = data[7]

		itemsData, err := s.Character.ShowItems()
		if err != nil {
			return nil, err
		}

		p := nats.CastPacket{CastNear: true, CharacterID: s.Character.ID, Type: nats.SHOW_ITEMS, Data: itemsData}
		if err = p.Cast(); err != nil {
			return nil, err
		}

		resp.Concat(itemsData)

		return resp, nil
	}

	return nil, nil
}

func (h *SplitItemHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	where := uint16(utils.BytesToInt(data[6:8], true))
	to := uint16(utils.BytesToInt(data[8:10], true))
	quantity := uint16(utils.BytesToInt(data[10:12], true))

	return s.Character.SplitItem(where, to, quantity)
}

func (h *HolyWaterUpgradeHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	slots, err := s.Character.InventorySlots()
	if err != nil {
		return nil, err
	}

	itemSlot := int16(utils.BytesToInt(data[6:8], true))
	item := slots[itemSlot]
	if itemSlot == 0 || item.ItemID == 0 {
		return nil, nil
	}

	holyWaterSlot := int16(utils.BytesToInt(data[8:10], true))
	holyWater := slots[holyWaterSlot]
	if holyWaterSlot == 0 || holyWater.ItemID == 0 {
		return nil, nil
	}

	return s.Character.HolyWaterUpgrade(item, holyWater, itemSlot, holyWaterSlot)
}
func (h *MaterialDestructionMenuHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	return utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x54, 0x13, 0x0A, 0x00, 0x55, 0xAA}, nil
}
func (h *MaterialDestructionHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	resp := utils.Packet{}
	lenght := utils.BytesToInt(data[6:7], true)
	items := []*database.InventorySlot{}
	slots, err := s.Character.InventorySlots()
	if err != nil {
		return nil, err
	}

	index := 7
	for i := 1; i <= int(lenght); i++ {
		itemSlot := utils.BytesToInt(data[index:index+2], true)
		index += 2
		item := slots[itemSlot]
		items = append(items, item)
	}
	for _, item := range items {

		dismantleData, success, err := s.Character.Dismantle(item, nil, true)
		if err != nil {
			continue
		}
		if !success {
			continue
		}
		resp.Concat(dismantleData)

	}
	return resp, nil
}
func (h *MaterialDestructionHandlerCancel) Handle(s *database.Socket, data []byte) ([]byte, error) {
	resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x54, 0x14, 0x0A, 0x00, 0x55, 0xAA}

	return resp, nil
}

func (h *UseConsumableHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	slots, err := s.Character.InventorySlots()
	if err != nil {
		return nil, err
	}

	itemID := int64(utils.BytesToInt(data[6:10], true))
	slotID := int16(utils.BytesToInt(data[10:12], true))

	item := slots[slotID]
	if item == nil || int64(item.ItemID) != itemID {
		return nil, nil
	}

	return s.Character.UseConsumable(item, slotID)
}

func (h *OpenBoxHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	slots, err := s.Character.InventorySlots()
	if err != nil {
		return nil, err
	}

	slotID := int16(utils.BytesToInt(data[6:8], true))

	item := slots[slotID]
	if item == nil {
		return nil, nil
	}

	return s.Character.UseConsumable(item, slotID)
}

func (h *OpenBoxHandler2) Handle(s *database.Socket, data []byte) ([]byte, error) {

	slots, err := s.Character.InventorySlots()
	if err != nil {
		return nil, err
	}

	itemID := utils.BytesToInt(data[6:10], true)
	slotID := int16(utils.BytesToInt(data[10:12], true))

	item := slots[slotID]
	if item == nil || int64(item.ItemID) != itemID {
		return nil, nil
	}

	return s.Character.UseConsumable(item, slotID)
}

func (h *ActivateTimeLimitedItemHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	slots, err := s.Character.InventorySlots()
	if err != nil {
		return nil, err
	}

	slotID := int16(utils.BytesToInt(data[6:8], true))
	item := slots[slotID]
	if item == nil {
		return nil, nil
	}

	info, _ := database.GetItemInfo(item.ItemID)
	if info == nil || info.Timer == 0 {
		return nil, nil
	}

	hasSameBuff := len(funk.Filter(slots, func(slot *database.InventorySlot) bool {
		slotiteminfo, ok := database.GetItemInfo(slot.ItemID)
		if ok && slotiteminfo != nil {
			if slot.Activated && (slotiteminfo.UIF == info.UIF) {
				return true
			}
		}

		return slot.Activated && slot.ItemID == item.ItemID
	}).([]*database.InventorySlot)) > 0

	if hasSameBuff {
		return []byte{0xAA, 0x55, 0x04, 0x00, 0x59, 0x04, 0xFA, 0x03, 0x55, 0xAA}, nil
	}

	resp := utils.Packet{}
	item.Activated = !item.Activated
	item.InUse = !item.InUse
	resp.Concat(item.GetData(slotID))

	item.Update()
	statsData, _ := s.Character.GetStats()
	resp.Concat(statsData)
	return resp, nil
}

func (h *ActivateTimeLimitedItemHandler2) Handle(s *database.Socket, data []byte) ([]byte, error) {

	slots, err := s.Character.InventorySlots()
	if err != nil {
		return nil, err
	}

	where := int16(utils.BytesToInt(data[6:8], true))
	itemID := utils.BytesToInt(data[8:12], true)
	to := int16(utils.BytesToInt(data[12:14], true))

	item := slots[where]
	if item == nil || int64(item.ItemID) != itemID {
		return nil, nil
	}

	info, _ := database.GetItemInfo(item.ItemID)
	if info == nil || info.Timer == 0 {
		return nil, nil
	}

	hasSameBuff := len(funk.Filter(slots, func(slot *database.InventorySlot) bool {
		return slot.Activated && slot.ItemID == item.ItemID
	}).([]*database.InventorySlot)) > 0

	if hasSameBuff {
		return []byte{0xAA, 0x55, 0x04, 0x00, 0x59, 0x04, 0xFA, 0x03, 0x55, 0xAA}, nil
	}

	resp := utils.Packet{}
	item.Activated = !item.Activated
	item.InUse = !item.InUse
	s.Write(item.GetData(where))

	var itemData utils.Packet
	if slots[to].ItemID == 0 {
		itemData, err = s.Character.ReplaceItem(int(itemID), where, to)
	} else {
		itemData, err = s.Character.SwapItems(where, to)
	}

	if err != nil {
		return nil, err
	}
	resp.Concat(itemData)

	item.Update()
	statsData, _ := s.Character.GetStats()
	resp.Concat(statsData)
	return resp, nil
}
func (h *ToggleMountPetHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	return s.Character.TogglePet(), nil
}

func (h *TogglePetHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	return s.Character.TogglePet(), nil
}

func (h *PetCombatModeHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	CombatMode := utils.BytesToInt(data[7:8], true)
	slots, err := s.Character.InventorySlots()
	if err != nil {
		log.Println(err)
		return nil, nil
	}
	petSlot := slots[0x0A]
	pet := petSlot.Pet
	if pet == nil || petSlot.ItemID == 0 || !pet.IsOnline {
		return nil, nil
	}
	pet.PetCombatMode = int16(CombatMode)
	resp := PET_COMBAT
	resp.Insert(utils.IntToBytes(uint64(CombatMode), 1, true), 9)
	return resp, nil
}

func (h *PetMoveHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	cX := utils.BytesToFloat(data[8:12], true)
	cY := utils.BytesToFloat(data[12:16], true)
	tX := utils.BytesToFloat(data[20:24], true)
	tY := utils.BytesToFloat(data[24:28], true)
	currentLocation := &utils.Location{X: cX, Y: cY}
	targetLocation := &utils.Location{X: tX, Y: tY}
	slots, err := s.Character.InventorySlots()
	if err != nil {
		log.Println(err)
		return nil, nil
	}
	petSlot := slots[0x0A]
	pet := petSlot.Pet
	if pet == nil || petSlot.ItemID == 0 || !pet.IsOnline {
		return nil, nil
	}
	pet.IsMoving = true
	pet.TargetLocation = *targetLocation
	speed := s.Character.RunningSpeed
	token := pet.MovementToken
	for token == pet.MovementToken {
		pet.MovementToken = utils.RandInt(1, math.MaxInt64)
	}
	go pet.MovementHandler(pet.MovementToken, currentLocation, targetLocation, speed)
	return nil, nil
}

func (h *PetSkillAttackHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	aiID := uint16(utils.BytesToInt(data[7:9], true))

	slots, err := s.Character.InventorySlots()
	if err != nil {
		log.Println(err)
		return nil, nil
	}

	petSlot := slots[0x0A]
	pet := petSlot.Pet
	if pet == nil || petSlot.ItemID == 0 || !pet.IsOnline {
		return nil, nil
	}
	pet.Target = int(aiID)
	petInfo, ok := database.Pets[petSlot.ItemID]
	if !ok {
		return nil, nil
	}
	skillIds := []int{petInfo.Skill_1, petInfo.Skill_2, petInfo.Skill_3}
	randomSkill := utils.RandInt(0, int64(len(skillIds)-1))
	skillID := skillIds[randomSkill]

	return pet.CastSkill(s.Character, skillID), nil
}

func (h *PetAttackHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	//log.Print(data)
	aiID := uint16(utils.BytesToInt(data[7:9], true))

	slots, err := s.Character.InventorySlots()
	if err != nil {
		log.Println(err)
		return nil, nil
	}

	petSlot := slots[0x0A]
	pet := petSlot.Pet
	if pet == nil || petSlot.ItemID == 0 || !pet.IsOnline {
		return nil, nil
	}
	pet.Target = int(aiID)

	return pet.Attack(s.Character), nil
}

func (h *DiscriminateItemHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	slots, err := s.Character.InventorySlots()
	if err != nil {
		resp := messaging.SystemMessage(messaging.INSUFFICIENT_GOLD)
		return resp, nil
	}

	slotID := int(utils.BytesToInt(data[6:8], true)) //max: 754700
	item := slots[slotID]
	if item.ItemType != 1 {
		return nil, nil
	}
	itemstat, ok := database.GetItemInfo(item.ItemID)
	if itemstat == nil || !ok {
		return nil, nil
	}
	discprice := itemstat.MinLevel * 5000
	if s.Character.Gold < uint64(discprice) {
		return nil, nil
	}
	index := 0
	seed := int(utils.RandInt(0, 754700))
	for _, prob := range database.ItemJudgements {
		if prob.Probabilities >= seed {
			index = prob.ID
			break
		}
	}
	if index != 0 {

		resp := utils.Packet{}
		if !s.Character.SubtractGold(uint64(discprice)) {
			return nil, nil
		}
		slot := slots[slotID]
		slot.ItemType = 2
		slot.JudgementStat = int64(index)
		err = slot.Update()
		if err != nil {
			return nil, nil
		}
		database.InventoryItems.Add(slot.ID, slot)
		resp.Concat(slot.GetData(int16(slotID)))
		return resp, nil
	}
	return nil, nil
}

func (h *InitDiscrimination) Handle(s *database.Socket, data []byte) ([]byte, error) {
	slotID := int16(utils.BytesToInt(data[6:8], true))
	initslotID := int16(utils.BytesToInt(data[8:10], true))

	slots, err := s.Character.InventorySlots()
	if err != nil {
		resp := messaging.SystemMessage(messaging.INSUFFICIENT_GOLD)
		return resp, nil
	}

	item := slots[slotID]
	if item.ItemType != 2 {
		return nil, nil
	}

	initdiscitem := slots[initslotID]
	iteminfo, ok := database.GetItemInfo(initdiscitem.ItemID)
	if !ok {
		return nil, nil
	}
	if iteminfo.Type != 45 {
		return nil, nil
	}

	itemsData, err := s.Character.RemoveItem(int16(initslotID))
	if err != nil {
		return nil, err
	}

	item.ItemType = 1
	item.JudgementStat = 0

	err = item.Update()
	if err != nil {
		return nil, nil
	}
	resp := utils.Packet{}
	resp.Concat(item.GetData(int16(slotID)))
	resp.Concat(itemsData)

	return resp, nil
}

func (h *RemoveBoxFromOpener) Handle(s *database.Socket, data []byte) ([]byte, error) {

	c := s.Character

	where := int16(utils.BytesToInt(data[10:11], true))

	to := int16(utils.BytesToInt(data[11:13], true))

	slots, _ := c.InventorySlots()

	toItem := slots[to]
	if toItem.ItemID != 0 {
		return nil, nil
	}
	whereItem := slots[where+402]
	if whereItem.ItemID == 0 {
		return nil, nil
	}

	whereItem.SlotID = to
	whereItem.CharacterID = null.IntFrom(int64(c.ID))
	*toItem = *whereItem
	*whereItem = *database.NewSlot()

	toItem.Update()
	database.InventoryItems.Add(toItem.ID, toItem)

	c.Socket.Write(toItem.GetData(toItem.SlotID))

	return nil, nil
}

func (h *FireworkHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	slotID, item, err := s.Character.FindItemInInventory(nil,
		3000040,
		13000241,
		16210019,
		16210020,
		17300008,
		17300009,
		17504859,
		17507515,
		17509032,
		17509049,
		92001033,
		92001065,
		92001110,
		99002335,
	)
	if err != nil || item == nil {
		return nil, err
	}
	if slotID == -1 {
		return nil, nil
	}

	resp := utils.Packet{0xAA, 0x55, 0x9B, 0x00, 0x72, 0x09, 0x55, 0xAA}
	coordinate := database.ConvertPointToLocation(s.Character.Coordinate)

	//itemID := utils.BytesToInt(data[6:10], true)
	//slotID := utils.BytesToInt(data[10:12], true)

	rotation := funk.RandomInt(0, 360)

	index := 6
	resp.Insert(utils.IntToBytes(uint64(rotation), 2, true), index) // coordinate-x
	index += 2
	resp.Insert(utils.FloatToBytes(coordinate.X, 4, true), index) // coordinate-x
	index += 4
	resp.Insert(utils.FloatToBytes(coordinate.Y, 4, true), index) // coordinate-y
	index += 4

	data = data[6 : len(data)-2]
	resp.Insert(data, index)
	resp.SetLength(int16(len(resp)) - 6)

	r := s.Character.DecrementItem(slotID, 1)
	s.Write(*r)

	p := nats.CastPacket{CastNear: true, CharacterID: s.Character.ID, Data: resp}
	if err := p.Cast(); err == nil {
		s.Write(resp)
	}

	return resp, nil
}

func (h *EnchantBookHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	slots, err := s.Character.InventorySlots()
	if err != nil {
		return nil, err
	}

	bookID := int64(utils.BytesToInt(data[6:10], true))

	ReqItemsAmount := int16(utils.BytesToInt(data[10:11], true))
	matsSlotsIds := []int16{}
	matsIds := []int64{}

	index := 11
	for i := 0; i < int(ReqItemsAmount); i++ {
		slotid := int(utils.BytesToInt(data[index:index+2], true))
		index += 2
		matsSlotsIds = append(matsSlotsIds, int16(slotid))
		matsIds = append(matsIds, int64(slots[slotid].ItemID))
	}

	resp := utils.Packet{}
	prodData, err := s.Character.Enchant(bookID, matsSlotsIds, matsIds)
	if err != nil {
		return nil, err
	}
	resp.Concat(prodData)
	return resp, nil

}

func (h *SaveMapBookHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	slot := int(data[6]) + 1
	resp := utils.Packet{}
	resp = data
	if s.Character.Map == 249 || s.Character.Map == 255 {
		return nil, nil
	}
	teleports, err := database.FindTeleportsByID(s.Character.ID)
	if err != nil {
		return nil, err
	}

	teleportSlots, err := teleports.GetTeleports()
	if err != nil {
		return nil, err
	}
	coordinate := database.ConvertPointToLocation(s.Character.Coordinate)
	if teleportSlots.Slots[slot-1].Teleportslots == nil {
		set := &database.TeleportSet{}
		set.Teleportslots = append(set.Teleportslots, &database.SlotsTuple{SlotID: slot, MapID: int(s.Character.Map), Coordx: int(coordinate.X), Coordy: int(coordinate.Y)})
		teleportSlots.Slots[slot-1] = set
		teleports.SetTeleports(teleportSlots)
	} else {
		sete := teleportSlots.Slots[slot-1]
		sete.Teleportslots[0].MapID = int(s.Character.Map)
		sete.Teleportslots[0].Coordx = int(coordinate.X)
		sete.Teleportslots[0].Coordy = int(coordinate.Y)
		teleports.SetTeleports(teleportSlots)
	}
	go teleports.Update()
	gih := &GetInventoryHandler{}
	inventory, err := gih.Handle(s)
	if err != nil {
		return nil, err
	}

	s.Write(inventory)
	resp.Insert([]byte{0x0a, 0x00}, 6)
	resp.SetLength(int16(binary.Size(resp) - 6))
	return resp, nil
}
func (h *TeleportMapBookHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.Character == nil {
		return nil, nil
	}

	resp := utils.Packet{}
	resp = data
	resp.Insert([]byte{0x0a, 0x00}, 6)
	resp.SetLength(int16(binary.Size(resp) - 6))

	if s.User.MapBookCooldown > 0 {
		if s.User.MapBookCooldown > 60 {
			resp.Concat(messaging.InfoMessage("You cannot teleport for another " + strconv.Itoa(int(s.User.MapBookCooldown/60)) + " minutes."))
		} else {
			resp.Concat(messaging.InfoMessage("You cannot teleport for another " + strconv.Itoa(int(s.User.MapBookCooldown)) + " seconds."))
		}
		return resp, nil
	}

	slot := int(data[6]) + 1

	teleports, err := database.FindTeleportsByID(s.Character.ID)
	if err != nil {
		return nil, err
	}

	teleportSlots, err := teleports.GetTeleports()
	if err != nil {
		return nil, err
	}
	set := teleportSlots.Slots[slot-1].Teleportslots[0]
	if int(s.Character.Map) != set.MapID {
		mapID, _ := s.Character.ChangeMap(int16(set.MapID), database.ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", float64(set.Coordx), float64(set.Coordy))))
		resp.Concat(mapID)
	}
	teleportresp := s.Character.Teleport(database.ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", float64(set.Coordx), float64(set.Coordy))))
	resp.Concat(teleportresp)

	s.User.MapBookCooldown = 300
	go func() {
		for s.User.MapBookCooldown > 0 {
			time.Sleep(time.Second)
			s.User.MapBookCooldown--
		}
	}()

	return resp, nil
}
