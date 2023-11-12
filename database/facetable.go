package database

import (
	"log"

	"github.com/twodragon/kore-server/utils"
	"github.com/xuri/excelize/v2"
)

var (
	FaceItems = make(map[int]*FaceItem)
)

type FaceItem struct {
	ID     int
	ItemID int64
}

func GetFaceItem() error {
	log.Print("Reading Face table...")
	f, err := excelize.OpenFile("data/tb_FaceTable.xlsx")
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
		FaceItems[utils.StringToInt(row[1])] = &FaceItem{
			ID:     utils.StringToInt(row[1]),
			ItemID: int64(utils.StringToInt(row[2])),
		}
	}
	return nil
}
