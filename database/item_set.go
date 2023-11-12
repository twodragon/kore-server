package database

import (
	"log"
	"sync"

	"github.com/twodragon/kore-server/utils"
	"github.com/xuri/excelize/v2"
)

var (
	ItemSets           = make(map[int]*ItemSet)
	ItemSetsTableMutex sync.RWMutex
)

type ItemSet struct {
	ID           uint32
	SetItemCount int
	SetItemsIDs  []int64
	SetBonusIDs  []int64
}

func ReadItemSets() error {
	log.Print("Reading Item Sets...")
	f, err := excelize.OpenFile("data/tb_ItemSet.xlsx")
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
		item := &ItemSet{
			ID:           uint32(utils.StringToInt(row[1])),
			SetItemCount: int(utils.StringToInt(row[2])),
		}
		for i := 3; i <= 23; i++ {
			if i < 12 { //
				item.SetItemsIDs = append(item.SetItemsIDs, int64(utils.StringToInt(row[i])))
			} else { //}
				item.SetBonusIDs = append(item.SetBonusIDs, int64(utils.StringToInt(row[i])))
			}
		}
		ItemSetsTableMutex.Lock()
		ItemSets[utils.StringToInt(row[1])] = item
		ItemSetsTableMutex.Unlock()
	}

	return nil
}
func GetItemsSets() map[int]*ItemSet {
	ItemSetsTableMutex.RLock()
	defer ItemSetsTableMutex.RUnlock()
	return ItemSets
}
