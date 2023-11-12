package npc

import (
	"github.com/twodragon/kore-server/database"
	"github.com/twodragon/kore-server/utils"
)

type (
	CreateSocketHandler  struct{}
	UpgradeSocketHandler struct{}
	CoProductionHandler  struct{}
)

var (
	CREATED_SOCKET = utils.Packet{}
)

func (h *CreateSocketHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	slots, err := s.Character.InventorySlots()
	if err != nil {
		return nil, err
	}

	itemSlot := int16(utils.BytesToInt(data[6:8], true))
	if itemSlot == 0 {
		return nil, nil
	} else if slots[itemSlot].ItemID == 0 {
		return nil, nil
	}

	var special *database.InventorySlot
	specialSlot := int16(utils.BytesToInt(data[8:10], true))
	if specialSlot == 0 {
		special = nil
	} else {
		special = slots[specialSlot]
	}

	return s.Character.CreateSocket(slots[itemSlot], special, itemSlot, specialSlot)
}

func (h *UpgradeSocketHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	slots, err := s.Character.InventorySlots()
	if err != nil {
		return nil, err
	}

	itemSlot := int16(utils.BytesToInt(data[6:8], true))
	if itemSlot == 0 {
		return nil, nil
	} else if slots[itemSlot].ItemID == 0 {
		return nil, nil
	}

	socketSlot := int16(utils.BytesToInt(data[8:10], true))
	socket := slots[socketSlot]
	if socketSlot == 0 {
		resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x54, 0x17, 0x0E, 0xCF, 0x55, 0xAA}
		return resp, nil
	} else if socket.ItemID == 0 || socket.ItemID != 235 {
		resp := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x54, 0x17, 0x0E, 0xCF, 0x55, 0xAA}
		return resp, nil
	}

	var special *database.InventorySlot
	specialSlot := int16(utils.BytesToInt(data[10:12], true))
	if specialSlot == 0 {
		special = nil
	} else {
		special = slots[specialSlot]
	}

	var edit *database.InventorySlot
	editSlot := int16(utils.BytesToInt(data[12:14], true))
	if editSlot == 0 {
		edit = nil
	} else {
		edit = slots[editSlot]
	}

	index, locks := 14, make([]bool, 5)
	if edit != nil {
		for i := 0; i < 5; i++ {
			locks[i] = data[index] == 1
			index++
		}
	}

	return s.Character.UpgradeSocket(slots[itemSlot], slots[socketSlot], special, edit, itemSlot, socketSlot, specialSlot, editSlot, locks)
}

func (h *CoProductionHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	resp := utils.Packet{}
	craftID := int(utils.BytesToInt(data[6:10], true))
	bFinished := int(data[10])
	canCraft := true
	production := database.CraftItems[int(craftID)]
	if production == nil {
		return nil, nil
	}
	var prodMaterials []int
	var prodQty []int
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

	for i := 0; i < len(prodMaterials); i++ {
		matItemId := int64(prodMaterials[i])
		matCount := uint(prodQty[i])

		slotID, _, _ := s.Character.FindItemInInventory(nil, matItemId)
		if slotID == -1 {
			return nil, nil
		}
		slots, err := s.Character.InventorySlots()
		if err != nil {
			return nil, err
		}
		item := slots[slotID]
		if item.Quantity < matCount {
			canCraft = false
		}
	}
	if canCraft {
		prod, _ := s.Character.CoProduction(craftID, bFinished)
		resp.Concat(prod)
	}
	return resp, nil
}
