package database

import (
	"log"

	"github.com/twodragon/kore-server/utils"
	"github.com/xuri/excelize/v2"
)

var (
	StarterItems = make(map[int]*StarterItem)
)

type StarterItem struct {
	CharType int
	ItemIDs  []int64
}

func GetStarerItems() error {
	log.Print("Reading StarterItems...")
	f, err := excelize.OpenFile("data/tb_Starteritem.xlsx")
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
		chartype := utils.StringToInt(row[1])
		StarterItems[chartype] = &StarterItem{
			CharType: chartype,
		}
		for i := 2; i <= 21; i++ {
			if utils.StringToInt(row[i]) == 0 {
				continue
			}
			StarterItems[chartype].ItemIDs = append(StarterItems[chartype].ItemIDs, int64(utils.StringToInt(row[i])))
		}
	}

	return nil
}
