package database

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/thoas/go-funk"
	"github.com/twodragon/kore-server/nats"
	"github.com/twodragon/kore-server/utils"
	"gopkg.in/guregu/null.v3"
)

var (
	BabyPets = make(map[int]*BabyPet)
)

type BabyPet struct {
	ID         int       `db:"id"`
	Npcid      int64     `db:"npcid"`
	Mapid      int16     `db:"mapid"`
	Server     int       `db:"server"`
	Max_HP     int       `db:"max_hp"`
	HP         int       `db:"hp"`
	Hunger     int       `db:"hunger"`
	Created_at null.Time `db:"created_at"`
	Coordinate string    `db:"coordinates"`
	OwnerID    int       `db:"owner_id"`
	Name       string    `db:"name"`
	Level      int       `db:"level"`

	Handler  func() `db:"-" json:"-"`
	IsMoving bool   `db:"-" json:"-"`

	PseudoID       int            `db:"-"`
	TargetLocation utils.Location `db:"-"`
	MovementToken  int64          `db:"-" json:"-"`

	IsDead         bool                `db:"-"`
	PlayersMutex   sync.RWMutex        `db:"-"`
	OnSightPlayers map[int]interface{} `db:"-" json:"players"`
}

func (c *BabyPet) Update() error {
	_, err := pgsql_DbMap.Update(c)
	return err
}

func (c *BabyPet) Create() error {
	return pgsql_DbMap.Insert(c)
}

func GetBabyPets() error {
	var babies []*BabyPet
	query := `select * from hops.baby_pets`

	if _, err := pgsql_DbMap.Select(&babies, query); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("GetBabyPets: %s", err.Error())
	}

	for _, cr := range babies {
		cr.IsDead = false
		BabyPets[cr.ID] = cr
	}
	return nil
}

