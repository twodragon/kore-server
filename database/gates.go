package database

import (
	"log"

	"github.com/twodragon/kore-server/utils"
	"github.com/xuri/excelize/v2"
)

var (
	Gates = make(map[int]*Gate)
)

type Gate struct {
	ID                 int
	TargetMap          uint8
	Point_X            float64
	Point_Y            float64
	MinLevelRequirment int
	FactionRequirment  int
}

func GetGates() error {
	log.Print("Reading Gates table...")
	f, err := excelize.OpenFile("data/tb_ZoneChanGeTable.xlsx")
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
		index := utils.StringToInt(row[1])
		Gates[index] = &Gate{
			ID:                 utils.StringToInt(row[1]),
			TargetMap:          uint8(utils.StringToInt(row[2])),
			Point_X:            utils.StringToFloat64(row[4]),
			Point_Y:            utils.StringToFloat64(row[5]),
			MinLevelRequirment: utils.StringToInt(row[11]),
			FactionRequirment:  utils.StringToInt(row[19]),
		}
	}
	return nil
}
