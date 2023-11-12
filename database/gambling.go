package database

import (
	"log"

	"github.com/twodragon/kore-server/utils"
	"github.com/xuri/excelize/v2"
)

var (
	GamblingItems = make(map[int]*Gambling)

	rewardCounts2 = map[int]map[int]uint{
		13370000: {32049: 3},
		13370001: {1031: 2, 32240: 3, 32050: 3, 221: 3, 222: 3, 223: 3},
		13370002: {92000013: 2},
		13370003: {92000012: 2, 253: 2, 17502731: 2, 17502733: 2, 240: 2, 241: 2, 232: 2, 17200187: 2},
		13370004: {92000063: 2, 17502731: 4, 417502733: 4, 240: 4, 241: 4, 253: 4, 232: 4, 17200187: 4},
		13370005: {15700001: 2},
	}
)

type Gambling struct {
	ID           int    `db:"id"`
	Cost         uint64 `db:"cost"`
	DropID       int    `db:"drop_id"`
	RewardCounts int64  `db:"reward_counts"`
}

func GetGamblings() error {
	log.Print("Reading Gamblings...")
	f, err := excelize.OpenFile("data/tb_gamblingitem.xlsx")
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
		GamblingItems[utils.StringToInt(row[1])] = &Gambling{
			ID:           utils.StringToInt(row[1]),
			Cost:         uint64(utils.StringToInt(row[3])),
			DropID:       utils.StringToInt(row[16]),
			RewardCounts: int64(utils.StringToInt(row[13])),
		}
	}
	return nil
}
