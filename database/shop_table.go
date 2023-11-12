package database

import (
	"log"

	"github.com/twodragon/kore-server/utils"
	"github.com/xuri/excelize/v2"
)

var (
	Shops = make(map[int]*Shop)
)

type Shop struct {
	ID         int
	Name       string
	ItemsTable []*ShopItem
}

func (e *Shop) IsPurchasable(itemID int) bool {

	shop := e.ItemsTable
	for _, itemTable := range shop {
		for _, item := range itemTable.Items {
			if item == itemID {
				return true
			}
		}
	}

	return false
}

func GetShopsTable() error {
	log.Print("Reading ShopsTable...")
	f, err := excelize.OpenFile("data/tb_ShopTable_Item.xlsx")
	if err != nil {
		return err
	}
	defer f.Close()
	// Get all the rows in the Sheet1.
	rows, err := f.GetRows("Sheet1")
	if err != nil {
		return err
	}
	for index, row := range rows {
		if index == 0 {
			continue
		}
		Shops[utils.StringToInt(row[1])] = &Shop{
			ID:   utils.StringToInt(row[1]),
			Name: row[2],
		}
		for i := 5; i <= 9; i++ {
			itemstable, ok := ShopItems[utils.StringToInt(row[i])]
			if !ok || utils.StringToInt(row[i]) == 0 {
				continue
			}
			Shops[utils.StringToInt(row[1])].ItemsTable = append(Shops[utils.StringToInt(row[1])].ItemsTable, itemstable)
		}
	}

	return nil
}
