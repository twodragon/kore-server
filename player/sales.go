package player

import (
	"github.com/twodragon/kore-server/database"
	"github.com/twodragon/kore-server/utils"
)

type (
	OpenBuyMenuHandler  struct{}
	OpenSaleMenuHandler struct{}
	OpenSaleHandler     struct{}
	VisitSaleHandler    struct{}
	CloseSaleHandler    struct{}
	BuySaleItemHandler  struct{}
)

var (
	OPEN_SALE_MENU = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x55, 0x09, 0x0A, 0x00, 0x00, 0x55, 0xAA}
	OPEN_BUY_MENU  = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x68, 0x09, 0x0A, 0x00, 0x00, 0x55, 0xAA}
)

func (h *OpenSaleMenuHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	c := s.Character
	if c.AidMode {
		return nil, nil
	}
	if c.Socket.User.UserType >= 2 && c.Socket.User.UserType < 5 {
		return nil, nil
	}

	sale := database.FindSale(c.PseudoID)
	if sale != nil {
		return nil, nil
	}

	return OPEN_SALE_MENU, nil
}

func (h *OpenBuyMenuHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	c := s.Character
	if c.AidMode {
		return nil, nil
	}

	sale := database.FindSale(c.PseudoID)
	if sale != nil {
		return nil, nil
	}

	return OPEN_BUY_MENU, nil
}

func (h *OpenSaleHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	c := s.Character

	if c.AidMode {
		return nil, nil
	}

	if c.Socket.User.UserType >= 2 && c.Socket.User.UserType < 5 {
		return nil, nil
	}

	sale := database.FindSale(c.PseudoID)
	if sale != nil {
		return nil, nil
	}

	saleNameLength := data[6]
	saleName := string(data[7 : 7+saleNameLength])

	index := 7 + saleNameLength

	itemCount := int(data[index])
	index++

	slotIDs, prices := []int16{}, []uint64{}
	for i := 0; i < itemCount; i++ {
		slotID := int16(utils.BytesToInt(data[index:index+2], true))
		index += 2
		index += 2

		price := uint64(utils.BytesToInt(data[index:index+8], true))
		index += 8

		slotIDs = append(slotIDs, slotID)
		prices = append(prices, price)
	}

	return c.OpenSale(saleName, slotIDs, prices)
}

func (h *VisitSaleHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	c := s.Character

	if c.AidMode {
		return nil, nil
	}

	sale := database.FindSale(c.PseudoID)
	if sale != nil {
		return nil, nil
	}

	saleID := uint16(utils.BytesToInt(data[6:8], true))
	sale = database.FindSale(saleID)
	if sale != nil {
		c.VisitedSaleID = saleID
		return sale.Data, nil
	}

	return nil, nil
}

func (h *CloseSaleHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	c := s.Character

	sale := database.FindSale(c.PseudoID)
	if sale == nil {
		return nil, nil
	}
	visitors := database.FindSaleVisitors(sale.ID)
	for _, v := range visitors {
		v.Socket.Write(database.CLOSE_SALE)
		v.VisitedSaleID = 0
	}

	return c.CloseSale()
}

func (h *BuySaleItemHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	c := s.Character

	if c.AidMode {
		return nil, nil
	}

	sale := database.FindSale(c.PseudoID)
	if sale != nil {
		return nil, nil
	}

	saleID := uint16(utils.BytesToInt(data[6:8], true))
	saleSlotID := int16(utils.BytesToInt(data[8:10], true))
	invSlotID := int16(utils.BytesToInt(data[10:12], true))

	return c.BuySaleItem(saleID, saleSlotID, invSlotID)
}
