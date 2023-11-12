package database

import (
	"log"

	"github.com/twodragon/kore-server/utils"
	"github.com/xuri/excelize/v2"
)

var (
	SkillInfos  = make(map[int]*SkillInfo)
	SkillBooks  = make(map[int64]*SkillBook)
	SkillPoints = make([]uint64, 12000)
)

type SkillInfo struct {
	ID                        int
	BookID                    int64
	Name                      string
	Target                    int8
	PassiveType               uint8
	Type                      uint8
	MaxPlus                   int8
	Slot                      int
	BaseTime                  int
	AdditionalTime            int
	CastTime                  float64
	BaseChi                   int
	AdditionalChi             int
	BaseMinMultiplier         int
	AdditionalMinMultiplier   int
	BaseMaxMultiplier         int
	AdditionalMaxMultiplier   int
	BaseRadius                float64
	AdditionalRadius          float64
	Passive                   bool
	BasePassive               int
	AdditionalPassive         float64
	InfectionID               int
	AreaCenter                int
	Cooldown                  float64
	PoisonDamage              int
	AdditionalPoisonDamage    int
	ConfusionDamage           int
	AdditionalConfusionDamage int
	ParaDamage                int
	AdditionalParaDamage      int
	BaseMinHP                 int
	AdditionalMinHP           int
	BaseMaxHP                 int
	AdditionalMaxHP           int
	CharType                  int
	RangeDistance             int
	IsIncreasing              bool
}

type SkillBook struct {
	ID        int64
	Name      string
	MinLevel  int
	Type      int
	SkillTree []*SkillInfo
}

func GetSkills() error {
	log.Print("Reading SkillTable...")
	f, err := excelize.OpenFile("data/tb_SkillTable.xlsx")
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
		skillinfo := &SkillInfo{
			ID:                        utils.StringToInt(row[1]),
			BookID:                    int64(utils.StringToInt(row[118])),
			Name:                      row[2],
			Target:                    int8(utils.StringToInt(row[8])),
			PassiveType:               uint8(utils.StringToInt(row[9])),
			IsIncreasing:              utils.StringToBool(row[12]),
			Type:                      uint8(utils.StringToInt(row[13])),
			MaxPlus:                   int8(utils.StringToInt(row[16])),
			Slot:                      utils.StringToInt(row[17]),
			BaseTime:                  utils.StringToInt(row[44]),
			AdditionalTime:            utils.StringToInt(row[45]),
			CastTime:                  utils.StringToFloat64(row[21]),
			BaseChi:                   utils.StringToInt(row[48]),
			AdditionalChi:             utils.StringToInt(row[49]),
			BaseMinMultiplier:         utils.StringToInt(row[54]),
			AdditionalMinMultiplier:   utils.StringToInt(row[55]),
			BaseMaxMultiplier:         utils.StringToInt(row[56]),
			AdditionalMaxMultiplier:   utils.StringToInt(row[57]),
			RangeDistance:             utils.StringToInt(row[58]),
			BaseRadius:                utils.StringToFloat64(row[60]),
			AdditionalRadius:          utils.StringToFloat64(row[61]),
			Passive:                   utils.StringToBool(row[12]),
			BasePassive:               utils.StringToInt(row[65]),
			AdditionalPassive:         utils.StringToFloat64(row[66]),
			InfectionID:               utils.StringToInt(row[122]),
			AreaCenter:                utils.StringToInt(row[10]),
			Cooldown:                  utils.StringToFloat64(row[62]),
			PoisonDamage:              utils.StringToInt(row[67]),
			AdditionalPoisonDamage:    utils.StringToInt(row[68]),
			ConfusionDamage:           utils.StringToInt(row[69]),
			AdditionalConfusionDamage: utils.StringToInt(row[70]),
			ParaDamage:                utils.StringToInt(row[71]),
			AdditionalParaDamage:      utils.StringToInt(row[72]),
			BaseMinHP:                 utils.StringToInt(row[99]),
			AdditionalMinHP:           utils.StringToInt(row[100]),
			BaseMaxHP:                 utils.StringToInt(row[101]),
			AdditionalMaxHP:           utils.StringToInt(row[102]),
			CharType:                  utils.StringToInt(row[20]),
		}
		SkillInfos[utils.StringToInt(row[1])] = skillinfo
	}

	for i := uint64(0); i < uint64(len(SkillPoints)); i++ {
		SkillPoints[i] = 2000 * i * i
	}

	return GetSkillBooks()
}
func GetSkillBooks() error {
	log.Print("Reading UkungTable...")
	f, err := excelize.OpenFile("data/tb_UkungTable.xlsx")
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
		skillbook := &SkillBook{
			ID:   int64(utils.StringToInt(row[1])),
			Name: row[2],
			Type: utils.StringToInt(row[5]),
		}
		for i := 22; i <= 45; i++ {
			if SkillInfos[utils.StringToInt(row[i])] == nil || utils.StringToInt(row[i]) == 0 {
				continue
			}
			skillbook.SkillTree = append(skillbook.SkillTree, SkillInfos[utils.StringToInt(row[i])])
		}
		SkillBooks[int64(utils.StringToInt(row[1]))] = skillbook
	}

	return nil
}
