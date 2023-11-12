package npc

import (
	"log"

	"github.com/twodragon/kore-server/database"
	"github.com/twodragon/kore-server/utils"
)

type BuyItemHandler struct {
}

type SellItemHandler struct {
}

var (
	NOT_ENOUGH_GOLD = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x59, 0x08, 0xF9, 0x03, 0x55, 0xAA} // not enough gold
	ITEM_BUYED      = utils.Packet{0xaa, 0x55, 0x3c, 0x00, 0x58, 0x01, 0x0a, 0x00, 0x15, 0x64, 0x64, 0x00, 0x00, 0x00, 0x00, 0x00, 0x20, 0x1c, 0x00, 0x00, 0x55, 0xaa}
)

func (h *BuyItemHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	c := s.Character
	if c == nil {
		return nil, nil
	}

	itemID := utils.BytesToInt(data[6:10], true)
	quantity := utils.BytesToInt(data[10:12], true)
	slotID := int16(utils.BytesToInt(data[16:18], true))
	npcID := int(utils.BytesToInt(data[18:22], true))
	shopID, ok := shops[npcID]
	if !ok {
		shopID = 25
		log.Println("Shop id not found and set id=25")
	}

	shop, ok := database.Shops[shopID]
	if !ok {
		return nil, nil
	}

	canPurchase := shop.IsPurchasable(int(itemID))
	if !canPurchase {
		return nil, nil
	}

	resp, err := c.BuyItem(quantity, itemID, slotID, shopID)

	return resp, err
}
func (h *SellItemHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	/*if s.Character.Socket.User.UserType >= 2 && s.Character.Socket.User.UserType < 5 {
		return nil, nil
	}*/

	c := s.Character
	if c == nil {
		return nil, nil
	}

	slots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}

	itemID := utils.BytesToInt(data[6:10], true)
	quantity := int(utils.BytesToInt(data[10:12], true))
	slotID := int16(utils.BytesToInt(data[12:14], true))

	item, ok := database.GetItemInfo(itemID)
	if !ok {
		return nil, nil
	}
	slot := slots[slotID]

	multiplier := 0
	if slot.ItemID == itemID && quantity > 0 && uint(quantity) <= slot.Quantity {
		upgs := slot.GetUpgrades()
		for i := uint8(0); i < slot.Plus; i++ {
			upg := upgs[i]
			if code, ok := database.HaxCodes[int(upg)]; ok {
				multiplier += code.SaleMultiplier
			}
		}

		multiplier /= 1000
		if multiplier == 0 {
			multiplier = 1
		}

		stat := c.Socket.Stats
		unitPrice := uint64(item.SellPrice) * uint64(multiplier)
		unitPrice = unitPrice - uint64(float32(unitPrice)*stat.ShopMultiplier)

		if slot.Plus > 0 {
			unitPrice *= uint64(slot.Plus)
		}

		return c.SellItem(int(itemID), int(slotID), int(quantity), unitPrice)
	}

	return nil, nil
}
