package database

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/twodragon/kore-server/utils"
)

var (
	MapRegister    = make([]map[int16]map[uint16]interface{}, SERVER_COUNT+1)
	mrMutex        sync.RWMutex
	PlayerRegister = make(map[uint16]interface{}, SERVER_COUNT+1)
	prMutex        sync.RWMutex
	InitRegisters  = make(chan bool, 1)
)

func init() {

	for j := 0; j <= SERVER_COUNT; j++ {
		MapRegister[j] = make(map[int16]map[uint16]interface{})
	}

	for i := int16(1); i <= 255; i++ {
		for j := 0; j <= SERVER_COUNT; j++ {
			MapRegister[j][i] = make(map[uint16]interface{})
		}
	}

	RemoveFromRegister = func(c *Character) {
		pseudo := c.PseudoID
		c.PseudoID = 0

		time.AfterFunc(time.Second*10, func() {
			prMutex.Lock()
			defer prMutex.Unlock()
			delete(PlayerRegister, pseudo)
		})
	}

	RemovePetFromRegister = func(c *Character) {
		user, err := FindUserByID(c.UserID)
		if err != nil || user == nil {
			log.Println("RemoveFromRegister failed:", err)
			return
		}

		slots, err := c.InventorySlots()
		if err != nil {
			return
		}

		pet := slots[0x0A].Pet
		if pet == nil || pet.PseudoID == 0 {
			return
		}

		mrMutex.Lock()
		defer mrMutex.Unlock()
		delete(MapRegister[user.ConnectedServer][c.Map], uint16(pet.PseudoID))
		pet.PseudoID = 0
	}

	GetFromRegister = func(server int, mapID int16, ID uint16) interface{} {
		mrMutex.RLock()
		defer mrMutex.RUnlock()
		return MapRegister[server][mapID][ID]
	}

	FindCharacterByPseudoID = FindCharacter
	GeneratePetID = GenerateIDForPet

	InitRegisters <- true
}

func GenerateIDforCharacter(character *Character) error {

	prMutex.Lock()
	defer prMutex.Unlock()
	for i := uint16(1); i <= 2000; i++ {
		if _, ok := PlayerRegister[i]; !ok {
			log.Println(i)
			PlayerRegister[i] = character
			character.PseudoID = i
			return nil
		}
	}

	return fmt.Errorf("all pseudo ids taken")
}

func GenerateIDForAI(AI *AI) {
	mrMutex.Lock()
	defer mrMutex.Unlock()
	for {
		i := uint16(utils.RandInt(40000, 50000))
		if _, ok := MapRegister[AI.Server][AI.Map][i]; !ok {
			AI.PseudoID = i
			MapRegister[AI.Server][AI.Map][i] = AI
			return
		}
	}
}

func GenerateIDForPet(owner *Character, pet *PetSlot) {
	mrMutex.Lock()
	defer mrMutex.Unlock()

	server := owner.Socket.User.ConnectedServer
	i := 2500 + owner.PseudoID
	if _, ok := MapRegister[server][owner.Map][i]; !ok {
		pet.PseudoID = int(i)
		MapRegister[server][owner.Map][i] = pet
		return
	}
}
func GenerateIDForBabyPet(bby *BabyPet) {
	mrMutex.Lock()
	defer mrMutex.Unlock()

	for i := uint16(3000); i <= 3500; i++ {
		if _, ok := MapRegister[bby.Server][bby.Mapid][i]; !ok {
			bby.PseudoID = int(i)
			MapRegister[bby.Server][bby.Mapid][i] = bby
			return
		}
	}
}

func GenerateIDForNPC(NPCPos *NpcPosition) {
	mrMutex.Lock()
	defer mrMutex.Unlock()
	c := 1
	//for c := 1; c <= database.SERVER_COUNT; c++ {
	for {
		i := uint16(utils.RandInt(20000, 30000))
		if _, ok := MapRegister[c][NPCPos.MapID][i]; !ok {
			//NPCPos.PseudoID = uint16(NPCPos.NPCID)
			NPCPos.PseudoID = i
			MapRegister[c][NPCPos.MapID][NPCPos.PseudoID] = NPCPos
			break
		}
	}
	//}
}
func GenerateIDForHouse(HouseItem *HousingItem) {
	mrMutex.Lock()
	defer mrMutex.Unlock()
	c := 1
	for {
		i := uint16(utils.RandInt(7000, 8000))
		if _, ok := MapRegister[c][int16(HouseItem.MapID)][i]; !ok {
			HouseItem.PseudoID = i
			MapRegister[c][int16(HouseItem.MapID)][HouseItem.PseudoID] = HouseItem
			break
		}
	}

}
func FindCharacter(server int, ID uint16) *Character {
	prMutex.RLock()
	defer prMutex.RUnlock()
	if c, ok := PlayerRegister[ID].(*Character); ok {
		return c
	}
	return nil
}
