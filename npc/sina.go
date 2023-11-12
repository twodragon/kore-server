package npc

import (
	"fmt"

	"github.com/twodragon/kore-server/database"
	"github.com/twodragon/kore-server/messaging"
	"github.com/twodragon/kore-server/utils"
)

type (
	AppearanceHandler        struct{}
	AppearanceRestoreHandler struct{}
)

func (h *AppearanceHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	resp := utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x54, 0x22, 0x0a, 0x00, 0x00, 0x55, 0xAA}
	itemSlot := int(utils.BytesToInt(data[6:8], true))
	newitemSlot := int(utils.BytesToInt(data[8:10], true))
	matitemSlot := int(utils.BytesToInt(data[10:12], true))
	inventory, err := s.Character.InventorySlots()
	if err != nil {
		return nil, err
	}
	item := inventory[itemSlot]
	newitem := inventory[newitemSlot]
	matitem := inventory[matitemSlot]
	iteminfo, ok := database.GetItemInfo(matitem.ItemID)
	if !ok || iteminfo == nil {
		return nil, nil
	}

	if iteminfo.GetType() != database.ILLUSION_WATER_TYPE {
		return nil, nil
	}

	item.Appearance = int64(newitem.ItemID)
	item.Update()
	s.Write(item.GetData(item.SlotID))

	r, err := s.Character.RemoveItem(int16(matitemSlot))
	if err != nil {
		return nil, err
	}

	resp.Concat(r)

	r, err = s.Character.RemoveItem(int16(newitemSlot))
	if err != nil {
		return nil, err
	}
	resp.Concat(r)

	return resp, nil
}
func (h *AppearanceRestoreHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	resp := utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x54, 0x24, 0x0a, 0x00, 0x00, 0x55, 0xAA}
	cost := uint64(5000000)
	itemSlot := int(utils.BytesToInt(data[6:8], true))
	inventory, err := s.Character.InventorySlots()
	if err != nil {
		return nil, err
	}
	freeslot, err := s.Character.FindFreeSlot()
	if err != nil {
		return nil, err
	} else if freeslot == -1 { // no free slot
		return messaging.InfoMessage(fmt.Sprintf("You don't have enough space in inventory")), nil
	}
	if s.Character.Gold < cost {
		return messaging.InfoMessage(fmt.Sprintf("Not enogh gold!")), nil
	}

	item := inventory[itemSlot]
	item.Appearance = 0
	item.SlotID = freeslot
	item.Update()
	refresh := item.GetData(item.SlotID)
	resp.Concat(refresh)

	remove := utils.Packet{0xAA, 0x55, 0x0B, 0x00, 0x59, 0x02, 0x0A, 0x00, 0x01, 0x55, 0xAA}
	remove.Insert(utils.IntToBytes(uint64(item.ItemID), 4, true), 9) // item id
	remove.Insert(utils.IntToBytes(uint64(itemSlot), 2, true), 13)   // slot id
	resp.Concat(remove)

	if !s.Character.SubtractGold(uint64(cost)) {
		return nil, nil
	}

	return resp, nil
}
