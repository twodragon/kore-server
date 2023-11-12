package database

import (
	"log"

	"github.com/twodragon/kore-server/utils"
	"github.com/xuri/excelize/v2"
)

var (
	Pets = make(map[int64]*Pet)
)

type Pet struct {
	ID            int64
	Name          string
	Evolution     int
	Level         int
	TargetLevel   int
	EvolvedID     int64
	BaseSTR       int
	AdditionalSTR int
	BaseDEX       int
	AdditionalDEX int
	BaseINT       int
	AdditionalINT int
	BaseHP        int
	AdditionalHP  float64
	BaseChi       int
	AdditionalChi float64
	RunningSpeed  float64
	Skill_1       int
	Skill_2       int
	Skill_3       int
	Combat        bool
}

func GetPets() error {
	log.Print("Reading Pets table...")
	f, err := excelize.OpenFile("data/tb_PetTable.xlsx")
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
		index := int64(utils.StringToInt(row[1]))
		Pets[index] = &Pet{
			ID:            int64(utils.StringToInt(row[1])),
			Name:          row[2],
			Evolution:     utils.StringToInt(row[20]),
			Level:         utils.StringToInt(row[21]),
			TargetLevel:   utils.StringToInt(row[22]),
			EvolvedID:     int64(utils.StringToInt(row[23])),
			BaseSTR:       utils.StringToInt(row[27]),
			AdditionalSTR: utils.StringToInt(row[28]),
			BaseDEX:       utils.StringToInt(row[29]),
			AdditionalDEX: utils.StringToInt(row[30]),
			BaseINT:       utils.StringToInt(row[31]),
			AdditionalINT: utils.StringToInt(row[32]),
			BaseHP:        utils.StringToInt(row[39]),
			AdditionalHP:  utils.StringToFloat64(row[40]),
			BaseChi:       utils.StringToInt(row[41]),
			AdditionalChi: utils.StringToFloat64(row[42]),
			RunningSpeed:  utils.StringToFloat64(row[71]),
			Skill_1:       utils.StringToInt(row[75]),
			Skill_2:       utils.StringToInt(row[76]),
			Skill_3:       utils.StringToInt(row[77]),
			Combat:        false,
		}
		if utils.StringToInt(row[115]) > 0 {
			Pets[index].Combat = true
		}
	}

	return nil
}