func (b *BabyPet) SpawnBabyPet() ([]byte, error) {

	owner, err := FindCharacterByID(b.OwnerID)
	if err != nil {
		return nil, err
	}
	if owner == nil {
		return nil, fmt.Errorf("SpawnBabyPet: Owner not found")
	}
	r := PET_APPEARED
	r.Insert(utils.IntToBytes(uint64(b.PseudoID), 2, true), 6) // pet pseudo id
	r.Insert(utils.IntToBytes(uint64(15710030), 4, true), 8)   // pet npc id
	r.Insert(utils.IntToBytes(uint64(b.Level), 4, true), 12)   // pet level
	r.Overwrite(utils.IntToBytes(3, 4, true), 16)              //Pets to neutral
	//r.Insert([]byte{0x09, 0x57, 0x69, 0x6C, 0x64, 0x20, 0x42, 0x6F, 0x61, 0x72}, 20)
	//r.Insert(utils.IntToBytes(uint64(len(pet.Name)), 1, true), 20)
	//	index++
	index := 0
	if b.Name != "" {
		r.Insert(utils.IntToBytes(uint64(len(owner.Name+"|"+b.Name)), 1, true), 20)
		r.Insert([]byte(owner.Name+"|"+b.Name), 21) // pet name
		index = len(owner.Name+"|"+b.Name) + 21
	} else {
		r.Insert(utils.IntToBytes(uint64(len(owner.Name)), 1, true), 20)
		r.Insert([]byte(owner.Name), 21) // pet name
		index = len(owner.Name) + 21
	}

	coordinate := ConvertPointToLocation(b.Coordinate)
	r.Insert(utils.IntToBytes(uint64(b.HP), 4, true), index)        // pet hp
	r.Insert(utils.IntToBytes(uint64(b.HP), 4, true), index+4)      // pet chi
	r.Insert(utils.IntToBytes(uint64(b.Max_HP), 4, true), index+8)  // pet max hp
	r.Insert(utils.IntToBytes(uint64(b.Max_HP), 4, true), index+12) // pet max chi
	r.Insert(utils.IntToBytes(3, 2, true), index+16)                //
	r.Insert(utils.FloatToBytes(coordinate.X, 4, true), index+18)   // coordinate-x
	r.Insert(utils.FloatToBytes(coordinate.Y, 4, true), index+22)   // coordinate-y
	r.Insert(utils.FloatToBytes(12, 4, true), index+26)             // z?
	r.Insert(utils.FloatToBytes(coordinate.X, 4, true), index+30)   // coordinate-x
	r.Insert(utils.FloatToBytes(coordinate.Y, 4, true), index+34)   // coordinate-y
	r.Insert(utils.FloatToBytes(12, 4, true), index+38)             // z?
	r.Insert([]byte{0x00, 0x00, 0x00, 0x00}, index+42)
	r = append(r[:index+42], r[index+50:]...)
	r.Overwrite(utils.Packet{0x00, 0x00, 0x00, 0x00, 0xFF, 0xFF, 0x00, 0xE8, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, index+42)

	r.SetLength(int16(binary.Size(r) - 6))

	return r, nil
}

func (baby *BabyPet) BabyPetHandler() {

	npc, ok := GetNpcInfo(int(baby.Npcid))
	if !ok {
		return
	}
	if baby.IsDead {
		baby.HP = 0
		goto OUT
	}

	if len(baby.OnSightPlayers) == 0 && baby.HP > 0 && baby.HP != npc.MaxHp {
		goto OUT
	}

	if len(baby.OnSightPlayers) > 0 && baby.HP > 0 {
		baby.PlayersMutex.RLock()
		ids := funk.Keys(baby.OnSightPlayers).([]int)
		baby.PlayersMutex.RUnlock()

		for _, id := range ids {
			remove := false

			c, err := FindCharacterByID(id)
			if err != nil || c == nil || !c.IsOnline || c.Map != baby.Mapid {
				remove = true
			}

			if c != nil {
				user, err := FindUserByID(c.UserID)
				if err != nil || user == nil || user.ConnectedIP == "" || user.ConnectedServer == 0 || user.ConnectedServer != baby.Server {
					remove = true
				}
			}

			if remove {
				baby.PlayersMutex.Lock()
				delete(baby.OnSightPlayers, id)
				baby.PlayersMutex.Unlock()
			}
		}

		if baby.IsMoving {
			goto OUT
		}

		if baby.HP > 0 { // Idle mode

			coordinate := ConvertPointToLocation(baby.Coordinate)

			if utils.RandInt(0, 1000) < 100 { // 75% chance to move
				baby.IsMoving = true
				targetX := utils.RandFloat(coordinate.X+utils.RandFloat(-15, 15), coordinate.X+utils.RandFloat(-15, 15))
				targetY := utils.RandFloat(coordinate.Y+utils.RandFloat(-15, 15), coordinate.Y+utils.RandFloat(-15, 15))
				target := utils.Location{X: targetX, Y: targetY}

				baby.TargetLocation = target

				token := baby.MovementToken
				for token == baby.MovementToken {
					baby.MovementToken = utils.RandInt(1, math.MaxInt64)
				}

				go baby.MovementHandler(baby.MovementToken, coordinate, &target, 3)

			}

		}
	}

OUT:
	delay := utils.RandFloat(1.0, 1.5) * 1500
	time.AfterFunc(time.Duration(delay)*time.Millisecond, func() {
		if baby.Handler != nil {
			baby.BabyPetHandler()
		}
	})
}

func (bby *BabyPet) MovementHandler(token int64, start, end *utils.Location, speed float64) {

	diff := utils.CalculateDistance(start, end)

	if diff < 1 {
		bby.Coordinate = ConvertPointToCoordinate(end.X, end.Y)
		bby.MovementToken = 0
		bby.IsMoving = false
		return
	}

	bby.Coordinate = ConvertPointToCoordinate(start.X, start.Y)
	bby.TargetLocation = *end

	r := bby.Move(*end, 1)

	p := nats.CastPacket{CastNear: true, BabyPetID: bby.ID, Data: r, Type: nats.PET_MOVEMENT}
	p.Cast()

	if diff <= speed { // target is so close
		*start = *end
		time.AfterFunc(time.Duration(diff/speed)*time.Millisecond, func() {
			if token == bby.MovementToken {
				bby.MovementHandler(token, start, end, speed)
			}
		})
	} else { // target is away
		start.X += (end.X - start.X) * speed / diff
		start.Y += (end.Y - start.Y) * speed / diff
		time.AfterFunc(1000*time.Millisecond, func() {
			if token == bby.MovementToken {
				bby.MovementHandler(token, start, end, speed)
			}
		})
	}
}
func (bby *BabyPet) Move(targetLocation utils.Location, runningMode byte) []byte {

	resp := MOB_MOVEMENT
	currentLocation := ConvertPointToLocation(bby.Coordinate)

	resp.Insert(utils.IntToBytes(uint64(bby.PseudoID), 2, true), 5) // pet pseudo id
	resp[7] = runningMode
	resp.Insert(utils.FloatToBytes(currentLocation.X, 4, true), 8)  // current coordinate-x
	resp.Insert(utils.FloatToBytes(currentLocation.Y, 4, true), 12) // current coordinate-y
	resp.Insert(utils.FloatToBytes(targetLocation.X, 4, true), 20)  // current coordinate-x
	resp.Insert(utils.FloatToBytes(targetLocation.Y, 4, true), 24)  // current coordinate-y

	speed := 3.0
	resp.Insert(utils.FloatToBytes(speed, 4, true), 32) // speed

	return resp
}
