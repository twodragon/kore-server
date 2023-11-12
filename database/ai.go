package database

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/twodragon/kore-server/messaging"
	"github.com/twodragon/kore-server/nats"

	"github.com/twodragon/kore-server/utils"

	"github.com/thoas/go-funk"
)

type Damage struct {
	DealerID int
	Damage   int
}

type AI struct {
	ID         int    `db:"id" json:"id"`
	PosID      int    `db:"pos_id" json:"pos_id"`
	Server     int    `db:"server" json:"server"`
	Faction    int    `db:"faction" json:"faction"`
	Map        int16  `db:"map" json:"map"`
	Coordinate string `db:"spawning_point" json:"spawning_point"`
	CanAttack  bool   `db:"canattack" json:"canattack"`

	IsDead bool `db:"is_dead" json:"is_dead"`

	//Coordinate     string              `db:"-"`
	DamageDealers  utils.SMap          `db:"-"`
	TargetLocation utils.Location      `db:"-"`
	PseudoID       uint16              `db:"-"`
	CHI            int                 `db:"-" json:"chi"`
	HP             int                 `db:"-" json:"hp"`
	IsMoving       bool                `db:"-" json:"is_moving"`
	MovementToken  int64               `db:"-" json:"-"`
	OnSightPlayers map[int]interface{} `db:"-" json:"players"`
	PlayersMutex   sync.RWMutex        `db:"-"`
	TargetPlayerID int                 `db:"-" json:"target_player"`
	TargetPetID    int                 `db:"-" json:"target_pet"`
	TargetAiID     int                 `db:"-" json:"target_ai"`
	Handler        func()              `db:"-" json:"-"`
	Once           bool                `db:"-"`
	Poisoned       int                 `db:"-"`
	Paralisied     int                 `db:"-"`
	Confusioned    int                 `db:"-"`
	FiveClanGuild  int                 `db:"-"`

	WalkingSpeed float64      `db:"-"`
	RunningSpeed float64      `db:"-"`
	NPCpos       *NpcPosition `db:"-"`

	IsMating       bool `db:"-"`
	MatingPartner  *AI  `db:"-"`
	MatingCooldown int  `db:"-"`
}

type DungeonMobsCounter struct {
	BlackBandits int
	Rogues       int
	Ghosts       int
	Animals      int
}

type SeasonCaveMobs struct {
	Bats      int
	Spiders   int
	Snakes    int
	Centipede int
	Demon     int
}

var (
	AIs             = make(map[int]*AI)
	DungeonsAiByMap []map[int16][]*AI
	DungeonsTest    = make(map[int]*AI)

	YingYangMobsCounter   = make(map[int16]*DungeonMobsCounter)
	SeasonCaveMobsCounter = make(map[int]*SeasonCaveMobs)

	AIsByMap          []map[int16][]*AI
	BabyPetsByMap     []map[int16][]*BabyPet
	HousingItemsByMap []map[int16][]*HousingItem

	DungeonsByMap []map[int16]int

	eventBosses        = []int{50009, 50010}
	EventItems         = []int{}
	EventProb          = 20
	PermanentItems     = []int{13370000}
	PermanentItemsProb = 1
	SocketBox          = []int{99009121}
	SocketBoxRate      = 1

	STONE_APPEARED = utils.Packet{0xAA, 0x55, 0x57, 0x00, 0x31, 0x01, 0x01, 0x00, 0x00, 0x00, 0x0c, 0x45, 0x6d, 0x70, 0x69, 0x72, 0x65, 0x20, 0x53, 0x74, 0x6f, 0x6e, 0x65, 0x01, 0x01,
		0x8E, 0xE5, 0x38, 0xC0, 0xD9, 0xB8, 0x05, 0xC0, 0x00, 0x00, 0x00, 0x40, 0xFF, 0xFF, 0x00, 0xFC, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x64, 0x00, 0x55, 0xAA}
	MOB_MOVEMENT    = utils.Packet{0xAA, 0x55, 0x21, 0x00, 0x33, 0x00, 0xBC, 0xDB, 0x9F, 0x41, 0x52, 0x70, 0xA2, 0x41, 0x00, 0x55, 0xAA}
	MOB_ATTACK      = utils.Packet{0xAA, 0x55, 0x0C, 0x00, 0x41, 0x01, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x55, 0xAA}
	MOB_SKILL       = utils.Packet{0xAA, 0x55, 0x1B, 0x00, 0x42, 0x0A, 0x00, 0xDF, 0x28, 0xFA, 0xBE, 0x01, 0x01, 0x55, 0xAA}
	MOB_DEAL_DAMAGE = utils.Packet{0xAA, 0x55, 0x28, 0x00, 0x16, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x27, 0x00, 0x00, 0x55, 0xAA}
	ITEM_DROPPED = utils.Packet{0xAA, 0x55, 0x42, 0x00, 0x67, 0x02, 0x01, 0x01, 0x7A, 0xFB, 0x7B, 0xBF, 0x00, 0xA2, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x55, 0xAA}

	MOB_APPEARED = utils.Packet{0xAA, 0x55, 0x54, 0x00, 0x31, 0x01, 0xFF, 0xFF, 0xFF, 0xFF, 0x01, 0x01,
		0x8E, 0xE5, 0x38, 0xC0, 0xD9, 0xB8, 0x05, 0xC0, 0x00, 0x00, 0x00, 0x40, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x64, 0x00, 0x55, 0xAA}

	PET_APPEARED = utils.Packet{0xAA, 0x55, 0x54, 0x00, 0x31, 0x01, 0xF0, 0xFF, 0xFF, 0xFF, 0x01, 0x01,
		0x8E, 0xE5, 0x38, 0xC0, 0xD9, 0xB8, 0x05, 0xC0, 0x00, 0x00, 0x00, 0x40, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x64, 0x00, 0x55, 0xAA}

	DROP_DISAPPEARED = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x67, 0x04, 0x55, 0xAA}

	/*dropOffsets = []*utils.Location{&utils.Location{0, 0}, &utils.Location{0, 1}, &utils.Location{1, 0}, &utils.Location{1, 1}, &utils.Location{-1, 0},
	&utils.Location{-1, 1}, &utils.Location{-1, -1}, &utils.Location{0, -1}, &utils.Location{1, -1}, &utils.Location{-1, 2}, &utils.Location{0, 2},
	&utils.Location{2, 2}, &utils.Location{2, 1}, &utils.Location{2, 0}, &utils.Location{2, -1}, &utils.Location{2, -2}, &utils.Location{1, -2},
	&utils.Location{0, -2}, &utils.Location{-1, -2}, &utils.Location{-2, -2}, &utils.Location{-2, -1}, &utils.Location{-2, 0}, &utils.Location{-2, 1},
	&utils.Location{-2, 2}, &utils.Location{-2, 3}, &utils.Location{-1, 3}, &utils.Location{0, 3}, &utils.Location{1, 3}, &utils.Location{2, 3},
	&utils.Location{3, 3}, &utils.Location{3, 2}, &utils.Location{3, 1}, &utils.Location{3, 0}, &utils.Location{3, -1}, &utils.Location{3, -2},
	&utils.Location{3, -3}, &utils.Location{2, -3}, &utils.Location{1, -3}, &utils.Location{0, -3}, &utils.Location{-1, -3}, &utils.Location{-2, -3},
	&utils.Location{-3, -3}, &utils.Location{-3, -2}, &utils.Location{-3, -1}, &utils.Location{-3, 0}, &utils.Location{-3, 1}, &utils.Location{-3, 2}, &utils.Location{-3, 3}}*/
	dropOffsets = []utils.Location{{X: 0, Y: 0}, {X: 0, Y: 1},
		{X: 1, Y: 0}, {X: 1, Y: 1}, {X: -1, Y: 0}, {X: -1, Y: 1}, {X: -1, Y: -1},
		{X: 0, Y: -1}, {X: 1, Y: -1}, {X: -1, Y: 2}, {X: 0, Y: 2}, {X: 2, Y: 2},
		{X: 2, Y: 1}, {X: 2, Y: 0}, {X: 2, Y: -1}, {X: 2, Y: -2}, {X: 1, Y: -2}, {X: 0, Y: -2},
		{X: -1, Y: -2}, {X: -2, Y: -2}, {X: -2, Y: -1}, {X: -2, Y: 0}, {X: -2, Y: 1},
		{X: -2, Y: 2}, {X: -2, Y: 3}, {X: -1, Y: 3}, {X: 0, Y: 3}, {X: 1, Y: 3}, {X: 2, Y: 3},
		{X: 3, Y: 3}, {X: 3, Y: 2}, {X: 3, Y: 1}, {X: 3, Y: 0}, {X: 3, Y: -1}, {X: 3, Y: -2},
		{X: 3, Y: -3}, {X: 2, Y: -3}, {X: 1, Y: -3}, {X: 0, Y: -3}, {X: -1, Y: -3},
		{X: -2, Y: -3}, {X: -3, Y: -3}, {X: -3, Y: -2}, {X: -3, Y: -1}, {X: -3, Y: 0},
		{X: -3, Y: 1}, {X: -3, Y: 2}, {X: -3, Y: 3}}
)

