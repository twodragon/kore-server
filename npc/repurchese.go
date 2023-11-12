package npc

import (
	"github.com/twodragon/kore-server/database"
	"github.com/twodragon/kore-server/utils"
)

type (
	RepurchaseItemHandler struct{}
)

var (
	REPURCHASED_ITEM = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0xAF, 0x01, 0x0A, 0x00, 0x00, 0x55, 0xAA}
)

func (h *RepurchaseItemHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	c := s.Character
	if c == nil {
		return nil, nil
	}

	itemID := utils.BytesToInt(data[6:10], true)
	index := int(data[10])
	to := int16(utils.BytesToInt(data[11:13], true))

	r2 := REPURCHASED_ITEM
	r2[8] = byte(index)

	check := c.RepurchaseList.HasItem(itemID)
	if check == -1 {
		return nil, nil
	}

	slot := c.RepurchaseList.Pop(index)
	iteminfo, ok := database.GetItemInfo(slot.ItemID)
	if !ok || iteminfo == nil {
		return nil, nil
	}

	multiplier := 0
	quantity := slot.Quantity
	if slot.ItemID == itemID {
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
		unitPrice := uint64(iteminfo.SellPrice) * uint64(multiplier)
		if slot.Plus > 0 {
			unitPrice *= uint64(slot.Plus)
		}

		amount := (unitPrice * uint64(quantity))
		amount = uint64(float64(amount))
		if amount < 0 {
			return nil, nil
		}
		amount *= 2
		if !c.SubtractGold(amount) {
			return nil, nil
		}
	} else {
		return nil, nil
	}

	itemData, _, err := c.AddItem(slot, to, false)
	if err != nil {
		return nil, err
	} else if itemData == nil {
		return nil, nil
	}
	r2.Concat(c.GetGold())
	r2.Concat(*itemData)

	return r2, nil
}
