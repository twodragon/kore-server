package database

import (
	"log"

	"github.com/twodragon/kore-server/utils"
	"github.com/xuri/excelize/v2"
)

type Map struct {
	ID                 int
	Name               string
	IsDark             bool
	IsWarZone          bool
	MinLevelRequirment int
	MaxLevelRequirment int
	HousingMap         int
}

var (
	Maps = make(map[int]*Map)

	DKMaps = map[int16][]int16{
		18: {18, 193, 200}, 19: {19, 194, 201}, 25: {25, 195, 202}, 26: {26, 196, 203}, 27: {27, 197, 204}, 29: {29, 198, 205}, 30: {30, 199, 206}, // Normal Maps
		193: {18, 193, 200}, 194: {19, 194, 201}, 195: {25, 195, 202}, 196: {26, 196, 203}, 197: {27, 197, 204}, 198: {29, 198, 205}, 199: {30, 199, 206}, // DK Maps
		200: {18, 193, 200}, 201: {19, 194, 201}, 202: {25, 195, 202}, 203: {26, 196, 203}, 204: {27, 197, 204}, 205: {29, 198, 205}, 206: {30, 199, 206}, // Normal Maps
	}

	sharedMaps = []int16{1, 2, 3, 14, 15, 10, 20, 21, 22, 23, 26, 33, 34, 36, 37, 38, 42, 43, 44, 45, 46, 47,
		70, 72, 73, 74, 75, 89, 100, 101, 102, 109, 110, 111, 112, 120, 164, 165, 166, 167, 168, 169, 170,
		213, 214, 215, 221, 222, 223, 224, 225, 226, 227, 228, 233, 236, 237, 238, 239, 240, 243, 244, 252, 254, 255, 108, 110, 249}

	DungeonZones = []int16{229}

	PVPServers     = []int16{2, 3}
	LoseEXPServers = []int16{}

	GMRanks = []int16{2, 3, 4, 5}

	DisabledAIDMaps = []int16{212, 230, 233, 243, 255}

	PvPZones = []int16{12, 17, 100, 101, 102, 108, 109, 110, 111, 112, 255, 230, 249}

	WarMaps = []int{230, 233, 255, 249}

	ZhuangFactionMobs = []int{424203, 424204, 424205, 424206, 424207, 41766, 424201, //great war mobs
		425101, 425102, 425103, 425104, 425105, 425106, 425107, 425108, 425109, 425501, 425502, 425503, 425504} //faction war mobs
	ShaoFactionMobs = []int{424203, 424204, 424205, 424206, 424207, 41767, 424202,
		425110, 425111, 425112, 425113, 425114, 425115, 425116, 425117, 425118, 425505, 425506, 425507, 425508} //great war mobs
)

func GetMaps() error {
	log.Print("Reading Maps...")
	f, err := excelize.OpenFile("data/tb_ZoneTable.xlsx")
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
		Maps[utils.StringToInt(row[1])] = &Map{
			ID:                 utils.StringToInt(row[1]),
			Name:               row[2],
			IsWarZone:          utils.StringToBool(row[10]),
			HousingMap:         utils.StringToInt(row[13]),
			MinLevelRequirment: utils.StringToInt(row[36]),
			MaxLevelRequirment: utils.StringToInt(row[37]),
			IsDark:             !utils.StringToBool(row[41]),
		}
	}
	return nil
}
