package player

import (
	"encoding/binary"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/osamingo/boolconv"
	"github.com/thoas/go-funk"
	"github.com/twodragon/kore-server/database"
	"github.com/twodragon/kore-server/messaging"
	"github.com/twodragon/kore-server/nats"

	"github.com/twodragon/kore-server/utils"
	"gopkg.in/guregu/null.v3"
)

type (
	BattleModeHandler        struct{}
	MeditationHandler        struct{}
	TargetSelectionHandler   struct{}
	TravelToCastleHandler    struct{}
	OpenTacticalSpaceHandler struct{}
	TacticalSpaceTPHandler   struct{}
	InTacticalSpaceTPHandler struct{}
	OpenLotHandler           struct{}
	QuestHandler             struct{}
	EnterGateHandler         struct{}
	SendPvPRequestHandler    struct{}
	RespondPvPRequestHandler struct{}
	TransferSoulHandler      struct{}
	CharmOfIdentity          struct{}
	StyleHandler             struct{}
	TravelToFiveClanArea     struct{}
	TransferItemTypeHandler  struct{}

	EnhancementTransfer    struct{}
	ClothImproveChest      struct{}
	ChangePetName          struct{}
	ChangePartyModeHandler struct{}
	CtrlClick              struct{}
	SendReport             struct{}

	DeclineOmokRequestHandler struct{}
	AcceptSoulHandler         struct{}
	FinishSoulHandler         struct{}

	HouseItemInteract  struct{}
	RemoveHouseItem    struct{}
	AllowHouseOutsider struct{}
	GatherCropHandler  struct{}
	HarvestCropHandler struct{}

	PlaceHouseItem struct {
		CoordX float64
		CoordZ float64
		CoordY float64
	}
	AddNewFriend struct{}
	RemoveFriend struct{}

	HireAdventurer         struct{}
	CancelEmployment       struct{}
	ReceiveAdventurerItems struct{}
	HeartBeat              struct{}
)

var (
	FreeLotQuantities = map[int]int{10820001: 5, 10600033: 10, 10600036: 10, 17500346: 5, 10600057: 5}
	PaidLotQuantities = map[int]int{92000001: 5, 92000011: 5, 10820001: 5, 17500346: 10, 10601023: 20, 10601024: 20, 10601007: 50, 10601008: 50, 10600057: 10,
		17502966: 5, 17502967: 5, 243: 3}

	BATTLE_MODE         = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x43, 0x00, 0x55, 0xAA}
	MEDITATION_MODE     = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x82, 0x05, 0x00, 0x55, 0xAA}
	TACTICAL_SPACE_MENU = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x50, 0x01, 0x01, 0x55, 0xAA, 0xAA, 0x55, 0x05, 0x00, 0x28, 0xFF, 0x00, 0x00, 0x00, 0x55, 0xAA}
	TACTICAL_SPACE_TP   = utils.Packet{0xAA, 0x55, 0x07, 0x00, 0x01, 0xB9, 0x0A, 0x00, 0x00, 0x00, 0x01, 0x55, 0xAA}
	OPEN_LOT            = utils.Packet{0xAA, 0x55, 0x0C, 0x00, 0xA2, 0x01, 0x32, 0x00, 0x00, 0x00, 0x00, 0x01, 0x55, 0xAA}
	SELECTION_CHANGED   = utils.Packet{0xAA, 0x55, 0x09, 0x00, 0xCF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}
	PVP_REQUEST         = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x2A, 0x01, 0x55, 0xAA}
	PVP_STARTED         = utils.Packet{0xAA, 0x55, 0x0A, 0x00, 0x2A, 0x02, 0x55, 0xAA}
	CLANCASTLE_MAP      = utils.Packet{0xaa, 0x55, 0x62, 0x00, 0xbb, 0x03, 0x05, 0x55, 0xAA}
	CANNOT_MOVE         = utils.Packet{0xaa, 0x55, 0x04, 0x00, 0xbb, 0x02, 0x00, 0x00, 0x55, 0xaa}

	HOUSE_APPEAR = utils.Packet{0xAA, 0x55, 0x3A, 0x00, 0xAC, 0x03, 0x0A, 0x00, 0xD5, 0x1B, 0x01, 0x00, 0xD5, 0x1B, 0x01, 0x00, 0x90, 0x01,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}

	HOUSE_INFO = utils.Packet{0xAA, 0x55, 0x2D, 0x00, 0xAC, 0x06, 0x0A, 0x00, 0x43, 0x22, 0x01, 0x00, 0xF4, 0x01, 0x00, 0x00, 0x00, 0x13,
		0x32, 0x30, 0x32, 0x32, 0x2D, 0x30, 0x34, 0x2D, 0x31, 0x35, 0x20, 0x30, 0x31, 0x3A, 0x33, 0x38, 0x3A, 0x34, 0x37, 0x00, 0x80, 0xA5,
		0x43, 0xF8, 0xAA, 0xC0, 0x40, 0x00, 0x80, 0xA1, 0x43, 0x55, 0xAA}
)

