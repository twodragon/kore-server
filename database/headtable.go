package database

import (
	"log"

	"github.com/twodragon/kore-server/utils"
	"github.com/xuri/excelize/v2"
)

var (
	HeadItems = make(map[int]*HeadItem)
)

type HeadItem struct {
	ID     int
	ItemID int64
}

func GetHeadItem() error {
	log.Print("Reading Head table...")
	f, err := excelize.OpenFile("data/tb_HeadTable.xlsx")
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
		HeadItems[utils.StringToInt(row[1])] = &HeadItem{
			ID:     utils.StringToInt(row[1]),
			ItemID: int64(utils.StringToInt(row[2])),
		}
	}
	return nil
}
