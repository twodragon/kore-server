package database

import (
	"log"
	"strings"

	"github.com/twodragon/kore-server/utils"
	"github.com/xuri/excelize/v2"
)

var (
	SavePoints = make(map[int]*SavePoint)
)

type SavePoint struct {
	ID       int     `db:"mapid"`
	X        float64 `db:"x"`
	Y        float64 `db:"y"`
	MinLevel int     `db:"min_level"`
}

func (e *SavePoint) SetPoint(point *utils.Location) {
	e.X = point.X
	e.Y = point.Y
}

func GetAllSavePoints() error {
	log.Print("Reading Save Points...")
	f, err := excelize.OpenFile("data/tb_SavePoint.xlsx")
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
		SavePoints[utils.StringToInt(row[1])] = &SavePoint{
			ID:       utils.StringToInt(row[1]),
			X:        utils.StringToFloat64(row[2]),
			Y:        utils.StringToFloat64(row[3]),
			MinLevel: 0,
		}
	}
	return nil
}

func ConvertPointToLocation(point string) *utils.Location {

	location := &utils.Location{}
	parts := strings.Split(strings.Trim(point, "()"), ",")
	if parts[0] != "" && parts[1] != "" {
		location.X = utils.StringToFloat64(parts[0])
		location.Y = utils.StringToFloat64(parts[1])
	} else {
		location.X = 0
		location.Y = 0
	}
	return location
}
func CoordPoint(x float64, y float64) *utils.Location {
	location := &utils.Location{X: x, Y: y}
	return location
}