func (h *HarvestCropHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	pseudoID := uint16(utils.BytesToInt(data[6:10], true))

	houseitem := database.FindHouseByPseudoId(int(pseudoID))
	if houseitem.OwnerID != s.Character.ID {
		return messaging.SystemMessage(53114), nil //no authorization
	}
	info, ok := database.HouseItemsInfos[int(houseitem.HouseID)]
	if !ok {
		return messaging.SystemMessage(53118), nil
	}
	if info.Category != 2 {
		return messaging.SystemMessage(53126), nil

	}
	if info.CanCollect == 0 {
		return messaging.SystemMessage(53124), nil
	}

	_, err := s.Character.FindFreeSlots(1)
	if err != nil {
		return messaging.InfoMessage("No free slots!"), nil
	}
	drop, ok := database.GetDropInfo(info.DropID)
	if drop == nil || !ok {
		return nil, nil
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
		drop, ok = database.GetDropInfo(itemID)
	}

	if itemID == 0 {
		return nil, nil
	}
	rewardInfo, ok := database.GetItemInfo(int64(itemID))
	if !ok {
		return nil, nil
	}

	item := &database.InventorySlot{ItemID: rewardInfo.ID, Quantity: 1}

	itemdata, _, err := s.Character.AddItem(item, -1, true)
	if err != nil {
		return nil, err
	}

	resp := utils.Packet{0xaa, 0x55, 0x0c, 0x00, 0xac, 0x09, 0x0a, 0x00, 0x2f, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xaa}
	resp.Overwrite(utils.IntToBytes(uint64(houseitem.PseudoID), 4, true), 8)

	r := utils.Packet{0xaa, 0x55, 0x08, 0x00, 0xac, 0x04, 0x0a, 0x00, 0x3e, 0x07, 0x01, 0x00, 0x55, 0xaa}
	r.Overwrite(utils.IntToBytes(uint64(houseitem.PseudoID), 4, true), 8)
	resp.Concat(r)
	resp.Concat(*itemdata)

	p := &nats.CastPacket{CastNear: true, CharacterID: houseitem.OwnerID, Data: resp, Type: nats.PLAYER_SPAWN}
	if err := p.Cast(); err != nil {
		return nil, nil
	}
	p.Cast()

	s.Character.OnSight.HousingitemsMutex.Lock()
	delete(s.Character.OnSight.Housingitems, houseitem.ID)
	s.Character.OnSight.HousingitemsMutex.Unlock()

	houseitem.Delete()

	database.HousingItemsMutex.Lock()
	delete(database.HousingItems, houseitem.ID)
	database.HousingItemsMutex.Unlock()

	s.Character.DoAnimation(0)

	return resp, nil
}
func (h *GatherCropHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	pseudoID := uint16(utils.BytesToInt(data[6:10], true))
	x := utils.BytesToFloat(data[10:14], true)
	y := utils.BytesToFloat(data[14:18], true)
	z := utils.BytesToFloat(data[18:24], true)

	houseitem := database.FindHouseByPseudoId(int(pseudoID))
	if houseitem.OwnerID != s.Character.ID {
		return messaging.SystemMessage(53114), nil //no authorization
	}

	resp := utils.Packet{0xaa, 0x55, 0x14, 0x00, 0xac, 0x07, 0x0a, 0x00, 0x30, 0x23, 0x01, 0x00, 0x2f, 0x03, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0xdb, 0x0f, 0x49, 0x40, 0x55, 0xaa}

	resp.Overwrite(utils.IntToBytes(uint64(houseitem.PseudoID), 4, true), 8)
	resp.Overwrite(utils.FloatToBytes((x), 4, true), 12)
	resp.Overwrite(utils.FloatToBytes((y), 4, true), 16)
	resp.Overwrite(utils.FloatToBytes((z), 4, true), 20)

	return resp, nil

}
func (h *HireAdventurer) Handle(s *database.Socket, data []byte) ([]byte, error) {

	id := int(utils.BytesToInt(data[6:8], true))

	count := 0
	for _, adv := range database.Adventurers {
		if adv.CharID == s.Character.ID && adv.Status == 1 {
			count++
		}
	}
	cost := uint64(400000)
	if id > 11 {
		cost += 200000
	}

	if s.Character.Gold < cost {
		return messaging.SystemMessage(2131), nil //insufucient funds
	}

	if count == 5 {
		return messaging.SystemMessage(53130), nil //you can't hire more than 5 adventurers
	}
	if count >= 3 {
		_, item, err := s.Character.FindItemInInventory(nil, 20050152)
		if err != nil {
			return nil, err
		}
		if item == nil {
			return messaging.SystemMessage(53128), nil //you need a scroll to hire an adventurer
		}
		data := s.Character.DecrementItem(item.SlotID, 10)
		s.Write(*data)
	}

	adv := &database.Adventurer{
		CharID:          s.Character.ID,
		Index:           id,
		FinishAt:        null.NewTime(time.Now().Add(time.Second*time.Duration(21600)), true),
		TotalAdventures: 0,
		Level:           0,
		Status:          1,
	}

	ok := false
	for _, adventurer := range database.Adventurers {
		if adventurer.Index == id && adventurer.CharID == s.Character.ID {
			adv = adventurer
			adv.FinishAt = null.NewTime(time.Now().Add(time.Second*time.Duration(21600)), true)
			adv.Status = 1
			adv.Update()
			ok = true
			break
		}
	}
	if !ok {
		adv.Create()
		database.Adventurers[adv.ID] = adv
	}
	adv.Init()
	resp := utils.Packet{}
	resp = adv.GetData()
	if !s.Character.SubtractGold(uint64(cost)) {
		return nil, nil
	}

	return resp, nil
}

func (h *CancelEmployment) Handle(s *database.Socket, data []byte) ([]byte, error) {
	id := int(utils.BytesToInt(data[6:8], true))

	for _, adv := range database.Adventurers {
		if adv.Index == id && adv.CharID == s.Character.ID && adv.Status == 1 {
			adv.Status = 0
			adv.Update()

			resp := adv.GetData()
			return resp, nil
		}
	}

	return nil, nil
}

