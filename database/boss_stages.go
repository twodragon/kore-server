package database

import (
	"log"

	"github.com/twodragon/kore-server/utils"
	"github.com/xuri/excelize/v2"
)

var (
	BossStages = make(map[int]*BossStage)
)

type BossStage struct {
	ID     int
	Stages []int
}

func GetBossStages() error {
	log.Print("Reading Bosses Stages...")
	f, err := excelize.OpenFile("data/tb_linknpctable.xlsx")
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
		BossStages[utils.StringToInt(row[1])] = &BossStage{
			ID: utils.StringToInt(row[1]),
		}
		BossStages[utils.StringToInt(row[1])].Stages = append(BossStages[utils.StringToInt(row[1])].Stages, utils.StringToInt(row[2]))
		BossStages[utils.StringToInt(row[1])].Stages = append(BossStages[utils.StringToInt(row[1])].Stages, utils.StringToInt(row[3]))
		BossStages[utils.StringToInt(row[1])].Stages = append(BossStages[utils.StringToInt(row[1])].Stages, utils.StringToInt(row[4]))
		BossStages[utils.StringToInt(row[1])].Stages = append(BossStages[utils.StringToInt(row[1])].Stages, utils.StringToInt(row[5]))
		BossStages[utils.StringToInt(row[1])].Stages = append(BossStages[utils.StringToInt(row[1])].Stages, utils.StringToInt(row[6]))

	}

	return nil
}
