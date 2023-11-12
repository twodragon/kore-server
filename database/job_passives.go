package database

import (
	"log"

	"github.com/twodragon/kore-server/utils"
	"github.com/xuri/excelize/v2"
)

type JobPassive struct {
	ID                     int8
	BaseHp                 int
	AdditionalHp           int
	BaseChi                int
	AdditionalChi          int
	BaseATK                int
	BaseArtsATK            int
	BaseDEF                int
	BaseArtsDef            int
	AdditionalArtsDef      int
	BaseAccuracy           int
	BaseDodge              int
	BaseConfusionDEF       int
	BasePoisonDEF          int
	BaseParalysisDEF       int
	BaseHPRecoveryRate     int
	RunningSpeed           float64
	AdditionalRunningSpeed float64

	AdditionalHPRecoveryRate  int
	BaseChiRecoveryRate       int
	AdditionalChiRecoveryRate int
	AdditionalATK             int
	AdditionalArtsATK         int
	BaseMinDmg                int
	AdditionalMinDmg          int
	BaseMaxDmg                int
	AdditionalMaxDmg          int
	AdditionalDEF             int
	AdditionalAccuracy        int
	AdditionalDodge           int
	AdditionalConfusionDEF    int
	AdditionalPoisonDEF       int
	AddtitionalParalysisDEF   int

	JobRequirement     int
	MinLevelRequirment int
	MaxPlus            int
}

var (
	JobPassives = make(map[int64]*JobPassive)
)

func GetJobPassives() error {
	log.Print("Reading Passive skills...")
	f, err := excelize.OpenFile("data/tb_ShinkungTable.xlsx")
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
		JobPassives[int64(utils.StringToInt(row[1]))] = &JobPassive{
			ID:                int8(utils.StringToInt(row[1])),
			BaseArtsDef:       utils.StringToInt(row[66]),
			AdditionalArtsDef: utils.StringToInt(row[67]),
			BaseHp:            utils.StringToInt(row[36]),
			AdditionalHp:      utils.StringToInt(row[37]),

			BaseHPRecoveryRate:       utils.StringToInt(row[38]),
			AdditionalHPRecoveryRate: utils.StringToInt(row[39]),

			BaseChi:                   utils.StringToInt(row[40]),
			AdditionalChi:             utils.StringToInt(row[41]),
			BaseChiRecoveryRate:       utils.StringToInt(row[38]),
			AdditionalChiRecoveryRate: utils.StringToInt(row[39]),

			RunningSpeed:            utils.StringToFloat64(row[44]),
			AdditionalRunningSpeed:  utils.StringToFloat64(row[45]),
			BaseATK:                 utils.StringToInt(row[56]),
			AdditionalATK:           utils.StringToInt(row[57]),
			BaseArtsATK:             utils.StringToInt(row[58]),
			AdditionalArtsATK:       utils.StringToInt(row[59]),
			BaseMinDmg:              utils.StringToInt(row[48]),
			AdditionalMinDmg:        utils.StringToInt(row[49]),
			BaseMaxDmg:              utils.StringToInt(row[50]),
			AdditionalMaxDmg:        utils.StringToInt(row[51]),
			BaseDEF:                 utils.StringToInt(row[60]),
			AdditionalDEF:           utils.StringToInt(row[61]),
			BaseAccuracy:            utils.StringToInt(row[72]),
			AdditionalAccuracy:      utils.StringToInt(row[73]),
			BaseDodge:               utils.StringToInt(row[74]),
			AdditionalDodge:         utils.StringToInt(row[75]),
			BaseConfusionDEF:        utils.StringToInt(row[28]),
			AdditionalConfusionDEF:  utils.StringToInt(row[29]),
			BasePoisonDEF:           utils.StringToInt(row[30]),
			AdditionalPoisonDEF:     utils.StringToInt(row[31]),
			BaseParalysisDEF:        utils.StringToInt(row[32]),
			AddtitionalParalysisDEF: utils.StringToInt(row[33]),
			JobRequirement:          utils.StringToInt(row[7]),
			MinLevelRequirment:      utils.StringToInt(row[9]),
			MaxPlus:                 utils.StringToInt(row[10]),
		}
	}

	return nil
}