func (h *ReceiveAdventurerItems) Handle(s *database.Socket, data []byte) ([]byte, error) {
	id := int(utils.BytesToInt(data[6:8], true))
	resp := utils.Packet{}
	dropid := 31420
	for _, adv := range database.Adventurers {
		if adv.Index == id && adv.CharID == s.Character.ID && adv.Status == 2 {
			adv.Status = 0
			adv.TotalAdventures++
			adv.Update()
			dropid += adv.Index - 1

			resp = adv.GetData()
			break
		}
	}
	drop, ok := database.GetDropInfo(dropid)
	if drop == nil || !ok {
		return nil, nil
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
		drop, ok = database.GetDropInfo(itemID)
	}
	item := &database.InventorySlot{ItemID: int64(itemID), Quantity: 1}
	itemdata, _, err := s.Character.AddItem(item, -1, true)
	if err != nil {
		return nil, err
	}
	resp.Concat(*itemdata)

	return resp, nil
}
func (h *AllowHouseOutsider) Handle(s *database.Socket, data []byte) ([]byte, error) {

	pseudoID := uint16(utils.BytesToInt(data[6:10], true))
	isPublic := int(utils.BytesToInt(data[14:15], true))

	resp := utils.Packet{0xAA, 0x55, 0x09, 0x00, 0xAC, 0x0b, 0x0a, 0x00, 0x3b, 0x07, 0x01, 0x00, 0x01, 0x55, 0xaa}

	if s.Character.House.PseudoID != pseudoID {
		return messaging.SystemMessage(53114), nil //no authorization
	}

	s.Character.House.IsPublic = isPublic
	s.Character.House.Update()

	index := 8
	resp.Overwrite(utils.IntToBytes(uint64(s.Character.ID), 4, true), index)
	index += 4
	resp.Overwrite(utils.IntToBytes(uint64(s.Character.House.IsPublic), 1, false), index)
	index++

	s.Character.HousingDetails()

	return nil, nil
}

func (h *HouseItemInteract) Handle(s *database.Socket, data []byte) ([]byte, error) {

	pseudoID := uint16(utils.BytesToInt(data[6:10], true))
	action := uint16(utils.BytesToInt(data[10:11], true))
	//housepseudoID := uint16(utils.BytesToInt(data[14:18], true))

	selectedItem := database.FindHouseByPseudoId(int(pseudoID))
	owner, _ := database.FindCharacterByID(selectedItem.OwnerID)

	info, ok := database.HouseItemsInfos[int(selectedItem.HouseID)]
	if !ok {
		return nil, nil
	}

	resp := utils.Packet{}

	switch action {
	case 1:
		if selectedItem.HouseID > 10000 {
			return nil, nil
		}
		if uint16(s.Character.House.PseudoID) != pseudoID && selectedItem.IsPublic == 0 {
			return messaging.SystemMessage(53114), nil //no authorization
		}
		s.User.ConnectedServer = owner.ID
		s.User.Update()
		r, err := s.Character.ChangeMap(database.HouseItemsInfos[selectedItem.HouseID].Map, nil)
		if err != nil {
			return nil, err
		}
		resp.Concat(r)
	case 2:

	case 3:
		switch selectedItem.HouseID {
		case 120000:
			resp = s.Character.OpenAdventurerMenu()
			return resp, nil
		case 120001:
			resp = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x3B, 0x55, 0xAA}
			resp.Insert(utils.IntToBytes(5, 1, true), 6)
			return resp, nil
		case 120002:
			resp = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x50, 0x01, 0x01, 0x55, 0xAA} //tactical space
			return resp, nil
		case 120003:
			resp = utils.Packet{0xaa, 0x55, 0x03, 0x00, 0x57, 0x3b, 0x11, 0x55, 0xaa} //make food
			return resp, nil
		}
		switch info.Type {
		case 3:
			resp = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x3B, 0x55, 0xAA}
			resp.Insert(utils.IntToBytes(13, 1, true), 6)
			if info.Relaxetion >= 800 && info.Relaxetion < 1400 {
				resp.Overwrite(utils.IntToBytes(14, 1, true), 6)
			} else if info.Relaxetion >= 1400 {
				resp.Overwrite(utils.IntToBytes(15, 1, true), 6)
			}
		}
	}

	return resp, nil
}

func (h *PlaceHouseItem) Handle(s *database.Socket, data []byte) ([]byte, error) {
	slotid := int(utils.BytesToInt(data[6:8], true))
	CoordX := utils.BytesToFloat(data[14:18], true)
	CoordZ := utils.BytesToFloat(data[18:22], true)
	CoordY := utils.BytesToFloat(data[22:26], true)

	invSlots, err := s.Character.InventorySlots()
	if err != nil {
		return nil, err
	}

	item := invSlots[slotid]
	info, ok := database.GetItemInfo(item.ItemID)
	if !ok {
		return nil, nil
	}

	houseid := int(info.MinUpgradeLevel)

	houseinfo, ok := database.HouseItemsInfos[houseid]
	if !ok {
		return nil, nil
	}
	for _, house := range database.HousingItems {
		if house.HouseID == houseid && house.OwnerID == s.Character.ID && int(house.Server) == s.User.ConnectedServer && house.MapID == s.Character.Map &&
			houseinfo.Category != 2 {
			return messaging.InfoMessage("House already has this item!"), nil
		}
		if house.MapID == s.Character.Map && house.Server == int16(s.User.ConnectedServer) && house.PosX == CoordX && house.PosY == CoordY {
			return messaging.SystemMessage(53110), nil //"You can't place an item here."
		}
	}

	if s.Character.CountHouseItemsByCategory(houseinfo.Category) >= 1 && houseinfo.Category == 1 {
		return messaging.SystemMessage(53109), nil //You already have a house item.
	}
	if s.Character.CountHouseItemsByCategory(houseinfo.Category) >= database.MAX_CROPS && houseinfo.Category == 2 {
		return messaging.SystemMessage(53113), nil //You already have the maximum number of crops
	}
	if s.Character.CountHouseItemsByCategory(houseinfo.Category) >= database.MAX_FURNITURE && houseinfo.Category == 3 {
		return messaging.SystemMessage(53112), nil //You already have the maximum number of furniture
	}

	house := &database.HousingItem{
		PosX:           CoordX,
		PosY:           CoordY,
		PosZ:           CoordZ,
		ItemID:         item.ItemID,
		OwnerID:        s.Character.ID,
		ExpirationDate: null.NewTime(time.Now().Add(time.Hour*time.Duration(24*7)), true),
		MapID:          s.Character.Map,
		HouseID:        int(houseid),
		Server:         int16(s.User.ConnectedServer),
		MaxRelaxation:  database.HouseItemsInfos[int(houseid)].Relaxetion,
	}
	if houseinfo.Category == 2 {
		expiration := time.Now().Add(time.Second * time.Duration(int64(info.Timer)))
		house.ExpirationDate = null.NewTime(expiration, true)

	}

	database.GenerateIDForHouse(house)
	err = house.Create()
	if err != nil {
		return nil, err
	}
	database.HousingItemsMutex.Lock()
	house.OnSightPlayers = make(map[int]interface{})
	database.HousingItems[house.ID] = house
	database.HousingItemsMutex.Unlock()

	if database.HouseItemsInfos[house.HouseID].Category == 1 {
		s.Character.House = house
		database.HousingItemsByMap[s.Character.ID] = make(map[int16][]*database.HousingItem)
	}
	database.HousingItemsByMap[house.Server][house.MapID] = append(database.HousingItemsByMap[house.Server][house.MapID], house)
	s.Character.HousingDetails()
	itemsData := s.Character.DecrementItem(int16(slotid), 1)

	house.InitCrop()
	return *itemsData, nil
}
func (h *RemoveHouseItem) Handle(s *database.Socket, data []byte) ([]byte, error) {
	pseudoID := uint16(utils.BytesToInt(data[6:10], true))

	houseitem := database.FindHouseByPseudoId(int(pseudoID))
	if houseitem == nil {
		return nil, nil
	}
	if houseitem.OwnerID != s.Character.ID {
		return messaging.SystemMessage(53114), nil //no authorization
	}
	houseinfo, ok := database.HouseItemsInfos[houseitem.HouseID]
	if !ok {
		return nil, nil
	}
	if houseinfo.Category == 1 {
		for _, house := range database.HousingItems {
			if int(house.Server) == s.Character.ID {
				house.Remove()
			}
		}
	}
	houseitem.Remove()

	return nil, nil
}
func parseDate(date null.Time) string {
	if date.Valid {
		year, month, day := date.Time.Date()
		return fmt.Sprintf("%02d.%02d.%d", year, month, day)
	}

	return ""
}

