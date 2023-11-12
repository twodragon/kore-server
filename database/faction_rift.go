package database

import (
	"log"
	"math"
	"time"

	"github.com/twodragon/kore-server/nats"

	"github.com/twodragon/kore-server/utils"

	"github.com/thoas/go-funk"
)

var (
	ZhuangMovingPointsLeft = map[int][]float64{1: {131, 449}, 2: {191, 441}, 3: {237, 413}, 4: {255, 427}, 5: {339, 395}, 6: {389, 437}, 7: {423, 419}, 8: {413, 351}, 9: {391, 325}, 10: {417, 291}, 11: {417, 245}, 12: {403, 225}, 13: {409, 205}, 14: {443, 169}, 15: {447, 121}}
	ShaogMovingPointsLeft  = map[int][]float64{1: {393, 67}, 2: {349, 67}, 3: {331, 83}, 4: {311, 75}, 5: {285, 81}, 6: {249, 67}, 7: {211, 75}, 8: {175, 111}, 9: {139, 87}, 10: {115, 93}, 11: {115, 115}, 12: {131, 135}, 13: {99, 167}, 14: {95, 201}, 15: {111, 233}, 16: {111, 291}, 17: {99, 307}, 18: {67, 303}, 19: {63, 369}}

	RiftCreeps = []int{223820, 223821, 223822, 223823, 223824, 223825, 223826, 223827, 223828, 223829}
)

