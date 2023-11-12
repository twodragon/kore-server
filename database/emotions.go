package database

import (
	"log"

	"github.com/twodragon/kore-server/utils"
	"github.com/xuri/excelize/v2"
)

type Emotion struct {
	ID          int
	Cmd         string
	Type        int
	AnimationID int
}

var (
	Emotions = make(map[int]*Emotion)
)

func GetEmotions() error {
	log.Print("Reading Emotions...")
	f, err := excelize.OpenFile("data/tb_EMotion.xlsx")
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
		Emotions[utils.StringToInt(row[1])] = &Emotion{
			ID:          utils.StringToInt(row[1]),
			Cmd:         row[2],
			Type:        utils.StringToInt(row[5]),
			AnimationID: utils.StringToInt(row[6]),
		}
	}
	return nil
}
