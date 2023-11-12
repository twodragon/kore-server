package database

import (
	"log"
	"sync"

	"github.com/twodragon/kore-server/utils"
	"github.com/xuri/excelize/v2"
)

type ItemMelting struct {
	ID                     uint
	MeltedItems            []uint
	Quantities             []uint
	GoldMultiplier         float32
	Probability            uint16
	Cost                   uint32
	SpecialItem            uint
	SpecialItemProbability uint16
	IsDismantle            bool
}

var (
	Meltings         = make(map[int]*ItemMelting)
	GetMeltingsMutex sync.RWMutex
)

func ReadMeltings() error {
	log.Print("Reading MeltingTable...")
	f, err := excelize.OpenFile("data/tb_ItemMelting.xlsx")
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
		item := &ItemMelting{
			ID:                     uint(utils.StringToInt(row[1])),
			GoldMultiplier:         float32(utils.StringToFloat64(row[8])),
			Probability:            uint16(utils.StringToInt(row[10])),
			Cost:                   uint32(utils.StringToInt(row[11])),
			SpecialItem:            uint(utils.StringToInt(row[12])),
			SpecialItemProbability: uint16(utils.StringToInt(row[13])),
			IsDismantle:            utils.StringToBool(row[14]),
		}

		for i := 2; i <= 7; i++ {
			if i < 5 { //
				item.MeltedItems = append(item.MeltedItems, uint(utils.StringToInt(row[i])))
			} else { //}
				item.Quantities = append(item.Quantities, uint(utils.StringToInt(row[i])))
			}
		}
		GetMeltingsMutex.Lock()
		Meltings[utils.StringToInt(row[1])] = item
		GetMeltingsMutex.Unlock()
	}

	return nil
}

func GetMeltings() map[int]*ItemMelting {
	GetMeltingsMutex.RLock()
	defer GetMeltingsMutex.RUnlock()
	return Meltings
}
