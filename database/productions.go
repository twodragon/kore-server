package database

import (
	"log"
	"sync"

	"github.com/twodragon/kore-server/utils"
	"github.com/xuri/excelize/v2"
)

var (
	ProductionList  = make(map[int]*Production)
	ProductionMutex sync.RWMutex
)

type Production struct {
	ID           int   `db:"id"`
	Item2        int64 `db:"item2"`
	Count2       int   `db:"count2"`
	Item3        int64 `db:"item3"`
	Count3       int   `db:"count3"`
	Special      int64 `db:"special"`
	SpecialCount int   `db:"special_count"`
	Probability  int   `db:"probability"`
	Cost         int64 `db:"cost"`
	Production   int   `db:"production"`
	KeepTheBook  bool  `db:"keepthebook"`
}

func GetProductionsList() []*Production {
	ProductionMutex.RLock()
	defer ProductionMutex.RUnlock()
	var list []*Production
	for _, v := range ProductionList {
		list = append(list, v)
	}
	return list
}
func GetProductionById(id int) (*Production, bool) {
	ProductionMutex.RLock()
	defer ProductionMutex.RUnlock()
	prod, ok := ProductionList[id]
	return prod, ok
}
func SetProductionsOnList(prod *Production) {
	ProductionMutex.Lock()
	defer ProductionMutex.Unlock()
	ProductionList[prod.ID] = prod
}

func GetProductions() error {
	log.Print("Reading MeltingTable...")
	f, err := excelize.OpenFile("data/tb_ItemMaking.xlsx")
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
		item := &Production{
			ID:           utils.StringToInt(row[1]),
			Item2:        int64(utils.StringToInt(row[2])),
			Count2:       utils.StringToInt(row[3]),
			Item3:        int64(utils.StringToInt(row[4])),
			Count3:       utils.StringToInt(row[5]),
			Special:      int64(utils.StringToInt(row[6])),
			SpecialCount: utils.StringToInt(row[7]),
			Probability:  utils.StringToInt(row[8]),
			Cost:         int64(utils.StringToInt(row[9])),
			Production:   utils.StringToInt(row[10]),
		}

		SetProductionsOnList(item)
	}
	return err
}
