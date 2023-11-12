package database

import (
	"log"

	"github.com/twodragon/kore-server/utils"
	"github.com/xuri/excelize/v2"
)

type Fusion struct {
	Item1            int
	Item2            int
	Count2           int
	Item3            int
	Count3           int
	SpecialItem      int
	SpecialItemCount int
	Probability      int
	Cost             int
	Production       int
	DestroyOnFail    bool
}

var (
	Fusions = make(map[int]*Fusion)
)

func GetAdvancedFusions() error {
	log.Print("Reading AdvancedFusions...")
	f, err := excelize.OpenFile("data/tb_ItemMakingPI.xlsx")
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
		Fusions[utils.StringToInt(row[1])] = &Fusion{
			Item1:            utils.StringToInt(row[1]),
			Item2:            utils.StringToInt(row[2]),
			Count2:           utils.StringToInt(row[3]),
			Item3:            utils.StringToInt(row[4]),
			Count3:           utils.StringToInt(row[5]),
			SpecialItem:      utils.StringToInt(row[6]),
			SpecialItemCount: utils.StringToInt(row[7]),
			Probability:      utils.StringToInt(row[8]),
			Cost:             utils.StringToInt(row[9]),
			Production:       utils.StringToInt(row[10]),
			DestroyOnFail:    utils.StringToBool(row[11]),
		}
	}

	return nil
}
