package database

import (
	"log"
	"sync"

	"github.com/twodragon/kore-server/utils"
	"github.com/xuri/excelize/v2"
)

var (
	DropsInfos      = make(map[int]*DropInfo)
	DropsInfosMutex = sync.RWMutex{}
)

type DropInfo struct {
	ID            int
	Items         []int
	Probabilities []int
}

func GetDropInfo(id int) (*DropInfo, bool) {
	DropsInfosMutex.RLock()
	defer DropsInfosMutex.RUnlock()
	drop, ok := DropsInfos[id]
	return drop, ok
}
func SetDropInfo(drop *DropInfo) {
	DropsInfosMutex.Lock()
	defer DropsInfosMutex.Unlock()
	DropsInfos[drop.ID] = drop
}

func ReadAllDropsInfo() error {
	log.Print("Reading drops table...")
	f, err := excelize.OpenFile("data/tb_DropItem.xlsx")
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
		if index <= 1 {
			continue
		}
		drop := &DropInfo{
			ID: utils.StringToInt(row[1]),
		}
		for i := 2; i <= 21; i++ {
			itemid := utils.StringToInt(row[i])
			if itemid == 0 {
				continue
			}
			drop.Items = append(drop.Items, itemid)
		}
		for i := 22; i <= 41; i++ {
			prob := utils.StringToInt(row[i])
			if prob == 0 {
				continue
			}
			drop.Probabilities = append(drop.Probabilities, prob)
		}
		SetDropInfo(drop)
	}
	return nil
}
