package database

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
)

type CheckinReward struct {
	Day       int    `db:"day"`
	ItemIds   string `db:"item_ids"`
	Quantitys string `db:"quantitys"`

	ItemIdsArr   []int64 `db:"-"`
	QuantitysArr []int   `db:"-"`
}

func (reward *CheckinReward) GetItems() []int64 {

	var items []int64

	upgs := strings.Split(strings.Trim(string(reward.ItemIds), "{}"), ",")

	for _, a := range upgs {

		item, _ := strconv.ParseInt(a, 10, 64)
		items = append(items, item)
	}

	return items
}

func (reward *CheckinReward) GetQtys() []int {

	var qtys []int
	upgs := strings.Split(strings.Trim(string(reward.Quantitys), "{}"), ",")

	for _, a := range upgs {

		qty, _ := strconv.ParseInt(a, 10, 64)
		qtys = append(qtys, int(qty))
	}
	return qtys
}

var (
	CheckinRewards = make(map[int]*CheckinReward)
)

func (cr *CheckinReward) Create() error {
	return pgsql_DbMap.Insert(cr)
}

func (cr *CheckinReward) Update() error {

	_, err := pgsql_DbMap.Update(cr)
	if err != nil {
		log.Println(err)
	}
	return err
}

func GetAllCheckinRewards() error {
	var arr []*CheckinReward
	query := `select * from data.checkin_rewards order by day asc`

	if _, err := pgsql_DbMap.Select(&arr, query); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("GetAllCheckinRewards: %s", err.Error())
	}

	for _, a := range arr {
		a.ItemIdsArr = a.GetItems()
		a.QuantitysArr = a.GetQtys()
		CheckinRewards[a.Day] = a
	}

	return nil
}
