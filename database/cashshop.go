package database

import (
	"log"

	"github.com/twodragon/kore-server/utils"
	"github.com/xuri/excelize/v2"
)

type TavernItem struct {
	ID     int
	ItemID int
	Price  int
	//	ItemDescription  string
	//	UsageDesctiption string
	//	IsNew     bool
	//	IsPopular bool
	IsActive bool
}

var (
	TavernItems = make(map[int]*TavernItem)
)

func GetHTItems() error {
	log.Print("Reading tavern table...")
	f, err := excelize.OpenFile("data/tb_cashshop.xlsx")
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
		if index <= 1 {
			continue
		}
		TavernItems[utils.StringToInt(row[2])] = &TavernItem{
			ID:     utils.StringToInt(row[1]),
			ItemID: utils.StringToInt(row[2]),
			Price:  utils.StringToInt(row[3]),
			//			ItemDescription:  row[4],
			//			UsageDesctiption: row[5],
			//			IsNew:     utils.StringToBool(row[6]),
			//			IsPopular: utils.StringToBool(row[7]),
			IsActive: utils.StringToBool(row[13]), //kore 13
		}
	}
	return nil
}