func FindAIByID(ID int) *AI {
	return AIs[ID]
}

func (ai *AI) SetCoordinate(coordinate *utils.Location) {
	ai.Coordinate = fmt.Sprintf("(%.1f,%.1f)", coordinate.X, coordinate.Y)
}

func (ai *AI) Create() error {
	return pgsql_DbMap.Insert(ai)
}

func (ai *AI) Update() error {

	_, err := pgsql_DbMap.Update(ai)
	if err != nil {
		log.Println(err)
	}
	return err
}

func GetAllAI() error {
	var arr []*AI
	query := `select * from data.ai`

	if _, err := pgsql_DbMap.Select(&arr, query); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("GetAllAI: %s", err.Error())
	}

	for _, a := range arr {
		AIs[a.ID] = a
	}

	return nil
}

func (ai *AI) FindTargetCharacterID() (int, error) {

	var distance = 15.0

	if len(characters) == 0 {
		return 0, nil
	}
	if ai.Faction == 3 {
		return 0, nil
	}

	aiCoordinate := ConvertPointToLocation(ai.Coordinate)

	characterMutex.RLock()
	allChars := funk.Values(characters)
	characterMutex.RUnlock()
	filtered := funk.Filter(allChars, func(c *Character) bool {

		if c.Socket == nil || !c.IsOnline {
			return false
		}
		if c.Faction == ai.Faction {
			return false
		}

		user := c.Socket.User
		stat := c.Socket.Stats

		if user == nil || stat == nil {
			return false
		}

		characterCoordinate := ConvertPointToLocation(c.Coordinate)

		seed := utils.RandInt(0, 1000)

		return user.ConnectedServer == ai.Server && c.Map == ai.Map && stat.HP > 0 && !c.Invisible &&
			//characterCoordinate.X >= minCoordinate.X && characterCoordinate.X <= maxCoordinate.X &&
			//characterCoordinate.Y >= minCoordinate.Y && characterCoordinate.Y <= maxCoordinate.Y &&
			c.GetNumberOfAiAroundPlayer() < 7 &&
			utils.CalculateDistance(characterCoordinate, aiCoordinate) <= distance && seed < 500
	})

	filtered = funk.Shuffle(filtered)
	characters := filtered.([]*Character)
	if len(characters) > 0 {
		return characters[0].ID, nil
	}

	return 0, nil
}

