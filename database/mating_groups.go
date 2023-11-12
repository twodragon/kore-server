package database

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/gorp.v1"
)

var (
	MatingGroups = make(map[int]*MatingGroup)
)

type MatingGroup struct {
	ID      int    `db:"id"`
	Males   string `db:"males"`
	Females string `db:"females"`
}

func (e *MatingGroup) Create() error {
	return pgsql_DbMap.Insert(e)
}

func (e *MatingGroup) CreateWithTransaction(tr *gorp.Transaction) error {
	return tr.Insert(e)
}

func (e *MatingGroup) Update() error {
	_, err := pgsql_DbMap.Update(e)
	return err
}

func (e *MatingGroup) Delete() error {
	_, err := pgsql_DbMap.Delete(e)
	return err
}
func GetAllMatingGroups() error {
	var groups []*MatingGroup
	query := `select * from mating_groups`

	if _, err := pgsql_DbMap.Select(&groups, query); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("GetAllMatingGroups: %s", err.Error())
	}

	for _, d := range groups {
		MatingGroups[d.ID] = d
	}

	return nil
}

func (e *MatingGroup) GetMales() []int {
	males := strings.Trim(e.Males, "{}")
	sMales := strings.Split(males, ",")

	var arr []int
	for _, sItem := range sMales {
		d, _ := strconv.Atoi(sItem)
		arr = append(arr, d)
	}
	return arr
}

func (e *MatingGroup) GetFemales() []int {
	females := strings.Trim(e.Females, "{}")
	sFemales := strings.Split(females, ",")

	var arr []int
	for _, sItem := range sFemales {
		d, _ := strconv.Atoi(sItem)
		arr = append(arr, d)
	}
	return arr
}
