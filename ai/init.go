package ai

import (
	"log"

	"github.com/twodragon/kore-server/database"
	"github.com/twodragon/kore-server/utils"
)

func Init() error {

	database.AIsByMap = make([]map[int16][]*database.AI, database.SERVER_COUNT+1)
	for s := 0; s <= database.SERVER_COUNT; s++ {
		database.AIsByMap[s] = make(map[int16][]*database.AI)
	}
	database.DungeonsByMap = make([]map[int16]int, database.SERVER_COUNT+1)
	for s := 0; s <= database.SERVER_COUNT; s++ {
		database.DungeonsByMap[s] = make(map[int16]int)
	}
	database.DungeonsAiByMap = make([]map[int16][]*database.AI, database.SERVER_COUNT+1)
	for s := 0; s <= database.SERVER_COUNT; s++ {
		database.DungeonsAiByMap[s] = make(map[int16][]*database.AI)
	}
	func() {
		<-database.InitRegisters

		var err error

		err = database.GetAllNPCPos()
		if err != nil {
			log.Println(err)
			return
		}

		for _, pos := range database.GetNPCPostions() {
			if pos.IsNPC && !pos.Attackable {
				database.GenerateIDForNPC(pos)
			}
		}

		err = database.GetAllAI()
		if err != nil {
			log.Println(err)
			return
		}

		for _, ai := range database.AIs {
			database.AIsByMap[ai.Server][ai.Map] = append(database.AIsByMap[ai.Server][ai.Map], ai)
		}

		for _, AI := range database.AIs {

			if AI.ID == 0 {
				continue
			}

			pos := database.GetNPCPosByID(AI.PosID)
			if pos == nil {
				continue
			}
			npc, ok := database.GetNpcInfo(pos.NPCID)
			if !ok {
				log.Println(pos.NPCID)
				continue
			}

			AI.NPCpos = pos
			AI.WalkingSpeed = float64(npc.WalkingSpeed)
			AI.RunningSpeed = float64(npc.RunningSpeed)
			minLoc := database.ConvertPointToLocation(pos.MinLocation)
			maxLoc := database.ConvertPointToLocation(pos.MaxLocation)
			loc := utils.Location{X: utils.RandFloat(minLoc.X, maxLoc.X), Y: utils.RandFloat(minLoc.Y, maxLoc.Y)}
			AI.Coordinate = loc.String()

			AI.TargetLocation = *database.ConvertPointToLocation(AI.Coordinate)
			AI.HP = npc.MaxHp
			AI.CHI = npc.MaxChi
			AI.OnSightPlayers = make(map[int]interface{})
			AI.Handler = AI.AIHandler

			database.GenerateIDForAI(AI)
			/*	if funk.Contains(database.WarStonesIDs, npc.ID) { //115512, 115519, 115526, 115534, 115541
				newStone := &database.WarStone{PseudoID: AI.PseudoID, NpcID: pos.NPCID, ConquereValue: 100, AIid: AI.ID}
				database.WarStonesPseudoIDs = append(database.WarStonesPseudoIDs, AI.PseudoID)
				database.WarStones[int(AI.PseudoID)] = newStone
			}*/
			if AI.ID == 115512 || AI.ID == 115519 || AI.ID == 115526 || AI.ID == 115534 || AI.ID == 115541 {

				newStone := &database.WarStone{PseudoID: AI.PseudoID, NpcID: pos.NPCID, ConquereValue: 100}
				database.WarStonesIDs = append(database.WarStonesIDs, AI.PseudoID)
				database.WarStones[int(AI.PseudoID)] = newStone
			}
			if AI.WalkingSpeed > 0 {
				go AI.Handler()
			}
			if AI.IsDead {
				AI.Kill()
			}
			//log.Println(fmt.Sprintf("AI: %d", AI.ID))
		}
	}()
	return nil
}

func InitBabyPets() {

	database.BabyPetsByMap = make([]map[int16][]*database.BabyPet, database.SERVER_COUNT+1)
	for s := 0; s <= database.SERVER_COUNT; s++ {
		database.BabyPetsByMap[s] = make(map[int16][]*database.BabyPet)
	}

	for _, pos := range database.BabyPets {
		database.GenerateIDForBabyPet(pos)
		pos.OnSightPlayers = make(map[int]interface{})
		pos.Handler = pos.BabyPetHandler
		go pos.Handler()

	}

	for _, bby := range database.BabyPets {
		database.BabyPetsByMap[bby.Server][bby.Mapid] = append(database.BabyPetsByMap[bby.Server][bby.Mapid], bby)
	}

}
func InitHouseItems() {

	database.HousingItemsByMap = make([]map[int16][]*database.HousingItem, 10000)
	for s := 0; s <= database.SERVER_COUNT; s++ {
		database.HousingItemsByMap[s] = make(map[int16][]*database.HousingItem)
	}

	for _, pos := range database.HousingItems {
		database.HousingItemsByMap[pos.OwnerID] = make(map[int16][]*database.HousingItem)
		database.GenerateIDForHouse(pos)
		pos.OnSightPlayers = make(map[int]interface{})

	}

	for _, bby := range database.HousingItems {
		database.HousingItemsByMap[bby.Server][bby.MapID] = append(database.HousingItemsByMap[bby.Server][bby.MapID], bby)
	}
}