func (ai *AI) AIRiftHandler() {

	npcPos := GetNPCPosByID(ai.PosID)
	if npcPos == nil {
		return
	}
	npc, ok := GetNpcInfo(npcPos.NPCID)
	if !ok || npc == nil {
		return
	}

	if len(ai.OnSightPlayers) == 0 && ai.HP > 0 {
		ai.HP = npc.MaxHp
	}

	if len(ai.OnSightPlayers) > 0 && ai.HP > 0 {

		ai.PlayersMutex.RLock()
		ids := funk.Keys(ai.OnSightPlayers).([]int)
		ai.PlayersMutex.RUnlock()

		for _, id := range ids {
			remove := false

			c, err := FindCharacterByID(id)
			if err != nil || c == nil || !c.IsOnline || c.Map != ai.Map {
				remove = true
			}

			if c != nil {
				user, err := FindUserByID(c.UserID)
				if err != nil || user == nil || user.ConnectedIP == "" || user.ConnectedServer == 0 || user.ConnectedServer != ai.Server {
					remove = true
				}
			}
			if funk.Contains(WarStonesIDs, npc.ID) {
				coordinate := ConvertPointToLocation(c.Coordinate)
				aiCoordinate := ConvertPointToLocation(ai.Coordinate)
				distance := utils.CalculateDistance(coordinate, aiCoordinate)
				if distance < 20 {
					resp := STONE_APPEARED
					resp.Insert(utils.IntToBytes(uint64(ai.PseudoID), 2, true), 6) // mob pseudo id
					resp.Insert(utils.IntToBytes(uint64(npc.ID), 4, true), 8)      // mob npc id
					resp.Insert(utils.IntToBytes(uint64(npc.Level), 4, true), 12)  // mob level
					resp.Insert(utils.IntToBytes(uint64(ai.HP), 8, true), 33)      // mob hp
					resp.Insert(utils.IntToBytes(uint64(npc.MaxHp), 8, true), 41)  // mob max hp
					resp.Insert(utils.FloatToBytes(aiCoordinate.X, 4, true), 51)   // coordinate-x
					resp.Insert(utils.FloatToBytes(aiCoordinate.Y, 4, true), 55)   // coordinate-y
					resp.Insert(utils.FloatToBytes(aiCoordinate.X, 4, true), 63)   // coordinate-x
					resp.Insert(utils.FloatToBytes(aiCoordinate.Y, 4, true), 67)   // coordinate-y
					resp.Overwrite(utils.IntToBytes(uint64(WarStones[int(ai.PseudoID)].ConquereValue), 1, false), 37)
					resp.Overwrite([]byte{0xc8}, 45)
					c.Socket.Write(resp)

					if c.Faction == 1 {
						if !funk.Contains(WarStones[int(ai.PseudoID)].NearbyZuhangV, c) {
							WarStones[int(ai.PseudoID)].NearbyZuhangV = append(WarStones[int(ai.PseudoID)].NearbyZuhangV, c.ID)
						}
					} else {
						if !funk.Contains(WarStones[int(ai.PseudoID)].NearbyShaoV, c) {
							WarStones[int(ai.PseudoID)].NearbyShaoV = append(WarStones[int(ai.PseudoID)].NearbyShaoV, c.ID)
						}
					}
					if c.Socket.Stats.HP <= 0 {
						if c.Faction == 1 {
							if funk.Contains(WarStones[int(ai.PseudoID)].NearbyZuhangV, c) {
								WarStones[int(ai.PseudoID)].RemoveZuhang(c.ID)
							}
						} else {
							if funk.Contains(WarStones[int(ai.PseudoID)].NearbyShaoV, c) {
								WarStones[int(ai.PseudoID)].RemoveShao(c.ID)
							}
						}
					}
				} else {
					if c.Faction == 1 {
						if funk.Contains(WarStones[int(ai.PseudoID)].NearbyZuhangV, c) {
							WarStones[int(ai.PseudoID)].RemoveZuhang(c.ID)

						}
					} else {
						if funk.Contains(WarStones[int(ai.PseudoID)].NearbyShaoV, c) {
							WarStones[int(ai.PseudoID)].RemoveShao(c.ID)
						}
					}
				}
			}

			if remove {
				ai.PlayersMutex.Lock()
				delete(ai.OnSightPlayers, id)
				ai.PlayersMutex.Unlock()
				if funk.Contains(WarStonesIDs, npc.ID) {
					if c.Faction == 1 {
						if funk.Contains(WarStones[int(ai.PseudoID)].NearbyZuhangV, c) {
							WarStones[int(ai.PseudoID)].RemoveZuhang(c.ID)

						}
					} else {
						if funk.Contains(WarStones[int(ai.PseudoID)].NearbyShaoV, c) {
							WarStones[int(ai.PseudoID)].RemoveShao(c.ID)
						}
					}
				}
			}
		}

		if ai.TargetPetID > 0 {
			pet, ok := GetFromRegister(ai.Server, ai.Map, uint16(ai.TargetPetID)).(*PetSlot)
			if !ok || pet == nil || !pet.IsOnline || pet.HP <= 0 {
				ai.TargetPetID = 0
			}
		}

		if ai.TargetPlayerID > 0 {
			c, err := FindCharacterByID(ai.TargetPlayerID)
			if err != nil || c == nil || !c.IsOnline || c.Socket == nil || c.Socket.Stats.HP <= 0 {
				ai.TargetPlayerID = 0
				//ai.HP = npc.MaxHp
			} else {
				slots, _ := c.InventorySlots()
				petSlot := slots[0x0A]
				pet := petSlot.Pet
				petInfo, ok := Pets[petSlot.ItemID]
				if pet != nil && ok && pet.IsOnline && !petInfo.Combat {
					ai.TargetPlayerID = 0
					ai.TargetPetID = petSlot.Pet.PseudoID
				}
			}
		}
		if ai.TargetAiID > 0 {
			target := AIs[ai.TargetAiID]
			if target == nil || target.IsDead {
				ai.TargetAiID = 0
			}
		}

		var err error
		ai.TargetAiID = 0
		if ai.TargetPetID == 0 && ai.TargetPlayerID == 0 && ai.TargetAiID == 0 { // gotta find a target

			ai.TargetPlayerID, err = ai.FindTargetCharacterID() // 50% chance to trigger
			if err != nil {
				log.Println("AIHandler FindTargetPlayer error:", err)
			}

			petSlot, err := ai.FindTargetPetID(ai.TargetPlayerID)
			if err != nil {
				log.Println("AIHandler FindTargetPet error:", err)
			}
			/*ai.TargetAiID, err = ai.FindTargetAI()
			if err != nil {
				log.Println("AIHandler FindTargetAI error:", err)
			}
			*/
			ai.TargetAiID = 0

			if petSlot != nil {
				pet := petSlot.Pet
				character, _ := FindCharacterByID(ai.TargetPlayerID)
				if pet != nil && ai.TargetPlayerID > 0 && character.IsMounting {
					ai.TargetPlayerID = 0
					ai.TargetPetID = pet.PseudoID
				}
				seed := utils.RandInt(0, 1000)
				if pet != nil && seed > 420 {
					ai.TargetPlayerID = 0
					ai.TargetPetID = pet.PseudoID
				}
			}
		}

		if ai.TargetPlayerID > 0 || ai.TargetPetID > 0 || ai.TargetAiID > 0 {
			ai.IsMoving = false
		}

		if ai.IsMoving {
			goto OUT
		}

		if ai.TargetPlayerID == 0 && ai.TargetPetID == 0 && ai.TargetAiID == 0 { // Idle mode
			ai.MoveToNextPoint()

		} else if ai.TargetPetID > 0 {
			pet, ok := GetFromRegister(ai.Server, ai.Map, uint16(ai.TargetPetID)).(*PetSlot)
			if !ok || pet == nil || !pet.IsOnline || pet.HP <= 0 {
				ai.TargetPetID = 0
				goto OUT
			}

			aiCoordinate := ConvertPointToLocation(ai.Coordinate)
			distance := utils.CalculateDistance(&pet.Coordinate, aiCoordinate)

			if ai.ShouldGoBack() || distance > 100 { // better to retreat
				ai.TargetPlayerID = 0
				ai.MovementToken = 0
				aiMinCoordinate := ConvertPointToLocation(npcPos.MinLocation)
				go ai.MovementHandler(ai.MovementToken, aiCoordinate, aiMinCoordinate, ai.RunningSpeed)

			} else if distance <= 4 && pet.IsOnline && pet.HP > 0 { // attack
				seed := utils.RandInt(1, 1000)
				skillIds := npc.SkillIds
				skillsCount := len(skillIds) - 1
				randomSkill := utils.RandInt(0, int64(skillsCount))
				_, ok := SkillInfos[skillIds[randomSkill]]

				r := utils.Packet{}
				if seed < 400 && ok {
					r.Concat(ai.CastSkillToPet())
				} else {
					r.Concat(ai.AttackPet())
				}

				p := nats.CastPacket{CastNear: true, MobID: ai.ID, Data: r, Type: nats.MOB_ATTACK}
				p.Cast()

			} else if distance > 3 && distance <= 100 { // chase
				ai.IsMoving = true
				target := GeneratePoint(&pet.Coordinate)
				ai.TargetLocation = target

				token := ai.MovementToken
				for token == ai.MovementToken {
					ai.MovementToken = utils.RandInt(1, math.MaxInt64)
				}

				go ai.MovementHandler(ai.MovementToken, aiCoordinate, &target, ai.RunningSpeed)

			}

		} else if ai.TargetPlayerID > 0 { // Target mode player
			character, err := FindCharacterByID(ai.TargetPlayerID)
			if err != nil || character == nil || (character != nil && (!character.IsOnline || character.Invisible)) || character.IsMounting {
				ai.HP = npc.MaxHp
				ai.TargetPlayerID = 0
				goto OUT
			}
			stat := character.Socket.Stats

			characterCoordinate := ConvertPointToLocation(character.Coordinate)
			aiCoordinate := ConvertPointToLocation(ai.Coordinate)
			distance := utils.CalculateDistance(characterCoordinate, aiCoordinate)

			if ai.ShouldGoBack() || distance > 100 { // better to retreat
				ai.TargetPlayerID = 0
				ai.MovementToken = 0
				aiMinCoordinate := ConvertPointToLocation(npcPos.MinLocation)
				go ai.MovementHandler(ai.MovementToken, aiCoordinate, aiMinCoordinate, ai.RunningSpeed)
				//ai.HP = npc.MaxHp
			} else if distance <= 4 && distance > 1 && character.IsActive && stat.HP > 0 { // attack
				seed := utils.RandInt(1, 1000)
				skillIds := npc.SkillIds
				skillsCount := len(skillIds) - 1
				randomSkill := utils.RandInt(0, int64(skillsCount))
				_, ok := SkillInfos[skillIds[randomSkill]]

				r := utils.Packet{}
				if seed < 400 && ok {
					r.Concat(ai.CastSkill())
				} else {
					r.Concat(ai.Attack())
				}

				p := nats.CastPacket{CastNear: true, MobID: ai.ID, Data: r, Type: nats.MOB_ATTACK}
				p.Cast()

			} else if (distance > 4 && distance <= 100) || distance <= 1 { // chase
				ai.IsMoving = true
				target := GeneratePoint(characterCoordinate)
				rand := utils.RandFloat(0, 360)
				target.X += 3 * math.Cos(rand)
				target.Y += 3 * math.Sin(rand)
				ai.TargetLocation = target

				token := ai.MovementToken
				for token == ai.MovementToken {
					ai.MovementToken = utils.RandInt(1, math.MaxInt64)
				}

				go ai.MovementHandler(ai.MovementToken, aiCoordinate, &target, ai.RunningSpeed)

			}
		} else if ai.TargetAiID > 0 {
			target := AIs[ai.TargetAiID]
			targetCoordinate := ConvertPointToLocation(target.Coordinate)
			aiCoordinate := ConvertPointToLocation(ai.Coordinate)
			distance := utils.CalculateDistance(targetCoordinate, aiCoordinate)

			if ai.ShouldGoBack() || distance > 100 { // better to retreat
				ai.TargetAiID = 0
				ai.MovementToken = 0
				aiMinCoordinate := ConvertPointToLocation(npcPos.MinLocation)
				go ai.MovementHandler(ai.MovementToken, aiCoordinate, aiMinCoordinate, ai.RunningSpeed)
			} else if distance <= 5 && !target.IsDead { // attack
				seed := utils.RandInt(1, 1000)
				skillIds := npc.SkillIds
				skillsCount := len(skillIds) - 1
				randomSkill := utils.RandInt(0, int64(skillsCount))
				_, ok := SkillInfos[skillIds[randomSkill]]
				r := utils.Packet{}

				if seed < 400 && ok {
					attack := ai.CastSkillToAI()
					r.Concat(attack)
				} else {
					attack := ai.AttackAI()
					r.Concat(attack)
				}

				p := nats.CastPacket{CastNear: true, MobID: ai.ID, Data: r, Type: nats.MOB_ATTACK}
				p.Cast()
			} else if distance > 5 && distance <= 100 && !target.IsDead { // chase
				ai.IsMoving = true
				target := GeneratePoint(targetCoordinate)
				ai.TargetLocation = target

				token := ai.MovementToken
				for token == ai.MovementToken {
					ai.MovementToken = utils.RandInt(1, math.MaxInt64)
				}

				go ai.MovementHandler(ai.MovementToken, aiCoordinate, &target, ai.RunningSpeed)

			}
		}
	}

OUT:
	delay := utils.RandFloat(1.0, 1.5) * 1000
	time.AfterFunc(time.Duration(delay)*time.Millisecond, func() {
		ai.AIHandler()
	})
}
func (ai *AI) MoveToNextPoint() {
	coordinate := ConvertPointToLocation(ai.Coordinate)

	target := utils.Location{X: ZhuangMovingPointsLeft[2][1], Y: ZhuangMovingPointsLeft[2][2]}
	ai.TargetLocation = target

	token := ai.MovementToken
	for token == ai.MovementToken {
		ai.MovementToken = utils.RandInt(1, math.MaxInt64)
	}

	go ai.MovementHandler(ai.MovementToken, coordinate, &target, ai.WalkingSpeed)

}
func SpawnCreep(faction int, npcPos *NpcPosition) {
	npc, ok := GetNpcInfo(npcPos.NPCID)
	if !ok {
		return
	}

	ai := &AI{ID: len(AIs), HP: npc.MaxHp, Map: 251, PosID: npcPos.ID, RunningSpeed: 10, Server: 1, WalkingSpeed: 5, Once: true}

	ai.OnSightPlayers = make(map[int]interface{})

	loc := utils.Location{X: 1, Y: 1}
	if faction == 1 {
		loc = utils.Location{X: ZhuangMovingPointsLeft[1][1], Y: ZhuangMovingPointsLeft[1][2]}
	} else if faction == 2 {
		loc = utils.Location{X: ShaogMovingPointsLeft[1][0], Y: ShaogMovingPointsLeft[1][1]}
	}

	ai.Coordinate = loc.String()
	ai.Handler = ai.AIRiftHandler
	go ai.Handler()

	AIsByMap[ai.Server][251] = append(AIsByMap[ai.Server][251], ai)
	AIs[ai.ID] = ai
}
