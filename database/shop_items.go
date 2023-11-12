package database

import (
	"log"

	"github.com/twodragon/kore-server/utils"
	"github.com/xuri/excelize/v2"
)

var (
	ShopItems = make(map[int]*ShopItem)
)

type ShopItem struct {
	Id    int
	Items []int
}

func GetShopItems() error {
	log.Print("Reading AdvancedFusions...")
	f, err := excelize.OpenFile("data/tb_ShopTable.xlsx")
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
		ShopItems[utils.StringToInt(row[1])] = &ShopItem{
			Id: utils.StringToInt(row[1]),
		}
		for i := 2; i < 42; i += 2 {
			if utils.StringToInt(row[i]) == 0 {
				continue
			}
			ShopItems[utils.StringToInt(row[1])].Items = append(ShopItems[utils.StringToInt(row[1])].Items, utils.StringToInt(row[i]))
		}
	}

	return nil
}