func (h *BattleModeHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	battleMode := data[5]

	resp := BATTLE_MODE
	resp.Insert(utils.IntToBytes(uint64(s.Character.PseudoID), 2, true), 5) // character pseudo id
	resp[7] = battleMode
	s.Character.BattleMode = int(utils.BytesToInt(data[5:6], true))

	p := nats.CastPacket{CastNear: true, CharacterID: s.Character.ID, Type: nats.BATTLE_MODE, Data: resp}
	if err := p.Cast(); err != nil {
		return nil, err
	}

	return resp, nil
}

func (h *QuestHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	QUEST_MENU := utils.Packet{0xaa, 0x55, 0x13, 0x00, 0x57, 0x02, 0x3d, 0x4e, 0x00, 0x00, 0xb0, 0x5c, 0x56, 0x3d, 0x01, 0x0d, 0x00, 0x00, 0x00, 0x14, 0x5d, 0x56, 0x3d, 0x55, 0xaa}
	resp := QUEST_MENU
	return resp, nil
}

func (h *MeditationHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	meditationMode := data[6] == 1
	s.Character.Meditating = meditationMode

	resp := MEDITATION_MODE
	resp.Insert(utils.IntToBytes(uint64(s.Character.PseudoID), 2, true), 6) // character pseudo id
	resp[8] = data[6]

	p := nats.CastPacket{CastNear: true, CharacterID: s.Character.ID, Type: nats.MEDITATION_MODE, Data: resp}
	if err := p.Cast(); err != nil {
		return nil, err
	}

	return resp, nil
}

func (h *TargetSelectionHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	id := int(utils.BytesToInt(data[5:7], true))
	s.Character.Selection = id

	resp := SELECTION_CHANGED
	resp.Insert(utils.IntToBytes(uint64(s.Character.Selection), 2, true), 5)

	return resp, nil
}

func (h *TravelToCastleHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	if s.Character.Map == 233 {
		resp := CLANCASTLE_MAP
		index := 7
		length := 3
		if database.FiveClans[1].ClanID != 0 {
			resp.Insert([]byte{0x01, 0xdf, 0x04, 0x00, 0x00}, index)
			index += 5
			length += 5
			area, _ := database.FindGuildByID(database.FiveClans[1].ClanID) //FLAME WOLF TEMPLE
			resp.Insert([]byte{byte(len(area.Name))}, index)                // Guild name length
			index++
			resp.Insert([]byte(area.Name), index) // Guild name
			index += len(area.Name)
			length += 1 + len(area.Name)
		}
		if database.FiveClans[2].ClanID != 0 {
			resp.Insert([]byte{0x02, 0xeb, 0x00, 0x00, 0x00}, index)
			index += 5
			length += 5
			area, _ := database.FindGuildByID(database.FiveClans[2].ClanID) //Waterfall ARMY
			resp.Insert([]byte{byte(len(area.Name))}, index)                // Guild name length
			index++
			resp.Insert([]byte(area.Name), index) // Guild name
			index += len(area.Name)
			length += 1 + len(area.Name)
		}
		if database.FiveClans[3].ClanID != 0 {
			resp.Insert([]byte{0x03, 0x5d, 0x06, 0x00, 0x00}, index)
			index += 5
			length += 5
			area, _ := database.FindGuildByID(database.FiveClans[3].ClanID) //SKY HILL
			resp.Insert([]byte{byte(len(area.Name))}, index)                // Guild name length
			index++
			resp.Insert([]byte(area.Name), index) // Guild name
			index += len(area.Name)
			length += 1 + len(area.Name)
		}
		if database.FiveClans[4].ClanID != 0 {
			resp.Insert([]byte{0x04, 0xf0, 0x06, 0x00, 0x00}, index)
			index += 5
			length += 5
			area, _ := database.FindGuildByID(database.FiveClans[4].ClanID) //Forest WOOD TEMPLE
			resp.Insert([]byte{byte(len(area.Name))}, index)                // Guild name length
			index++
			resp.Insert([]byte(area.Name), index) // Guild name
			index += len(area.Name)
			length += 1 + len(area.Name)
		}
		if database.FiveClans[5].ClanID != 0 {
			resp.Insert([]byte{0x05, 0xd7, 0x05, 0x00, 0x00}, index)
			index += 5
			length += 5
			area, _ := database.FindGuildByID(database.FiveClans[5].ClanID) //Underground LAND TEMPLE
			resp.Insert([]byte{byte(len(area.Name))}, index)                // Guild name length
			index++
			resp.Insert([]byte(area.Name), index) // Guild name
			index += len(area.Name)
			length += 1 + len(area.Name)
		}
		resp.SetLength(int16(length))
		//fmt.Printf("RESP:\t %x \n", []byte(resp))
		return resp, nil
	}
	return s.Character.ChangeMap(233, nil)
}

