package player

import (
	"github.com/twodragon/kore-server/database"
	"github.com/twodragon/kore-server/utils"
)

type (
	CookingStarted struct{}
)

func (h *CookingStarted) Handle(s *database.Socket, data []byte) ([]byte, error) {
	resp := utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x54, 0x21, 0x0A, 0x00, 0x00, 0x55, 0xAA}
	resp.Concat(utils.Packet{0xAA, 0x55, 0x28, 0x00, 0x16, 0xA0, 0x00, 0xA0, 0x00, 0x1E, 0x04, 0x00, 0x00,
		0x15, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x27, 0x00, 0x00, 0x55, 0xAA})
	craftID := int(utils.BytesToInt(data[6:10], true))
	finished := int(data[10])
	canCraft := true

	receipe := database.CookingItems[int(craftID)]
	prodMaterials := receipe.GetMaterials()
	prodAmounts := receipe.GetAmounts()

	if finished == 1 {
		receipe := database.CookingItems[int(craftID)]
		if receipe == nil {
			resp.Concat(database.COOKING_ERROR)
			return database.COOKING_ERROR, nil
		}
		for i, material := range prodMaterials {
			matCount := uint(prodAmounts[i])

			slotID, _, _ := s.Character.FindItemInInventory(nil, int64(material))
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
			prod, _ := s.Character.CookFood(craftID)
			resp.Concat(prod)

		}
	}

	return resp, nil
}
