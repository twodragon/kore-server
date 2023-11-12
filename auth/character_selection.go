package auth

import (
	"encoding/json"
	"log"

	"github.com/twodragon/kore-server/database"
	"github.com/twodragon/kore-server/nats"
	"github.com/twodragon/kore-server/utils"

	NATS "github.com/nats-io/nats.go"
)

type CharacterSelectionHandler struct {
	id int
}

var (
	CHARACTER_SELECTED = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x01, 0x05, 0x0A, 0x00, 0x55, 0xAA}
)

func (csh *CharacterSelectionHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	csh.id = int(utils.BytesToInt(data[6:10], true))
	return csh.selectCharacter(s)
}

func (csh *CharacterSelectionHandler) selectCharacter(s *database.Socket) ([]byte, error) {

	character, err := database.FindCharacterByID(csh.id)
	if err != nil {
		return nil, err
	}
	if character == nil {
		return nil, nil
	}
	if s.User == nil {
		return nil, nil
	}
	if character.UserID != s.User.ID {
		return nil, nil
	}
	if s.User.ConnectedIP == "" {
		return nil, nil
	}

	character.IsOnline = false
	character.Socket = s
	s.Character = character
	err = database.GenerateIDforCharacter(s.Character)
	if err != nil {
		return nil, err
	}

	//go func() {
	func() {
		if s.HoustonSub != nil {
			return
		}

		s.HoustonSub, err = nats.Connection().Subscribe(nats.HOUSTON_CH, func(msg *NATS.Msg) {
			err := HoustonHandler(s, msg)
			if err != nil {
				log.Println(err)
			}
		})
		if err != nil {
			log.Fatalln(err)
		}
	}()

	s.Stats, err = database.FindStatByID(character.ID)
	if err != nil {
		return nil, err
	}

	s.Skills, err = database.FindSkillsByID(character.ID)
	if err != nil {
		return nil, err
	}

	resp := CHARACTER_SELECTED

	if !s.Character.CanAttack(character) {
		resp.Concat([]byte{0xAA, 0x55, 0x02, 0x00, 0x2A, 0x04, 0x55, 0xAA})
	}

	return resp, nil
}

func HoustonHandler(s *database.Socket, msg *NATS.Msg) error {

	var packet nats.CastPacket
	err := json.Unmarshal(msg.Data, &packet)
	if err != nil {
		return err
	}

	ok := false
	resp := utils.Packet(packet.Data)

	if packet.CharacterID > 0 {
		s.Character.OnSight.PlayerMutex.RLock()
		_, ok = s.Character.OnSight.Players[packet.CharacterID]
		s.Character.OnSight.PlayerMutex.RUnlock()

	} else if packet.MobID > 0 {
		s.Character.OnSight.MobMutex.RLock()
		_, ok = s.Character.OnSight.Mobs[packet.MobID]
		s.Character.OnSight.MobMutex.RUnlock()

	} else if packet.Location != nil {
		coordinate := database.ConvertPointToLocation(s.Character.Coordinate)
		distance := utils.CalculateDistance(coordinate, &utils.Location{X: packet.Location.X, Y: packet.Location.Y})
		if distance <= packet.MaxDistance {
			ok = true
		}

	} else if packet.DropID > 0 {
		s.Character.OnSight.DropsMutex.RLock()
		_, ok = s.Character.OnSight.Drops[packet.DropID]
		s.Character.OnSight.DropsMutex.RUnlock()

	} else if packet.PetID > 0 {
		s.Character.OnSight.PetsMutex.RLock()
		_, ok = s.Character.OnSight.Pets[packet.PetID]
		s.Character.OnSight.PetsMutex.RUnlock()

	} else if packet.BabyPetID > 0 {
		s.Character.OnSight.BabyPetsMutex.RLock()
		_, ok = s.Character.OnSight.BabyPets[packet.BabyPetID]
		s.Character.OnSight.BabyPetsMutex.RUnlock()

	}

	if ok && packet.Type == nats.PLAYER_RESPAWN {
		r := utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x21, 0x02, 0x00, 0x55, 0xAA}
		c, err := database.FindCharacterByID(packet.CharacterID)
		if err != nil || c == nil {
			return nil
		}

		r.Insert(utils.IntToBytes(uint64(c.PseudoID), 2, true), 6) // pseudo id

		d, err := c.SpawnCharacter()
		if err != nil {
			return err
		}
		r.Concat(d)
		r.Concat(c.GetHPandChi())

		s.Write(r)
	}

	if (!packet.CastNear) || (packet.CastNear && ok) {
		_, err = s.Conn.Write(resp)
		return err
	}

	return nil
}