func (h *OpenTacticalSpaceHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	return TACTICAL_SPACE_MENU, nil
}

func (h *TacticalSpaceTPHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	mapID := int16(data[6])
	return s.Character.ChangeMap(mapID, nil)
}

func (h *InTacticalSpaceTPHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	resp := TACTICAL_SPACE_TP
	resp[8] = data[6]
	return resp, nil
}

func (h *OpenLotHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	resp := OPEN_LOT
	if !s.Character.HasLot {
		return nil, nil
	}

	s.Character.HasLot = false
	paid := data[5] == 1
	dropID := 1185

	if paid && s.Character.Gold >= 150000 {
		dropID = 1186
		cost := uint64(150000)
		if !s.Character.SubtractGold(uint64(cost)) {
			return nil, nil
		}
	}

	drop, ok := database.GetDropInfo(dropID)
	if drop == nil {
		return nil, nil
	}

	itemID := int64(0)
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

		itemID = int64(items[index])
		drop, ok = database.GetDropInfo(int(itemID))
	}

	if itemID == 10002 {
		s.User.NCash += 1000
		s.User.Update()

	} else {

		quantity := 1
		if paid {
			if q, ok := PaidLotQuantities[int(itemID)]; ok {
				quantity = q
			}
		} else {
			if q, ok := FreeLotQuantities[int(itemID)]; ok {
				quantity = q
			}
		}

		info, ok := database.GetItemInfo(itemID)
		if !ok || info == nil {
			return nil, nil
		}
		if info.Timer > 0 {
			quantity = info.Timer
		}

		item := &database.InventorySlot{ItemID: itemID, Quantity: uint(quantity)}
		r, _, err := s.Character.AddItem(item, -1, false)
		if err != nil {
			return nil, err
		} else if r == nil {
			return nil, nil
		}

		resp.Concat(*r)
	}

	resp.Insert(utils.IntToBytes(uint64(itemID), 4, true), 11) // item id
	return resp, nil
}

func (h *EnterGateHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if len(data) < 9 {
		return s.Character.ChangeMap(1, nil)
	}
	gateID := int(utils.BytesToInt(data[5:9], true))
	gate, ok := database.Gates[gateID]
	if !ok {
		log.Print("GateID not found: " + strconv.Itoa(gateID))
		return s.Character.ChangeMap(int16(s.Character.Map), nil)
	}
	if gate == nil || s == nil {
		log.Print("Gate nil or socket nil.")
		return nil, nil
	}
	targetmap := gate.TargetMap
	if targetmap == 0 {
		targetmap = uint8(s.Character.Map)
	}
	coordinate := &utils.Location{
		X: float64(gate.Point_X),
		Y: float64(gate.Point_Y),
	}
	if s.Character.Level < gate.MinLevelRequirment {
		s.Write(messaging.SystemMessage(messaging.NO_LEVEL_REQUIREMENT))
		targetmap = uint8(s.Character.Map)
		coordinate = nil
	}
	if s.Character.Faction != gate.FactionRequirment && gate.FactionRequirment != 0 {
		s.Write(messaging.SystemMessage(messaging.INCORRECT_FACTION))
		targetmap = uint8(s.Character.Map)
		coordinate = nil
	}
	return s.Character.ChangeMap(int16(targetmap), coordinate)
}

func (h *SendPvPRequestHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	pseudoID := uint16(utils.BytesToInt(data[6:8], true))
	opponent := database.FindCharacterByPseudoID(s.User.ConnectedServer, pseudoID)
	if opponent == nil {
		return nil, nil

	} else if opponent.DuelID > 0 {
		resp := messaging.SystemMessage(messaging.ALREADY_IN_PVP)
		return resp, nil
	} else if opponent.IsinWar || opponent.Map == 255 {
		resp := messaging.SystemMessage(messaging.ALREADY_IN_PVP)
		return resp, nil
	}

	resp := PVP_REQUEST
	resp.Insert(utils.IntToBytes(uint64(s.Character.PseudoID), 2, true), 6) // sender pseudo id

	database.GetSocket(opponent.UserID).Write(resp)
	return nil, nil
}

func (h *RespondPvPRequestHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	pseudoID := uint16(utils.BytesToInt(data[6:8], true))
	accepted := data[8] == 1

	opponent := database.FindCharacterByPseudoID(s.User.ConnectedServer, pseudoID)
	if opponent == nil {
		return nil, nil
	}

	if !accepted {
		resp := messaging.SystemMessage(messaging.PVP_REQUEST_REJECTED)
		s.Write(resp)
		database.GetSocket(opponent.UserID).Write(resp)

	} else if opponent.DuelID > 0 {
		resp := messaging.SystemMessage(messaging.ALREADY_IN_PVP)
		return resp, nil

	} else { // start pvp
		mC := database.ConvertPointToLocation(s.Character.Coordinate)
		oC := database.ConvertPointToLocation(opponent.Coordinate)
		fC := utils.Location{X: (mC.X + oC.X) / 2, Y: (mC.Y + oC.Y) / 2}

		s.Character.DuelID = opponent.ID
		opponent.DuelID = s.Character.ID

		resp := PVP_STARTED
		resp.Insert(utils.FloatToBytes(fC.X, 4, true), 6)  // flag-X
		resp.Insert(utils.FloatToBytes(fC.Y, 4, true), 10) // flag-Y

		s.Character.Socket.Write(resp)
		opponent.Socket.Write(resp)

		go s.Character.StartPvP(3)
		go opponent.StartPvP(3)
	}

	return nil, nil
}

