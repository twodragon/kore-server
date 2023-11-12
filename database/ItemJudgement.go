package database

import (
	"log"

	"github.com/twodragon/kore-server/utils"
	"github.com/xuri/excelize/v2"
)

var (
	ItemJudgements = make(map[int]*ItemJudgement)
)

type ItemJudgement struct {
	ID               int     `db:"id"`
	Name             string  `db:"name"`
	AttackBonus      int     `db:"attack_plus"`
	AccuracyBonus    int     `db:"accuracy_plus"`
	StrBonus         int     `db:"str_plus"`
	DexBonus         int     `db:"dex_plus"`
	IntBonus         int     `db:"int_plus"`
	DEF              int     `db:"def"`
	Sdef             int     `db:"sdef"`
	Wind             int     `db:"wind_plus"`
	Water            int     `db:"water_plus"`
	Fire             int     `db:"fire_plus"`
	DodgeBonus       int     `db:"extra_dodge"`
	AttackSpeedBonus int     `db:"extra_attackspeed"`
	ArtsRangeBonus   float64 `db:"extra_arts_range"`
	MaxHpBonus       int     `db:"max_hp"`
	MaxChiBonus      int     `db:"max_chi"`
	Probabilities    int     `db:"probabilities"`
}

func getItemJudgements() error {
	log.Print("Reading judgement stats...")
	f, err := excelize.OpenFile("data/tb_ItemJudgement.xlsx")
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
		ItemJudgements[utils.StringToInt(row[1])] = &ItemJudgement{
			ID:               utils.StringToInt(row[1]),
			Name:             row[2],
			AttackBonus:      utils.StringToInt(row[3]),
			AccuracyBonus:    utils.StringToInt(row[4]),
			StrBonus:         utils.StringToInt(row[5]),
			DexBonus:         utils.StringToInt(row[6]),
			IntBonus:         utils.StringToInt(row[7]),
			DEF:              utils.StringToInt(row[8]),
			Sdef:             utils.StringToInt(row[9]),
			MaxHpBonus:       utils.StringToInt(row[10]),
			MaxChiBonus:      utils.StringToInt(row[11]),
			DodgeBonus:       utils.StringToInt(row[12]),
			AttackSpeedBonus: utils.StringToInt(row[13]),
			Wind:             utils.StringToInt(row[14]),
			Water:            utils.StringToInt(row[15]),
			Fire:             utils.StringToInt(row[16]),
			ArtsRangeBonus:   utils.StringToFloat64(row[17]),
			Probabilities:    utils.StringToInt(row[19]),
		}
	}
	return nil
}
