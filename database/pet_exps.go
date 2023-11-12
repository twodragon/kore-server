package database

import (
	"log"

	"github.com/twodragon/kore-server/utils"
	"github.com/xuri/excelize/v2"
)

var (
	PetExps = make(map[int]*PetExpInfo)
)

type PetExpInfo struct {
	Level         int
	ReqExpEvo1    int
	ReqExpEvo2    int
	ReqExpEvo3    int
	ReqExpHt      int
	ReqExpDivEvo1 int
	ReqExpDivEvo2 int
	ReqExpDivEvo3 int
	ReqExpDivEvo4 int
}

func GetPetsExps() error {
	log.Print("Reading PetsExps table...")
	f, err := excelize.OpenFile("data/tb_PetEXPTable.xlsx")
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
		index := int(utils.StringToInt(row[1]))
		PetExps[index] = &PetExpInfo{
			Level:         utils.StringToInt(row[1]),
			ReqExpEvo1:    utils.StringToInt(row[2]),
			ReqExpEvo2:    utils.StringToInt(row[3]),
			ReqExpEvo3:    utils.StringToInt(row[4]),
			ReqExpHt:      utils.StringToInt(row[5]),
			ReqExpDivEvo1: utils.StringToInt(row[6]),
			ReqExpDivEvo2: utils.StringToInt(row[7]),
			ReqExpDivEvo3: utils.StringToInt(row[8]),
			ReqExpDivEvo4: utils.StringToInt(row[9]),
		}
	}
	return nil
}