func (h *TransferSoulHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	//pseudoID := uint16(utils.BytesToInt(data[6:8], true))
	resp := utils.Packet{0xAA, 0x55, 0x06, 0x00, 0xA5, 0x01, 0x0A, 0x00, 0x55, 0xAA}
	resp.Insert(data[6:8], 8)
	resp.Print()
	return resp, nil
}
func (h *AcceptSoulHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	//pseudoID := uint16(utils.BytesToInt(data[6:8], true))
	resp := utils.Packet{0xaa, 0x55, 0x07, 0x00, 0xa5, 0x02, 0x0a, 0x00, 0x01, 0x01, 0x00, 0x55, 0xaa}
	//resp.Insert(data[6:8], 8)
	resp.Print()
	return resp, nil
}
func (h *FinishSoulHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	//pseudoID := uint16(utils.BytesToInt(data[6:8], true))
	resp := utils.Packet{0xaa, 0x55, 0x04, 0x00, 0xa5, 0x05, 0x0a, 0x00, 0x55, 0xaa}
	//resp.Insert(data[6:8], 8)
	resp.Print()
	return resp, nil
}

func (h *CharmOfIdentity) Handle(s *database.Socket, data []byte) ([]byte, error) {

	length := utils.BytesToInt(data[6:7], true)
	name := string(data[7 : 7+length])

	ok, err := database.IsValidUsername(name)
	if err != nil {
		return nil, err
	} else if !ok || length < 4 {
		return messaging.SystemMessage(messaging.INVALID_NAME), nil
	}
	slotIDitem, _, _ := s.Character.FindItemInInventory(nil, 15710005)
	rr, _ := s.Character.RemoveItem(slotIDitem)
	s.Write(rr)
	return s.Character.ChangeName(name)
}
func (h *StyleHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	slot, _, err := s.Character.FindItemInInventory(nil, 15830000, 15830001, 17502883)
	if err != nil {
		log.Println(err)
		return nil, err
	} else if slot == -1 {
		return nil, nil
	}
	s.Write(*s.Character.DecrementItem(slot, 1))
	height := data[6]
	head := utils.BytesToInt(data[7:11], true)
	face := utils.BytesToInt(data[11:15], true)
	headinfo, ok := database.GetItemInfo(head)
	if !ok || headinfo == nil {
		head = 0
	}
	faceinfo, ok := database.GetItemInfo(face)
	if !ok || faceinfo == nil {
		face = 0
	}
	STYLE_MENU := utils.Packet{0xaa, 0x55, 0x0d, 0x00, 0x01, 0xb5, 0x0a, 0x00, 0x00, 0x55, 0xaa}
	resp := STYLE_MENU
	resp[8] = height
	index := 9
	resp.Insert(utils.IntToBytes(uint64(head), 4, true), index)
	index += 4
	resp.Insert(utils.IntToBytes(uint64(face), 4, true), index)
	index += 4
	s.Character.Height = int(height)
	s.Character.HeadStyle = head
	s.Character.FaceStyle = face
	s.Character.Update()
	return resp, nil
}
func (h *TravelToFiveClanArea) Handle(s *database.Socket, data []byte) ([]byte, error) {
	areaID := int16(data[7])
	switch areaID {
	case 0:
		x := "508,564"
		coord := s.Character.Teleport(database.ConvertPointToLocation(x))
		s.Write(coord)
	case 1: //FLAME WOLF TEMPLE
		if s.Character.GuildID == database.FiveClans[1].ClanID {
			x := "243,777"
			coord := s.Character.Teleport(database.ConvertPointToLocation(x))
			s.Write(coord)
		} else {
			s.Write(CANNOT_MOVE)
		}
	case 2: //OCEAN ARMY
		if s.Character.GuildID == database.FiveClans[2].ClanID {
			x := "131,433"
			coord := s.Character.Teleport(database.ConvertPointToLocation(x))
			s.Write(coord)
		} else {
			s.Write(CANNOT_MOVE)
		}
	case 3: //LIGHTNING HILL
		if s.Character.GuildID == database.FiveClans[3].ClanID {
			x := "615,171"
			coord := s.Character.Teleport(database.ConvertPointToLocation(x))
			s.Write(coord)
		} else {
			s.Write(CANNOT_MOVE)
		}
	case 4: //SOUTHERN WOOD TEMPLE
		if s.Character.GuildID == database.FiveClans[4].ClanID {
			x := "863,425"
			coord := s.Character.Teleport(database.ConvertPointToLocation(x))
			s.Write(coord)
		} else {
			s.Write(CANNOT_MOVE)
		}
	case 5: //WESTERN LAND TEMPLE
		if s.Character.GuildID == database.FiveClans[5].ClanID {
			x := "689,867"
			coord := s.Character.Teleport(database.ConvertPointToLocation(x))
			s.Write(coord)
		} else {
			s.Write(CANNOT_MOVE)
		}
	}

	return nil, nil
}
func (h *TransferItemTypeHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	//pseudoID := uint16(utils.BytesToInt(data[6:8], true))
	resp := utils.Packet{0xAA, 0x55, 0x06, 0x00, 0x60, 0x01, 0x0A, 0x00, 0x55, 0xAA}
	resp.Insert(data[6:8], 8)
	fslot, _, err := s.Character.FindItemInInventory(nil, 15710007, 17200237)
	if err != nil {
		log.Println(err)
		return nil, err
	} else if fslot == -1 {
		return nil, nil
	}
	slot := utils.BytesToInt(data[6:8], true)
	invSlots, err := s.Character.InventorySlots()
	if err != nil {
		return nil, err
	}
	item := invSlots[slot]
	info, ok := database.GetItemInfo(item.ItemID)
	if !ok || info.ItemPair == 0 {
		return nil, nil
	} else {
		freeslot, err := s.Character.FindFreeSlot()
		if err != nil {
			return nil, err
		} else if freeslot == -1 { // no free slot
			return messaging.InfoMessage(fmt.Sprintf("You don't have enough space in inventory")), nil
		}
		s.Character.Socket.Write(*s.Character.DecrementItem(int16(fslot), 1))
		item.ItemID = info.ItemPair
		item.Update()
		resp.Concat(item.GetData(item.SlotID))
	}

	return resp, nil
}

