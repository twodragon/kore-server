package database

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
)

var (
	CookingItems = make(map[int]*CookingItem)
)

type CookingItem struct {
	ID            int    `db:"id"`
	Materials     string `db:"materials"`
	Amounts       string `db:"amount"`
	Productions   string `db:"production"`
	Probabilities string `db:"probabilities"`
	Cost          int64  `db:"cost"`
}

func (e *CookingItem) GetMaterials() []int {
	items := strings.Trim(e.Materials, "{}")
	sItems := strings.Split(items, ",")

	var arr []int
	for _, sItem := range sItems {
		d, _ := strconv.Atoi(sItem)
		arr = append(arr, d)
	}
	return arr
}
func (e *CookingItem) GetAmounts() []int {
	items := strings.Trim(e.Amounts, "{}")
	sItems := strings.Split(items, ",")

	var arr []int
	for _, sItem := range sItems {
		d, _ := strconv.Atoi(sItem)
		arr = append(arr, d)
	}
	return arr
}
func (e *CookingItem) GetProductions() []int {
	items := strings.Trim(e.Productions, "{}")
	sItems := strings.Split(items, ",")

	var arr []int
	for _, sItem := range sItems {
		d, _ := strconv.Atoi(sItem)
		arr = append(arr, d)
	}
	return arr
}
func (e *CookingItem) GetProbabilities() []int {
	items := strings.Trim(e.Probabilities, "{}")
	sItems := strings.Split(items, ",")

	var arr []int
	for _, sItem := range sItems {
		d, _ := strconv.Atoi(sItem)
		arr = append(arr, d)
	}
	return arr
}

func getCookingItems() error {
	var receipe []*CookingItem
	query := `select * from data.cooking`

	if _, err := pgsql_DbMap.Select(&receipe, query); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("getCookingItems: %s", err.Error())
	}

	for _, cr := range receipe {
		CookingItems[cr.ID] = cr
	}

	return nil
}

func RefreshCooking() error {
	var receipe []*CookingItem
	query := `select * from data.cooking`

	if _, err := pgsql_DbMap.Select(&receipe, query); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("getCookingItems: %s", err.Error())
	}

	for _, cr := range receipe {
		CookingItems[cr.ID] = cr
	}

	return nil
}