func (ai *AI) AIHandler() {

	npcPos := GetNPCPosByID(ai.PosID)
	if npcPos == nil {
		return
	}
	npc, ok := GetNpcInfo(npcPos.NPCID)
	if !ok || npc == nil {
		return
	}
	if ai.IsDead {
		ai.HP = 0
		goto OUT
	}

	if len(ai.OnSightPlayers) == 0 && ai.HP > 0 && ai.HP != npc.MaxHp {
		ai.HP = npc.MaxHp
		goto OUT
	}

	if len(ai.OnSightPlayers) > 0 && ai.HP > 0 && ai.CanAttack {
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

			if remove {
				ai.PlayersMutex.Lock()
				delete(ai.OnSightPlayers, id)
				ai.PlayersMutex.Unlock()
			}
		}
		//go ai.HandleAIBuffs()
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
		/*if ai.TargetAiID > 0 {
			target := AIs[ai.TargetAiID]
			if target == nil || target.IsDead {
				ai.TargetAiID = 0
			}
		}*/
		var err error
		ai.TargetAiID = 0
		if ai.TargetPetID == 0 && ai.TargetPlayerID == 0 { // gotta find a target   ai.TargetAiID == 0

			ai.TargetPlayerID, err = ai.FindTargetCharacterID() // 50% chance to trigger
			if err != nil {
				log.Println("AIHandler FindTargetPlayer error:", err)
			}
			if ai.TargetPlayerID != 0 {
				petSlot, err := ai.FindTargetPetID(ai.TargetPlayerID)
				if err != nil {
					log.Println("AIHandler FindTargetPet error:", err)
				}
				/*ai.TargetAiID, err = ai.FindTargetAI()
				if err != nil {
					log.Println("AIHandler FindTargetAI error:", err)
				}
				*/

				if petSlot != nil {
					pet := petSlot.Pet
					//petInfo, ok := Pets[petSlot.ItemID]
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

		}

		if ai.TargetPlayerID > 0 || ai.TargetPetID > 0 {
			ai.IsMoving = false
		}

		if ai.NPCpos.NPCID >= 20180 && ai.NPCpos.NPCID <= 20185 && !ai.IsMating {
			ids, err := ai.GetNearbyDrops(25)
			if err == nil {
				for _, id := range ids {
					drop := GetDrop(ai.Server, ai.Map, uint16(id))
					if drop == nil {
						continue
					} else if drop.Item.ItemID == 13003128 {
						dropCoordinate := ConvertPointToLocation(drop.Location.String())
						coordinate := ConvertPointToLocation(ai.Coordinate)
						if utils.CalculateDistance(coordinate, dropCoordinate) < 5 {
							ai.DoSkillAnimation(30122)
							RemoveDrop(ai.Server, ai.Map, uint16(id))
						} else if ai.TargetLocation != drop.Location {
							ai.TargetLocation = drop.Location

							token := ai.MovementToken
							for token == ai.MovementToken {
								ai.MovementToken = utils.RandInt(1, math.MaxInt64)
							}

							go ai.MovementHandler(ai.MovementToken, coordinate, &drop.Location, ai.WalkingSpeed)
						}
					}
				}
			}
		}

		if ai.NPCpos.NPCID == 50073 && ai.MatingCooldown == 0 {
			rand := utils.RandInt(0, 100)
			if rand < 5 {
				baseLocation := ConvertPointToLocation(ai.Coordinate)
				drop := NewSlot()
				items := []int64{17502868, 17502869, 17502870}
				rand := utils.RandInt(0, 2)
				drop.ItemID = items[rand]
				drop.Quantity = 120
				drop.Plus = 0
				dr := &Drop{Server: ai.Server, Map: ai.Map, Claimer: nil, Item: drop,
					Location: *baseLocation}

				dr.GenerateIDForDrop(ai.Server, ai.Map)

				dropID := uint16(dr.ID)
				ai.MatingCooldown = 1

				go time.AfterFunc(time.Duration(10)*time.Minute, func() { // remove drop after timeout
					RemoveDrop(ai.Server, ai.Map, dropID)
					ai.MatingCooldown = 0
				})
			}
		}

		if ai.IsMoving {
			goto OUT
		}

		if ai.TargetPlayerID == 0 && ai.TargetPetID == 0 && !ai.IsMating { // Idle mode

			coordinate := ConvertPointToLocation(ai.Coordinate)
			minCoordinate := ConvertPointToLocation(npcPos.MinLocation)
			maxCoordinate := ConvertPointToLocation(npcPos.MaxLocation)

			if utils.RandInt(0, 1000) < 750 && minCoordinate != maxCoordinate { // 75% chance to move
				ai.IsMoving = true

				targetX := utils.RandFloat(minCoordinate.X, maxCoordinate.X)
				targetY := utils.RandFloat(minCoordinate.Y, maxCoordinate.Y)
				target := utils.Location{X: targetX, Y: targetY}

				ai.TargetLocation = target

				token := ai.MovementToken
				for token == ai.MovementToken {
					ai.MovementToken = utils.RandInt(1, math.MaxInt64)
				}

				go ai.MovementHandler(ai.MovementToken, coordinate, &target, ai.WalkingSpeed)

			}

		} else if ai.TargetPetID > 0 {
			if ai.IsMating {
				ai.FinishMating()
			}
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
				randomSkill := uint64(utils.RandInt(0, int64(skillsCount)))
				_, ok := SkillInfos[skillIds[randomSkill]]

				r := utils.Packet{}
				if seed < 400 && ok {
					r.Concat(ai.CastSkillToPet())
				} else {
					r.Concat(ai.AttackPet())
				}
				//go pet.AddAttackingMob(ai)
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
			if ai.IsMating {
				ai.FinishMating()
			}
			character, err := FindCharacterByID(ai.TargetPlayerID)
			//character.CheckAttackingMobs()
			if err != nil || character == nil || character.Socket == nil || (character != nil && (!character.IsOnline || character.Invisible)) || character.IsMounting {
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
			} else if distance <= 4 && character.IsActive && stat.HP > 0 { // attack
				//	character.CheckAttackingMobs()
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

				//	go character.AddAttackingMob(ai)
				p := nats.CastPacket{CastNear: true, MobID: ai.ID, Data: r, Type: nats.MOB_ATTACK}
				p.Cast()
				time.Sleep(time.Millisecond * time.Duration(seed))

			} else if distance > 4 && distance <= 100 { // chase
				if err != nil || character == nil || (character != nil && (!character.IsOnline || character.Invisible)) {
					ai.HP = npc.MaxHp
					ai.TargetPlayerID = 0
					goto OUT
				}
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
		} /*else if ai.TargetAiID > 0 {
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
		}*/
	}

OUT:
	delay := utils.RandFloat(1.0, 1.5) * 1500
	time.AfterFunc(time.Duration(delay)*time.Millisecond, func() {
		if ai.Handler != nil {
			ai.AIHandler()
		}
	})
}
func (ai *AI) HandleAIBuffs() {
	buffs, err := FindBuffsByAiPseudoID(ai.PseudoID)
	if err != nil || len(buffs) == 0 {
		return
	}
	now := time.Now()
	secs := now.Unix()
	if buff := buffs[0]; buff.StartedAt+buff.Duration <= secs { // buff expired
		DeleteBuffByAiPseudoID(ai.PseudoID, buff.ID)
		go buff.Delete()
	}
	for _, buff := range buffs {
		if buff.ID == 257 { //POSION
			ai.DamageDealers.Add(buff.CharacterID, &Damage{Damage: buff.HPRecoveryRate, DealerID: buff.CharacterID})
		}
	}
}

func (ai *AI) GetNearbyDrops(radius float64) ([]int, error) {

	var (
		distance = radius
		ids      []int
	)

	allDrops := GetDropsInMap(ai.Server, ai.Map)
	filtered := funk.Filter(allDrops, func(drop *Drop) bool {

		characterCoordinate := ConvertPointToLocation(ai.Coordinate)

		return utils.CalculateDistance(characterCoordinate, &drop.Location) <= distance
	})

	for _, d := range filtered.([]*Drop) {
		ids = append(ids, d.ID)
	}

	return ids, nil
}

func (ai *AI) FinishMating() {
	ai.IsMating = false
	if ai.MatingPartner == nil {
		return
	}
	ai.MatingPartner.IsMating = false
	ai.MatingPartner.MatingPartner = nil
	ai.MatingPartner = nil

}
func (ai *AI) FindMatinPartner(mates []int) *AI {
	distance := 10.0
	aiCoordinate := ConvertPointToLocation(ai.Coordinate)

	allMobs := AIsByMap[1][10]

	filtered := funk.Filter(allMobs, func(target *AI) bool {

		targetCoordinate := ConvertPointToLocation(target.Coordinate)

		if !funk.Contains(mates, target.NPCpos.NPCID) {
			return false
		}

		return utils.CalculateDistance(
			targetCoordinate, aiCoordinate) <= distance &&
			ai.Map == target.Map &&
			ai.Server == target.Server &&
			ai != target &&
			target.HP > 0 &&
			target.TargetPetID == 0 &&
			target.TargetPlayerID == 0 &&
			target.TargetAiID == 0 &&
			!target.IsMating //target.IsMating == false

	})

	filtered = funk.Shuffle(filtered)
	ais := filtered.([]*AI)

	if len(ais) > 0 {

		return ais[0]
	}

	return nil
}
func (ai *AI) RecoveryHandler(npc *NPC) {
	if ai.HP < npc.MaxHp/20 {
		return
	}
	if ai.HP < npc.MaxHp && ai.HP > npc.HpRecovery && ai.HP > int(float32(npc.MaxHp)*0.2) {
		ai.HP += npc.HpRecovery
		if ai.HP > npc.MaxHp {
			ai.HP = npc.MaxHp
		}
	}
	index := 5
	r := DEAL_BUFF_AI
	r.Insert(utils.IntToBytes(uint64(ai.PseudoID), 2, true), index) // ai pseudo id
	index += 2
	r.Insert(utils.IntToBytes(uint64(ai.PseudoID), 2, true), index) // ai pseudo id
	index += 2
	r.Insert(utils.IntToBytes(uint64(ai.HP), 4, true), index) // ai current hp
	index += 4
	r.Insert(utils.IntToBytes(uint64(ai.CHI), 4, true), index) // ai current chi
	r.Insert(utils.IntToBytes(uint64(0), 4, true), 22)         //BUFF ID

	p := nats.CastPacket{CastNear: true, MobID: ai.ID, Data: r, Type: nats.MOB_ATTACK}
	p.Cast()

}
func (ai *AI) FindTargetAI() (int, error) {
	distance := 20.0
	aiCoordinate := ConvertPointToLocation(ai.Coordinate)

	allMobs := AIsByMap[ai.Server][ai.Map]
	allAIs := funk.Values(allMobs)

	filtered := funk.Filter(allAIs, func(target *AI) bool {
		pos := GetNPCPosByID(ai.PosID)
		if pos == nil {
			return false
		}

		targetCoordinate := ConvertPointToLocation(target.Coordinate)

		seed := utils.RandInt(0, 1000)

		return utils.CalculateDistance(targetCoordinate, aiCoordinate) <= distance && seed < 500 && target.Faction != ai.Faction &&
			ai.Map == target.Map && ai.Server == target.Server && target.Faction != 3 && pos.Attackable && ai != target && target.HP > 0

	})

	filtered = funk.Shuffle(filtered)
	ais := filtered.([]*AI)

	if len(ais) > 0 {

		return ais[0].ID, nil
	}

	return 0, nil
}

func (ai *AI) FindTargetPetID(characterID int) (*InventorySlot, error) {

	enemy, err := FindCharacterByID(characterID)
	if err != nil || enemy == nil {
		return nil, err
	}

	slots, err := enemy.InventorySlots()
	if err != nil {
		return nil, err
	}

	pet := slots[0x0A].Pet
	if pet == nil || !pet.IsOnline {
		return nil, nil
	}

	return slots[0x0A], nil
}
func (ai *AI) Move(targetLocation utils.Location, runningMode byte) []byte {

	resp := MOB_MOVEMENT
	currentLocation := ConvertPointToLocation(ai.Coordinate)

	resp.Insert(utils.IntToBytes(uint64(ai.PseudoID), 2, true), 5) // mob pseudo id
	resp[7] = runningMode
	resp.Insert(utils.FloatToBytes(currentLocation.X, 4, true), 8)  // current coordinate-x
	resp.Insert(utils.FloatToBytes(currentLocation.Y, 4, true), 12) // current coordinate-y
	resp.Insert(utils.FloatToBytes(targetLocation.X, 4, true), 20)  // current coordinate-x
	resp.Insert(utils.FloatToBytes(targetLocation.Y, 4, true), 24)  // current coordinate-y

	speeds := []float64{0, ai.WalkingSpeed, ai.RunningSpeed}
	resp.Insert(utils.FloatToBytes(speeds[runningMode], 4, true), 32) // current coordinate-y

	return resp
}

func (ai *AI) Attack() []byte {

	resp := MOB_ATTACK
	character, err := FindCharacterByID(ai.TargetPlayerID)
	if err != nil || character == nil {
		return nil
	}

	pos := GetNPCPosByID(ai.PosID)
	if pos == nil {
		return nil
	}

	npc, ok := GetNpcInfo(pos.NPCID)
	if !ok || npc == nil {
		return nil
	}
	if ai.Faction == character.Faction {
		ai.TargetPlayerID = 0
		return nil
	}

	stat := character.Socket.Stats

	rawDamage := int(utils.RandInt(int64(npc.MinATK), int64(npc.MaxATK)))
	if character.Level < 101 {
		rawDamage += int(float32(rawDamage) * 0.05 * float32(character.Reborns))
	}
	damage := int(math.Max(float64(rawDamage-stat.DEF), 3))
	damage = int(float32(damage) - float32(damage)*stat.DEXDamageReduction/100)

	expo := int(npc.Level) - character.Level
	damage += 3 * expo / 5

	reqAcc := (float64(stat.Dodge) * 0.5) - float64(character.Level-int(npc.Level))*10
	if reqAcc < 0 {
		reqAcc = 0
	}

	if ai.Map == 233 {
		if damage < 500 {
			damage = 500
		}
	}
	if character.Level > 200 {
		if utils.RandInt(0, 3500) < int64(reqAcc) {
			damage = 0
		}
	} else if utils.RandInt(0, 2000) < int64(reqAcc) {
		damage = 0
	}
	if damage < 0 {
		damage = 0
	}
	resp.Insert(utils.IntToBytes(uint64(ai.PseudoID), 2, true), 6)        // mob pseudo id
	resp.Insert(utils.IntToBytes(uint64(character.PseudoID), 2, true), 8) // character pseudo id

	resp[11] = 2
	if damage > 0 {
		resp[12] = 1 // damage sound
	}

	resp.Concat(ai.DealDamage(damage))

	return resp
}

func (ai *AI) DoSkillAnimation(skillId uint64) {

	pos := GetNPCPosByID(ai.PosID)
	if pos == nil {
		return
	}

	npc, ok := GetNpcInfo(pos.NPCID)
	if !ok || npc == nil {
		return
	}

	mC := ConvertPointToLocation(ai.Coordinate)

	resp := MOB_SKILL
	resp.Insert(utils.IntToBytes(uint64(ai.PseudoID), 2, true), 7)  // mob pseudo id
	resp.Insert(utils.IntToBytes(skillId, 4, true), 9)              // pet skill id
	resp.Insert(utils.FloatToBytes(mC.X, 4, true), 13)              // pet-x
	resp.Insert(utils.FloatToBytes(mC.Y, 4, true), 17)              // pet-x
	resp.Insert(utils.IntToBytes(uint64(ai.PseudoID), 2, true), 25) // target pseudo id
	resp.Insert(utils.IntToBytes(uint64(ai.PseudoID), 2, true), 28) // target pseudo id

	p := nats.CastPacket{CastNear: true, MobID: ai.ID, Data: resp, Type: nats.MOB_ATTACK}
	p.Cast()
}

func (ai *AI) CastSkill() []byte {

	pos := GetNPCPosByID(ai.PosID)
	if pos == nil {
		return nil
	}

	npc, ok := GetNpcInfo(pos.NPCID)
	if !ok || npc == nil {
		return nil
	}
	character, err := FindCharacterByID(ai.TargetPlayerID)
	if err != nil || character == nil {
		return nil
	}

	if ai.Faction == character.Faction {
		ai.TargetPlayerID = 0
		return nil
	}
	stat := character.Socket.Stats

	rawDamage := int(utils.RandInt(int64(npc.MinArtsATK), int64(npc.MaxArtsATK)))
	if character.Level < 101 {
		rawDamage += int(float32(rawDamage) * 0.05 * float32(character.Reborns))
	}
	damage := int(math.Max(float64(rawDamage-stat.ArtsDEF), 3))
	if character.Level < int(npc.Level) {
		expo := int(npc.Level) - character.Level
		damage += 3 * expo
	}

	reqAcc := (float64(stat.Dodge) * 0.5) - float64(character.Level-int(npc.Level))*10
	if reqAcc < 0 {
		reqAcc = 0
	}
	if character.Level > 200 {
		if utils.RandInt(0, 3500) < int64(reqAcc) {
			damage = 0
		}
	} else if utils.RandInt(0, 2000) < int64(reqAcc) {
		damage = 0
	}
	if damage < 0 {
		damage = 0
	}
	mC := ConvertPointToLocation(ai.Coordinate)
	skillIds := npc.SkillIds
	skillsCount := len(skillIds) - 1
	randomSkill := utils.RandInt(0, int64(skillsCount))
	resp := MOB_SKILL
	resp.Insert(utils.IntToBytes(uint64(ai.PseudoID), 2, true), 7)           // mob pseudo id
	resp.Insert(utils.IntToBytes(uint64(skillIds[randomSkill]), 4, true), 9) // pet skill id
	resp.Insert(utils.FloatToBytes(mC.X, 4, true), 13)                       // pet-x
	resp.Insert(utils.FloatToBytes(mC.Y, 4, true), 17)                       // pet-x
	resp.Insert(utils.IntToBytes(uint64(character.PseudoID), 2, true), 25)   // target pseudo id
	resp.Insert(utils.IntToBytes(uint64(character.PseudoID), 2, true), 28)   // target pseudo id

	resp.Concat(ai.DealDamage(damage))

	return resp

}

func (ai *AI) AttackPet() []byte {

	resp := MOB_ATTACK
	pet, ok := GetFromRegister(ai.Server, ai.Map, uint16(ai.TargetPetID)).(*PetSlot)
	if !ok || pet == nil || !pet.IsOnline || pet.HP <= 0 {
		return nil
	}

	npc, ok := GetNpcInfo(ai.NPCpos.NPCID)
	if !ok || npc == nil {
		return nil
	}

	if pet.Target == 0 {
		pet.Target = int(ai.PseudoID)
	}

	rawDamage := int(utils.RandInt(int64(npc.MinATK), int64(npc.MaxATK)))
	damage := int(math.Max(float64(rawDamage-pet.DEF), 3))

	if damage < 0 {
		damage = 0
	}

	reqAcc := float64(int(pet.Level)-int(npc.Level)) * 10
	if utils.RandInt(0, 1000) < int64(reqAcc) {
		damage = 0
	}

	resp.Insert(utils.IntToBytes(uint64(ai.PseudoID), 2, true), 6) // mob pseudo id
	//resp.Insert([]byte{0}, 8)
	resp.Insert(utils.IntToBytes(uint64(pet.PseudoID), 2, true), 8) // character pseudo id

	resp[11] = 2
	if damage > 0 {
		resp[12] = 1 // damage sound
	}

	resp.Concat(ai.DealDamageToPet(damage))
	return resp
}

func (ai *AI) AttackAI() []byte {

	resp := MOB_ATTACK

	targetAI := AIs[ai.TargetAiID]

	pos := GetNPCPosByID(ai.PosID)
	if pos == nil {
		return nil
	}
	targetPos := GetNPCPosByID(targetAI.PosID)
	if targetPos == nil {
		return nil
	}

	npc, ok := GetNpcInfo(ai.NPCpos.NPCID)
	if !ok || npc == nil {
		return nil
	}

	targetNpc, ok := GetNpcInfo(targetPos.NPCID)
	if !ok || targetNpc == nil {
		return nil
	}

	rawDamage := int(utils.RandInt(int64(npc.MinATK), int64(npc.MaxATK)))
	damage := int(math.Max(float64(rawDamage), 3))
	if damage < 0 {
		damage = 0
	}
	resp.Insert(utils.IntToBytes(uint64(ai.PseudoID), 2, true), 6) // mob pseudo id

	resp.Insert(utils.IntToBytes(uint64(targetAI.PseudoID), 2, true), 8) // character pseudo id

	resp[11] = 2
	if damage > 0 {
		resp[12] = 1 // damage sound
	}

	resp.Concat(ai.DealDamageToAI(damage))

	return resp

}
func (ai *AI) CastSkillToAI() []byte {

	targetAi := AIs[ai.TargetAiID]

	pos := GetNPCPosByID(ai.PosID)
	if pos == nil {
		return nil
	}
	targetPos := GetNPCPosByID(targetAi.PosID)
	if targetPos == nil {
		return nil
	}

	npc, ok := GetNpcInfo(ai.NPCpos.NPCID)
	if !ok || npc == nil {
		return nil
	}

	targetNpc, ok := GetNpcInfo(targetPos.NPCID)
	if !ok || targetNpc == nil {
		return nil
	}

	rawDamage := int(utils.RandInt(int64(npc.MinArtsATK), int64(npc.MaxArtsATK)))
	damage := int(math.Max(float64(rawDamage-targetNpc.SkillDEF), 3))
	if damage < 0 {
		damage = 0
	}
	mC := ConvertPointToLocation(ai.Coordinate)
	skillIds := npc.SkillIds
	skillsCount := len(skillIds) - 1
	randomSkill := utils.RandInt(0, int64(skillsCount))
	resp := MOB_SKILL
	resp.Insert(utils.IntToBytes(uint64(ai.PseudoID), 2, true), 7)           // mob pseudo id
	resp.Insert(utils.IntToBytes(uint64(skillIds[randomSkill]), 4, true), 9) // pet skill id
	resp.Insert(utils.FloatToBytes(mC.X, 4, true), 13)                       // pet-x
	resp.Insert(utils.FloatToBytes(mC.Y, 4, true), 17)                       // pet-x
	resp.Insert(utils.IntToBytes(uint64(targetAi.PseudoID), 2, true), 25)    // target pseudo id
	resp.Insert(utils.IntToBytes(uint64(targetAi.PseudoID), 2, true), 28)    // target pseudo id

	resp.Concat(ai.DealDamageToAI(damage))

	return resp
}
func (ai *AI) CastSkillToPet() []byte {

	pet, ok := GetFromRegister(ai.Server, ai.Map, uint16(ai.TargetPetID)).(*PetSlot)
	if !ok || pet == nil || !pet.IsOnline || pet.HP <= 0 {
		return nil
	}

	pos := GetNPCPosByID(ai.PosID)
	if pos == nil {
		return nil
	}

	npc, ok := GetNpcInfo(ai.NPCpos.NPCID)
	if !ok || npc == nil {
		return nil
	}

	rawDamage := int(utils.RandInt(int64(npc.MinArtsATK), int64(npc.MaxArtsATK)))
	damage := int(math.Max(float64(rawDamage-pet.ArtsDEF), 3))

	dodge := float64(pet.STR)
	reqAcc := dodge + float64(int(pet.Level)-int(npc.Level))*10
	if utils.RandInt(0, 2000) < int64(reqAcc) {
		damage = 0
	}
	if damage < 0 {
		damage = 0
	}
	mC := ConvertPointToLocation(ai.Coordinate)
	skillIds := npc.SkillIds
	skillsCount := len(skillIds) - 1
	randomSkill := utils.RandInt(0, int64(skillsCount))
	resp := MOB_SKILL
	resp.Insert(utils.IntToBytes(uint64(ai.PseudoID), 2, true), 7)           // mob pseudo id
	resp.Insert(utils.IntToBytes(uint64(skillIds[randomSkill]), 4, true), 9) // pet skill id
	resp.Insert(utils.FloatToBytes(mC.X, 4, true), 13)                       // pet-x
	resp.Insert(utils.FloatToBytes(mC.Y, 4, true), 17)                       // pet-x
	resp.Insert(utils.IntToBytes(uint64(pet.PseudoID), 2, true), 25)         // target pseudo id
	resp.Insert(utils.IntToBytes(uint64(pet.PseudoID), 2, true), 28)         // target pseudo id

	resp.Concat(ai.DealDamageToPet(damage))

	return resp
}
func (ai *AI) DealDamage(damage int) []byte {
	if ai.NPCpos == nil {
		return nil
	}
	npc, ok := GetNpcInfo(ai.NPCpos.NPCID)
	if !ok || npc == nil {
		return nil
	}

	resp := MOB_DEAL_DAMAGE
	character, err := FindCharacterByID(ai.TargetPlayerID)
	if err != nil || character == nil {
		return nil
	}
	if character.Meditating { //STOP MEDITATION
		character.Meditating = false
		med := MEDITATION_MODE
		med.Insert(utils.IntToBytes(uint64(character.PseudoID), 2, true), 6) // character pseudo id
		med[8] = 0

		p := nats.CastPacket{CastNear: true, CharacterID: character.ID, Type: nats.MEDITATION_MODE, Data: med}
		if err := p.Cast(); err == nil {
			character.Socket.Write(med)
		}
	}

	stat := character.Socket.Stats

	//reflected := false
	if stat.DamageReflectedRate > 0 && stat.DamageReflectedProbabilty > 0 {
		seed := utils.RandInt(0, 1000)
		if seed <= int64(stat.DamageReflectedProbabilty) {
			damage -= int(float32(damage) * float32(stat.DamageReflectedRate) / 1000)
			//reflected = true
		}
	}

	if damage > 0 {

		if npc.PoisonATK > stat.PoisonDEF {
			character.Poison(npc)
		}
		if npc.ParalysisATK > stat.ParalysisDEF {
			character.Paralysis(npc)
		}
		if npc.ConfusionATK > stat.ConfusionDEF {
			character.Confusion(npc)
		}

		if character.Injury < MAX_INJURY {
			character.Injury += 0.001
			if character.Injury > MAX_INJURY {
				character.Injury = MAX_INJURY
			}
			if character.Injury >= 70 {
				statData, err := character.GetStats()
				if err == nil {
					character.Socket.Write(statData)
				}
			}
		}

	}

	stat.HP = int(math.Max(float64(stat.HP-damage), 0)) // deal damage
	if stat.HP <= 0 {
		ai.TargetPlayerID = 0
	}

	resp.Insert(utils.IntToBytes(uint64(character.PseudoID), 2, true), 5) // character pseudo id
	resp.Insert(utils.IntToBytes(uint64(ai.PseudoID), 2, true), 7)        // mob pseudo id
	resp.Insert(utils.IntToBytes(uint64(stat.HP), 4, true), 9)            // character hp
	resp.Insert(utils.IntToBytes(uint64(stat.CHI), 4, true), 13)          // character chi

	resp.Concat(character.GetHPandChi())

	return resp

}

func (ai *AI) DealDamageToPet(damage int) []byte {

	resp := MOB_DEAL_DAMAGE
	pet, ok := GetFromRegister(ai.Server, ai.Map, uint16(ai.TargetPetID)).(*PetSlot)
	if !ok || pet == nil || !pet.IsOnline || pet.HP <= 0 {
		return nil
	}

	pet.HP = int(math.Max(float64(pet.HP-damage), 0)) // deal damage

	if pet.HP <= 0 {
		ai.TargetPetID = 0
		pet.PetOwner.DismissPet()
	}

	resp.Insert(utils.IntToBytes(uint64(pet.PseudoID), 2, true), 5) // pet pseudo id
	resp.Insert(utils.IntToBytes(uint64(ai.PseudoID), 2, true), 7)  // mob pseudo id
	resp.Insert(utils.IntToBytes(uint64(pet.HP), 2, true), 9)       // pet hp
	resp.Insert(utils.IntToBytes(uint64(pet.CHI), 2, true), 11)     // pet chi
	resp.SetLength(0x24)

	return resp
}
func (ai *AI) DealDamageToAI(damage int) []byte {

	resp := MOB_DEAL_DAMAGE
	targetAi := AIs[ai.TargetAiID]

	pos := GetNPCPosByID(ai.PosID)
	if pos == nil {
		return nil
	}
	targetPos := GetNPCPosByID(targetAi.PosID)
	if targetPos == nil {
		return nil
	}
	targetNpc, ok := GetNpcInfo(targetPos.NPCID)
	if !ok || targetNpc == nil {
		return nil
	}
	npc, ok := GetNpcInfo(pos.NPCID)
	if !ok || npc == nil {
		return nil
	}

	targetAi.HP = int(math.Max(float64(targetAi.HP-damage), 0)) // deal damage

	if damage > targetAi.HP {
		damage = targetAi.HP
	}
	targetAi.HP -= damage
	if targetAi.HP <= 0 {
		targetAi.HP = 0
	}

	resp.Insert(utils.IntToBytes(uint64(targetAi.PseudoID), 2, true), 5) // pet pseudo id
	resp.Insert(utils.IntToBytes(uint64(ai.PseudoID), 2, true), 7)       // mob pseudo id
	resp.Insert(utils.IntToBytes(uint64(targetAi.HP), 2, true), 9)       // pet hp
	resp.Insert(utils.IntToBytes(uint64(targetAi.CHI), 2, true), 11)     // pet chi
	resp.SetLength(0x24)

	if targetAi.HP <= 0 {
		ai.TargetAiID = 0
		time.AfterFunc(time.Second, func() { // disappear mob 1 sec later

			targetAi.IsDead = true

			if targetAi.Once {
				targetAi.Handler = nil
				delete(MapRegister[targetAi.Server][targetAi.Map], targetAi.PseudoID)
				targetAi.PseudoID = 0
			} else {
				respawnTimeX := targetPos.RespawnTime - int(float32(targetPos.RespawnTime)*0.15)
				respawnTimeY := targetPos.RespawnTime + int(float32(targetPos.RespawnTime)*0.15)
				respawnRange := utils.RandInt(int64(respawnTimeX), int64(respawnTimeY))
				minLoc := ConvertPointToLocation(pos.MinLocation)
				maxLoc := ConvertPointToLocation(pos.MaxLocation)
				loc := utils.Location{X: utils.RandFloat(minLoc.X, maxLoc.X), Y: utils.RandFloat(minLoc.Y, maxLoc.Y)}

				targetAi.SetCoordinate(ConvertPointToLocation(loc.String()))
				if ai.Map == 243 { //respawn time in dungeon
					respawnRange = 900
				}
				time.AfterFunc(time.Duration(respawnRange)*time.Second, func() { // respawn mob n secs later

					targetAi.HP = targetNpc.MaxHp
					targetAi.IsDead = false

				})
			}
		})
	}

	return resp
}
func (ai *AI) Kill() {

	targetAi := ai

	pos := GetNPCPosByID(ai.PosID)
	if pos == nil {
		return
	}
	targetPos := GetNPCPosByID(targetAi.PosID)
	if targetPos == nil {
		return
	}
	targetNpc, ok := GetNpcInfo(targetPos.NPCID)
	if !ok || targetNpc == nil {
		return
	}
	npc, ok := GetNpcInfo(pos.NPCID)
	if !ok || npc == nil {
		return
	}

	ai.TargetAiID = 0
	time.AfterFunc(time.Second, func() { // disappear mob 1 sec later

		targetAi.IsDead = true
		if targetAi.Once {
			targetAi.Handler = nil
		} else {
			respawnTimeX := targetPos.RespawnTime - int(float32(targetPos.RespawnTime)*0.15)
			respawnTimeY := targetPos.RespawnTime + int(float32(targetPos.RespawnTime)*0.15)
			respawnRange := utils.RandInt(int64(respawnTimeX), int64(respawnTimeY))
			minLoc := ConvertPointToLocation(pos.MinLocation)
			maxLoc := ConvertPointToLocation(pos.MaxLocation)
			loc := utils.Location{X: utils.RandFloat(minLoc.X, maxLoc.X), Y: utils.RandFloat(minLoc.Y, maxLoc.Y)}

			targetAi.SetCoordinate(ConvertPointToLocation(loc.String()))
			time.AfterFunc(time.Duration(respawnRange)*time.Second, func() { // respawn mob n secs later

				targetAi.HP = targetNpc.MaxHp
				targetAi.IsDead = false

			})
		}
	})

}

func (ai *AI) MovementHandler(token int64, start, end *utils.Location, speed float64) {

	diff := utils.CalculateDistance(start, end)

	if diff < 1 {
		ai.SetCoordinate(end)
		ai.MovementToken = 0
		ai.IsMoving = false
		return
	}

	ai.SetCoordinate(start)
	ai.TargetLocation = *end

	var r []byte //r := []byte{}
	if speed == ai.RunningSpeed {
		r = ai.Move(*end, 2)
	} else {
		r = ai.Move(*end, 1)
	}

	p := nats.CastPacket{CastNear: true, MobID: ai.ID, Data: r, Type: nats.MOB_MOVEMENT}
	p.Cast()

	if diff <= speed { // target is so close
		*start = *end
		time.AfterFunc(time.Duration(diff/speed)*time.Millisecond, func() {
			if token == ai.MovementToken {
				ai.MovementHandler(token, start, end, speed)
			}
		})
	} else { // target is away
		start.X += (end.X - start.X) * speed / diff
		start.Y += (end.Y - start.Y) * speed / diff
		time.AfterFunc(1000*time.Millisecond, func() {
			if token == ai.MovementToken {
				ai.MovementHandler(token, start, end, speed)
			}
		})
	}
}

func (ai *AI) FindClaimer() (*Character, error) {

	dealers := ai.DamageDealers.Values()
	sort.Slice(dealers, func(i, j int) bool {
		di := dealers[i].(*Damage)
		dj := dealers[j].(*Damage)
		return di.Damage > dj.Damage
	})

	if len(dealers) == 0 {
		return nil, nil
	}

	return FindCharacterByID(dealers[0].(*Damage).DealerID)
}

func (ai *AI) DropHandler(claimer *Character) {

	var (
		err error
	)

	npcPos := GetNPCPosByID(ai.PosID)
	if npcPos == nil {
		return
	}

	npc, ok := GetNpcInfo(npcPos.NPCID)
	if !ok || npc == nil {
		return
	}

	isEventBoss := false
	pvpDropMultiplier, bossMultiplier, dropCount, count, minCount, probabmax := 0.0, 0.0, 0, 0, 0, 1000
	baseLocation := ConvertPointToLocation(ai.Coordinate)
	if funk.Contains(bosses, npc.ID) {
		bossMultiplier = 20
		minCount = 20

	} else if funk.Contains(eventBosses, npc.ID) {
		bossMultiplier = 50.0
		minCount = 48
		isEventBoss = true

	} else if !npcPos.Attackable && claimer.PickaxeActivated() {
		bossMultiplier = 0.4
	}
	if ai.Map == 27 {
		bossMultiplier = 0.1
	}
	if funk.Contains(PVPServers, int16(ai.Server)) {
		pvpDropMultiplier = 0.05
	}

BEGIN:

	id := npc.DropID

	drop, ok := GetDropInfo(id)
	if drop == nil {
		return
	}

	itemID := 0
	end := false

	for ok {
		index := 0
		seed := int(utils.RandInt(0, 1000))
		if drop == nil {
			return
		}
		items := drop.Items
		probabilities := drop.Probabilities

		stat, err := FindStatByID(claimer.ID)
		if err != nil || stat == nil {
			return
		}
		totalDropRate := ((DROP_RATE * (stat.DropMultiplier + pvpDropMultiplier)) + bossMultiplier)
		if ai.Map == 26 || ai.Map == 27 {
			totalDropRate = (1 * stat.DropMultiplier) + bossMultiplier
		} else if ai.Map == 97 || ai.Map == 98 || ai.Map == 99 {
			totalDropRate *= 1.5
		} else {
			totalDropRate = ((DROP_RATE * (stat.DropMultiplier + pvpDropMultiplier)) + bossMultiplier)
		}

		dropFailRate := float64(probabmax - probabilities[len(probabilities)-1])
		dropFailRate /= totalDropRate
		newDropFailRate := float64(probabmax) - dropFailRate
		probMultiplier := float64(probabilities[len(probabilities)-1]) / newDropFailRate

		if float64(probabilities[len(probabilities)-1])*totalDropRate < 900 {
			probMultiplier = 1
			probabilities = funk.Map(probabilities, func(prob int) int {
				return int(float64(prob) * totalDropRate)
			}).([]int)
		}

		seed = int(float64(seed) * probMultiplier)
		index = sort.SearchInts(probabilities, seed)
		if index >= len(items) {
			if count >= minCount {
				end = true
				break
			} else {
				drop, ok = GetDropInfo(id)
				continue
			}
		}

		itemID = items[index]
		item, kk := GetItemInfo(int64(itemID))
		if kk {
			itemType := item.GetType()
			if itemType == QUEST_TYPE {
				drop, ok = GetDropInfo(id)
				continue
			}
			rand := utils.RandInt(0, 1000)
			if rand > 500 { //item.BeastType != 0 &&
				drop, ok = GetDropInfo(id)
				continue
			}
		}
		drop, ok = GetDropInfo(itemID)
	}
	if itemID > 0 && !end { // can drop an item
		count++
		if count >= 100 {
			return
		}
		go func() {
			resp := utils.Packet{}
			isRelic := false

			if _, ok := Relics[itemID]; ok { // relic drop
				if !RELIC_DROP_ENABLED {
					return
				}
				if claimer.Map == 10 {
					return
				}
				if claimer.Map == 93 || claimer.Map == 94 || claimer.Map == 95 || claimer.Map == 96 || claimer.Map == 97 || claimer.Map == 98 || claimer.Map == 99 {
					random := funk.RandomInt(0, 100)
					if random > 2 {
						return
					} else {
						random = funk.RandomInt(0, 100)
						if random > 35 {
							return
						}
					}
				}
				/*for _, reqItem := range Relics[itemID].requiredItems {
					if reqItem != 0 {
						f := func(item *InventorySlot) bool {
							return item.InUse
						}
						_, item, err := claimer.FindItemInInventory(f, reqItem)
						if err != nil {
							return
						} else if item == nil {
							return
						}
					}
				}*/

				resp.Concat(claimer.RelicDrop(int64(itemID)))
				isRelic = true
			}

			item, ok := GetItemInfo(int64(itemID))
			if ok && item != nil {
				seed := int(utils.RandInt(0, 1000))
				plus := byte(0)
				for i := 0; i < len(plusRates) && !isRelic; i++ {
					if seed > plusRates[i] {
						plus++
						continue
					}
					break
				}

				drop := NewSlot()
				drop.ItemID = item.ID
				drop.Quantity = 1
				drop.Plus = plus

				if item.Slot >= 0 && item.Slot <= 9 {
					discseed := utils.RandInt(1, 1000)
					if discseed < 100 {
						drop.ItemType = 1
						drop.Plus = 0
						plus = 0
					}
				}
				if item.Timer > 0 {
					drop.Quantity = uint(item.Timer)
				}

				var upgradesArray []byte
				itemType := item.GetType()
				if itemType == WEAPON_TYPE {
					upgradesArray = WeaponUpgrades
				} else if itemType == ARMOR_TYPE {
					upgradesArray = ArmorUpgrades
				} else if itemType == ACC_TYPE {
					upgradesArray = AccUpgrades
				} else if itemType == PENDENT_TYPE || item.ID == 254 || item.ID == 255 || item.ID == 242 {
					if plus == 0 {
						plus = 1
						drop.Plus = 1
					}
					upgradesArray = []byte{byte(item.ID)}

				} else {
					plus = 0
					drop.Plus = 0
				}

				for i := byte(0); i < plus; i++ {
					index := utils.RandInt(0, int64(len(upgradesArray)))
					drop.SetUpgrade(int(i), upgradesArray[index])
				}

				if isRelic || !npcPos.Attackable {

					slot := int16(-1)
					if npcPos.Attackable {
						slot, err = claimer.FindFreeSlot()
						if slot == 0 || err != nil {
							return
						}
					}

					data, _, err := claimer.AddItem(drop, slot, true)
					if err != nil || data == nil {
						return
					}
					claimer.Socket.Write(*data)

				} else {

					offset := dropOffsets[dropCount%len(dropOffsets)]
					dropCount++

					dr := &Drop{Server: ai.Server, Map: ai.Map, Claimer: claimer, Item: drop,
						Location: utils.Location{X: baseLocation.X + offset.X, Y: baseLocation.Y + offset.Y}}

					if isEventBoss || ai.Map == 10 {
						dr.Claimer = nil
					}
					time.AfterFunc(FREEDROP_LIFETIME, func() { //ALL PLAYER CAN PICKUP THE ITEMS
						dr.Claimer = nil
					})

					dr.GenerateIDForDrop(ai.Server, ai.Map)

					dropID := uint16(dr.ID)

					time.AfterFunc(DROP_LIFETIME, func() { // remove drop after timeout
						RemoveDrop(ai.Server, ai.Map, dropID)
					})
				}

				p := nats.CastPacket{CastNear: true, MobID: ai.ID, Data: resp, Type: nats.ITEM_DROP}
				if isRelic {
					p = nats.CastPacket{CastNear: false, Data: resp, Type: nats.ITEM_DROP}
				} else {
					p = nats.CastPacket{CastNear: true, MobID: ai.ID, Data: resp, Type: nats.BOSS_DROP}
				}

				if err := p.Cast(); err != nil {
					return
				}
				if npc.ID == 424901 ||
					npc.ID == 424902 ||
					npc.ID == 424903 {
					return
				}

			}
		}()
	}

	if !npcPos.Attackable {
		end = true
	}

	if !end {
		goto BEGIN
	}

}
func RemoveDrop(server int, mapID int16, dropID uint16) {
	drMutex.RLock()
	_, ok := DropRegister[server][mapID][dropID]
	drMutex.RUnlock()

	if ok {
		drMutex.Lock()
		delete(DropRegister[server][mapID], dropID)
		drMutex.Unlock()
	}
}

func (ai *AI) ShouldGoBack() bool {

	npcPos := GetNPCPosByID(ai.PosID)
	if npcPos == nil {
		return false
	}
	aiMinCoordinate := ConvertPointToLocation(npcPos.MinLocation)
	aiMaxCoordinate := ConvertPointToLocation(npcPos.MaxLocation)
	aiCoordinate := ConvertPointToLocation(ai.Coordinate)

	if ai.NPCpos.NPCID == 50073 {
		return false
	}
	if ai.Map == 10 || ai.Map == 212 {
		return false
	} else if ai.Faction == 3 {
		return true
	} else if ai.Map == 233 {
		if utils.CalculateDistance(aiMinCoordinate, aiCoordinate) < 50 {
			return false
		}
	} else if ai.Map == 43 || ai.Map == 44 || ai.Map == 63 || ai.Map == 64 || ai.Map == 65 {
		if utils.CalculateDistance(aiMinCoordinate, aiCoordinate) < 100 {
			return false
		}
	} else if aiCoordinate.X >= aiMinCoordinate.X && aiCoordinate.X <= aiMaxCoordinate.X &&
		aiCoordinate.Y >= aiMinCoordinate.Y && aiCoordinate.Y <= aiMaxCoordinate.Y {
		return false
	}

	return true
}

func GeneratePoint(location *utils.Location) utils.Location {

	r := 2.0
	alfa := utils.RandFloat(0, 360)
	targetX := location.X + r*float64(math.Cos(alfa*math.Pi/180))
	targetY := location.Y + r*float64(math.Sin(alfa*math.Pi/180))

	return utils.Location{X: targetX, Y: targetY}
}

func CountYingYangMobs(Map int16) {
	var mobs *DungeonMobsCounter = new(DungeonMobsCounter)
	i, j, k, l := 0, 0, 0, 0

	for _, mob := range AIsByMap[1][Map] {
		npcPos, _ := FindNPCPosByID(mob.PosID)
		if npcPos == nil {
			continue
		}
		if npcPos.NPCID == 60001 || npcPos.NPCID == 60002 || npcPos.NPCID == 60015 || npcPos.NPCID == 60016 {
			i++
		} else if npcPos.NPCID == 60004 || npcPos.NPCID == 60018 {
			j++
		} else if npcPos.NPCID == 60006 || npcPos.NPCID == 60007 || npcPos.NPCID == 60020 || npcPos.NPCID == 60021 {
			k++
		} else if npcPos.NPCID == 60009 || npcPos.NPCID == 60010 || npcPos.NPCID == 60011 || npcPos.NPCID == 60012 ||
			npcPos.NPCID == 60023 || npcPos.NPCID == 60024 || npcPos.NPCID == 60025 || npcPos.NPCID == 60026 {
			l++
		}

		mobs.BlackBandits = i
		mobs.Rogues = j
		mobs.Ghosts = k
		mobs.Animals = l
		YingYangMobsCounter[Map] = mobs
	}

}
func CountSeasonCave(server int) {
	var mobs *SeasonCaveMobs = new(SeasonCaveMobs)
	i, j, k := 0, 0, 0
	for _, mob := range AIsByMap[server][212] {
		npcPos, _ := FindNPCPosByID(mob.PosID)
		if npcPos.NPCID == 45003 {
			i++
		} else if npcPos.NPCID == 45004 {
			j++
		} else if npcPos.NPCID == 45005 {
			k++
		} else if npcPos.NPCID == 45006 {
			mobs.Centipede = 1
		} else if npcPos.NPCID == 45007 {
			mobs.Demon = 1
		}
	}
	mobs.Bats = i
	mobs.Spiders = j
	mobs.Snakes = k
	SeasonCaveMobsCounter[server] = mobs
}
func (ai *AI) RewardNcash() {

	npcPos, err := FindNPCPosByID(ai.PosID)
	if err != nil {
		return
	}
	npc, ok := GetNpcInfo((npcPos.NPCID))
	if !ok || npc == nil {
		return
	}

	characterMutex.RLock()
	allChars := funk.Values(characters)
	characterMutex.RUnlock()

	filtered := funk.Filter(allChars, func(c *Character) bool {

		if c.Socket == nil || !c.IsOnline {
			return false
		}

		user := c.Socket.User
		stat := c.Socket.Stats

		if user == nil || stat == nil {
			return false
		}

		return user.ConnectedServer == ai.Server && c.Map == ai.Map && stat.HP > 0
	})

	characters := filtered.([]*Character)

	for _, player := range characters {
		dealer := ai.DamageDealers.Get(player.ID)
		if dealer != nil && player != nil { //a dat dmg
			char, err := FindCharacterByID(dealer.(*Damage).DealerID)
			if err != nil {
				continue
			} else if !char.IsActive || !char.IsOnline {
				continue
			} else if char.Level > 100 && npc.Level <= 100 {
				continue
			}

			totalDamageDealt := dealer.(*Damage).Damage
			reward := uint64(float64(npc.GoldDrop) * (float64(totalDamageDealt) / float64(npc.MaxHp)))
			if reward > 400 {
				reward = 400
			}
			if char.IsActive && char.IsOnline {
				char.Socket.User.NCash += reward
				char.Socket.User.Update()
				text := fmt.Sprintf("%s earned %d by dealing %d to (%s)", char.Name, reward, totalDamageDealt, npc.Name)
				char.Socket.Write(messaging.InfoMessage(fmt.Sprintf("%s earned %d by dealing %d to (%s)", char.Name, reward, totalDamageDealt, npc.Name)))
				utils.NewLog("logs/ci_ncash_rewards.txt", text)

			}
		}
	}

	ai.DamageDealers.Clear()
}