func (h *EnhancementTransfer) Handle(s *database.Socket, data []byte) ([]byte, error) {

	resp := utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x06, 0x01, 0x55, 0xAA}

	firstWeapSlot := utils.BytesToInt(data[6:8], true)
	secWeapSlot := utils.BytesToInt(data[8:10], true)

	fslot, _, err := s.Character.FindItemInInventory(nil, 80006068)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if fslot == -1 {
		rip := utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x54, 0x04, 0x11, 0x10, 0x00, 0x55, 0xAA}
		resp.Concat(rip)
		return resp, err
	}

	invSlots, err := s.Character.InventorySlots()
	if err != nil {
		return nil, err
	}

	firstWeap := invSlots[firstWeapSlot]
	secWeap := invSlots[secWeapSlot]
	firstweaponiteminfo, ok := database.GetItemInfo(firstWeap.ItemID)
	secWeapItemInfo, ok := database.GetItemInfo(secWeap.ItemID)
	if !ok {
		return nil, nil
	}

	if firstWeap.Plus != 0 && firstweaponiteminfo.Slot == secWeapItemInfo.Slot {

		secWeap.Plus = firstWeap.Plus
		secWeap.UpgradeArr = firstWeap.UpgradeArr
		secWeap.SocketCount = firstWeap.SocketCount
		secWeap.SocketArr = firstWeap.SocketArr

		secWeap.Update()

		resp.Concat(secWeap.GetData(secWeap.SlotID))
		s.Write(resp)
		rr, _ := s.Character.RemoveItem(firstWeap.SlotID)
		s.Write(rr)
		rr, _ = s.Character.RemoveItem(fslot)
		s.Write(rr)
		rip := utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x54, 0x04, 0x10, 0x10, 0x00, 0x55, 0xAA}
		resp.Concat(rip)

	} else {
		rip := utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x54, 0x04, 0x11, 0x10, 0x00, 0x55, 0xAA}
		resp.Concat(rip)
	}
	return resp, nil
}

func (h *ClothImproveChest) Handle(s *database.Socket, data []byte) ([]byte, error) {

	resp := utils.Packet{}
	c := s.Character
	slots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}

	itemSlot := int16(utils.BytesToInt(data[6:8], true))
	if itemSlot == 0 {
		return nil, nil
	} else if slots[itemSlot].ItemID == 0 {
		return nil, nil
	}

	armor := slots[itemSlot]
	iteminfo, ok := database.GetItemInfo(armor.ItemID)
	if !ok {
		return nil, nil
	}

	if iteminfo.Type == 64 {
		boxid := []int64{}
		if iteminfo.HtType == 1 {
			boxid = []int64{15710617, 13003352}
		} else if iteminfo.HtType == 2 {
			boxid = []int64{15710616, 13003351, 13003192}
		} else if iteminfo.HtType == 3 {
			boxid = []int64{15710615}
		} else {
			return nil, nil
		}
		for _, v := range boxid {

			slot, chest, err := c.FindItemInInventory(nil, v)
			if chest == nil || err != nil {
				continue
			}
			if slot == -1 {
				continue
			}
			ok := false
			for !ok {
				maxplus := funk.Contains(iteminfo.Name, "(+5)")
				maxlevel := funk.Contains(iteminfo.Name, "_5Kyu)")
				if maxplus && maxlevel {
					s.Write(messaging.InfoMessage("You can't improve this item anymore"))
					return nil, nil
				}

				rand := utils.RandInt(1, 1000)
				if rand > 900 {
					s.Write(messaging.InfoMessage("Improvement failed"))
					rr, _ := s.Character.RemoveItem(slot)
					return rr, nil

				}
				if rand < 500 {
					if !maxplus {
						armor.ItemID++
						armor.Update()
						s.Write(armor.GetData(armor.SlotID))
					} else {
						continue
					}

				} else if rand > 500 {
					if !maxlevel {
						armor.ItemID += 6
						armor.Update()
						s.Write(armor.GetData(armor.SlotID))
					} else {
						continue
					}
				}

				rr, _ := s.Character.RemoveItem(slot)
				s.Write(messaging.InfoMessage("Improvement successful"))
				s.Write(rr)
				ok = true

			}
			return nil, nil
		}
		return nil, nil

	}

	slot, chest, err := c.FindItemInInventory(nil, 99002838)
	if chest == nil || err != nil {
		return nil, nil
	}
	if slot == -1 {
		return nil, nil
	}

	if armor.Buff != 0 {
		return messaging.SystemMessage(messaging.ALREADY_HAVE_AURA), nil
	}
	seed := utils.RandInt(0, 1000)
	if seed < 350 {
		iteminfo, ok := database.GetItemInfo(armor.ItemID)
		if !ok {
			return nil, nil
		}

		if iteminfo.Slot != 309 {
			return nil, nil
		}
		if armor.ItemID == 203001033 {
			armor.Buff = 60027
		} else {
			armor.Buff = 60026
		}
		armor.Update()
		resp.Concat(armor.GetData(armor.SlotID))
		resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_SUCCESS))
	} else {
		resp.Concat(messaging.SystemMessage(messaging.ENHANCEMENT_FAIL))
	}

	rr, _ := s.Character.RemoveItem(slot)
	resp.Concat(rr)

	return resp, nil

}
func (h *ChangePetName) Handle(s *database.Socket, data []byte) ([]byte, error) {

	length := int(data[12])

	name := string(data[12 : length+12])

	if length < 4 {
		return messaging.SystemMessage(messaging.INVALID_NAME), nil
	}
	slotIDitem, _, _ := s.Character.FindItemInInventory(nil, 17300186)
	if slotIDitem == -1 {
		return messaging.SystemMessage(messaging.INVALID_NAME), nil
	}
	rr, _ := s.Character.RemoveItem(slotIDitem)
	s.Write(rr)

	slots, _ := s.Character.InventorySlots()
	petSlot := slots[0x0A]
	pet := petSlot.Pet

	pet.Name = name
	petSlot.Update()

	resp := petSlot.GetData(petSlot.SlotID)

	return resp, nil

}
func (h *ChangePartyModeHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	settingIndex := utils.BytesToInt(data[6:7], false)
	c := s.Character
	switch settingIndex {
	case 33: //1-1
		c.GroupSettings.ExperienceSharingMethod = 1
		c.GroupSettings.LootDistriburionMethod = 1
	case 40: //1-2
		c.GroupSettings.ExperienceSharingMethod = 1
		c.GroupSettings.LootDistriburionMethod = 2
	case 34: //1-3
		c.GroupSettings.ExperienceSharingMethod = 1
		c.GroupSettings.LootDistriburionMethod = 3
	case 36: //1-4
		c.GroupSettings.ExperienceSharingMethod = 1
		c.GroupSettings.LootDistriburionMethod = 4
	case 17: // 2-1
		c.GroupSettings.ExperienceSharingMethod = 2
		c.GroupSettings.LootDistriburionMethod = 1
	case 24: //2-2
		c.GroupSettings.ExperienceSharingMethod = 2
		c.GroupSettings.LootDistriburionMethod = 2
	case 18: //2-3
		c.GroupSettings.ExperienceSharingMethod = 2
		c.GroupSettings.LootDistriburionMethod = 3
	case 20: //2-4
		c.GroupSettings.ExperienceSharingMethod = 2
		c.GroupSettings.LootDistriburionMethod = 4
	}

	return data, nil
}
func (h *CtrlClick) Handle(s *database.Socket, data []byte) ([]byte, error) {
	return nil, nil
}
func (h *SendReport) Handle(s *database.Socket, data []byte) ([]byte, error) {
	return nil, nil
}

