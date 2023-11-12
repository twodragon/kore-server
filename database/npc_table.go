package database

import (
	"fmt"
	"log"
	"sync"

	"github.com/twodragon/kore-server/utils"
	"github.com/xuri/excelize/v2"
	gorp "gopkg.in/gorp.v1"
)

var (
	NPCInfos      = make(map[int]*NPC)
	NPCInfosMutex sync.RWMutex
	//bosses        = []int{}
	bosses = []int{40951, 41171, 41371, 41381, 41671, 41851, 41852, 41853, 41941, 42452, 42561, 42562, 420108, 430108,
		42910, 43111, 43708, 43707, 44026, 44502, 44009, 495007,
		499007, 42912, 42913}
)

type NPC struct {
	ID              int
	Name            string
	Type            int
	Level           int16
	Exp             int64
	DivineExp       int64
	DarknessExp     int64
	GoldDrop        int
	DEF             int
	SkillDEF        int
	MaxHp           int
	MaxChi          int
	MinATK          int
	MaxATK          int
	MinArtsATK      int
	MaxArtsATK      int
	DropID          int
	SkillIds        []int
	PoisonATK       int
	PoisonDef       int
	PoisInflictTime int
	ParalysisATK    int
	ParalysisDef    int
	ParaInflictTime int
	ConfusionATK    int
	ConfusionDef    int
	ConfInflictTime int
	HpRecovery      int
	ChiRecovery     int

	WalkingSpeed int
	RunningSpeed int

	StageLink int
	Test      int
}

func GetNpcInfo(id int) (*NPC, bool) {
	NPCInfosMutex.RLock()
	defer NPCInfosMutex.RUnlock()
	info, ok := NPCInfos[id]
	return info, ok
}
func SetNpcInfo(npc *NPC) {
	NPCInfosMutex.Lock()
	defer NPCInfosMutex.Unlock()
	NPCInfos[npc.ID] = npc
}

func (e *NPC) Create() error {
	return pgsql_DbMap.Insert(e)
}

func (e *NPC) CreateWithTransaction(tr *gorp.Transaction) error {
	return tr.Insert(e)
}

func (e *NPC) Update() error {
	_, err := pgsql_DbMap.Update(e)
	return err
}

func (e *NPC) Delete() error {
	_, err := pgsql_DbMap.Delete(e)
	return err
}

func SpawnMob(npcPos *NpcPosition, newai *AI) error {

	npcinfo, ok := GetNpcInfo(npcPos.NPCID)
	if !ok || npcinfo == nil {
		return fmt.Errorf("SpawnMob: npc not found")
	}

	newai.Handler = newai.AIHandler

	AIsByMap[newai.Server][newai.Map] = append(AIsByMap[newai.Server][newai.Map], newai)

	if newai.WalkingSpeed > 0 {
		go newai.Handler()
	}

	return nil
}

func ReadAllNPCsInfo() error {
	log.Print("Reading NPCTable...")
	f, err := excelize.OpenFile("data/tb_NPCTable.xlsx")
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
		id := utils.StringToInt(row[1])
		NPC := &NPC{
			ID:   id,
			Name: row[2],
			//			Type:        utils.StringToInt(row[11]),
			//			Level:       int16(utils.StringToInt(row[15])),
			Exp:         int64(utils.StringToInt(row[16])),
			DivineExp:   int64(utils.StringToInt(row[17])),
			DarknessExp: int64(utils.StringToInt(row[18])),

			StageLink: utils.StringToInt(row[20]),

			DropID:   utils.StringToInt(row[25]),
			GoldDrop: utils.StringToInt(row[26]),
			DEF:      utils.StringToInt(row[30]),
			SkillDEF: utils.StringToInt(row[31]),

			MaxHp:        utils.StringToInt(row[36]),
			HpRecovery:   utils.StringToInt(row[37]),
			MaxChi:       utils.StringToInt(row[38]),
			ChiRecovery:  utils.StringToInt(row[39]),
			WalkingSpeed: utils.StringToInt(row[41]),
			RunningSpeed: utils.StringToInt(row[42]),
			MinATK:       utils.StringToInt(row[45]),
			MaxATK:       utils.StringToInt(row[46]),
			MinArtsATK:   utils.StringToInt(row[47]),
			MaxArtsATK:   utils.StringToInt(row[48]),

			SkillIds: []int{utils.StringToInt(row[67]), utils.StringToInt(row[68]), utils.StringToInt(row[69])},

			PoisonATK:       utils.StringToInt(row[54]),
			PoisonDef:       utils.StringToInt(row[55]),
			PoisInflictTime: utils.StringToInt(row[56]),

			ParalysisATK:    utils.StringToInt(row[57]),
			ParalysisDef:    utils.StringToInt(row[58]),
			ParaInflictTime: utils.StringToInt(row[59]),

			ConfusionATK:    utils.StringToInt(row[60]),
			ConfusionDef:    utils.StringToInt(row[61]),
			ConfInflictTime: utils.StringToInt(row[62]),
			Test:            utils.StringToInt(row[102]),
		}
		SetNpcInfo(NPC)

	}
	f.Save()

	return nil
}
