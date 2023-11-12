package database

import (
	"log"

	"github.com/twodragon/kore-server/utils"
	"github.com/xuri/excelize/v2"
)

var (
	HouseItemsInfos = make(map[int]*HouseItemInfo)
)

type HouseItemInfo struct {
	ID          int
	Name        string
	Category    int
	Type        int
	ItemID      int64
	Description string
	Relaxetion  int
	Map         int16
	Timer       int
	NextStage   int
	DropID      int

	CanCollect int
}

func GetHouseItems() error {
	log.Print("Reading House items...")
	f, err := excelize.OpenFile("data/tb_House_NPCTable.xlsx")
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
		HouseItemsInfos[utils.StringToInt(row[1])] = &HouseItemInfo{
			ID:       utils.StringToInt(row[1]),
			Name:     row[2],
			Category: utils.StringToInt(row[7]),
			Type:     utils.StringToInt(row[8]),
			ItemID:   int64(utils.StringToInt(row[17])),
			//			Description: row[21],
			Relaxetion: utils.StringToInt(row[10]),
			Map:        int16(utils.StringToInt(row[18])),
			Timer:      utils.StringToInt(row[12]),
			NextStage:  utils.StringToInt(row[13]),
			CanCollect: utils.StringToInt(row[14]),
			DropID:     utils.StringToInt(row[16]),
		}
	}
	return nil
}