func (h *AddNewFriend) Handle(s *database.Socket, data []byte) ([]byte, error) {

	length := int16(data[6])
	charname := string(data[7 : length+7])
	char, err := database.FindCharacterByName(charname)

	if err != nil {
		return nil, nil
	}
	if char == nil || char.ID == s.Character.ID {
		return []byte{0xaa, 0x55, 0x04, 0x00, 0xcb, 0x03, 0x51, 0x08, 0x55, 0xaa}, nil
	} else if int16(data[len(data)-3]) == 0 {

		g, err := database.FindFriendByCharacterAndFriendID(s.Character.ID, char.ID)
		if err != nil {
			log.Print("FindError: ", err)
			return nil, err
		}
		if g != nil {
			log.Print("Friend already added: ", g.FriendID)
			return nil, nil
		}
		f := &database.Friend{
			CharacterID: s.Character.ID,
			FriendID:    char.ID,
		}
		cerror := f.Create()
		if cerror != nil {
			return nil, cerror
		}
		index := 8
		resp := database.ADD_FRIEND
		resp.Insert(utils.IntToBytes(uint64(f.ID), 4, true), index)
		index += 4
		resp.Insert(utils.IntToBytes(uint64(len(charname)), 1, true), index)
		index++
		resp.Insert([]byte(charname), index) // character name
		index += len(charname) + 1
		online, err := boolconv.NewBoolByInterface(char.IsOnline)
		if err != nil {
			log.Println("error should not be nil")
		}
		resp.Overwrite(online.Bytes(), index)
		resp.SetLength(int16(binary.Size(resp) - 6))
		return resp, nil
		//}
	} else {
		var f *database.Friend
		g, err := database.FindFriendByCharacterAndFriendID(s.Character.ID, char.ID)
		if err != nil {
			log.Print("FindError: ", err)
			return nil, err
		}
		if g != nil {
			g.IsBlocked = true
			g.Update()
			f = g
		} else {
			f = &database.Friend{
				CharacterID: s.Character.ID,
				FriendID:    char.ID,
				IsBlocked:   true,
			}
			cerror := f.Create()
			if cerror != nil {
				return nil, cerror
			}
		}

		index := 8
		resp := database.BLOCK_FRIEND
		resp.Insert(utils.IntToBytes(uint64(f.ID), 4, true), index)
		index += 4
		resp.Insert(utils.IntToBytes(uint64(len(charname)), 1, true), index)
		index++
		resp.Insert([]byte(charname), index) // character name
		index += len(charname) + 1
		online, err := boolconv.NewBoolByInterface(char.IsOnline)
		if err != nil {
			log.Println("error should not be nil")
		}
		resp.Overwrite(online.Bytes(), index)
		resp.SetLength(int16(binary.Size(resp) - 6))
		return resp, nil
		//}
	}
	//s.Character.LoadQuests(int(questID), 3)

}
func (h *RemoveFriend) Handle(s *database.Socket, data []byte) ([]byte, error) {
	frid := utils.BytesToInt(data[6:10], true)
	friend, err := database.FindFriendsByID(int(frid))
	if err != nil {
		return nil, err
	}
	if friend == nil {
		return nil, nil
	}
	err = friend.Delete()
	if err != nil {
		return nil, err
	}
	return nil, nil
}
