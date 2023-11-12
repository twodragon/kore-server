package auth

import (
	"encoding/binary"
	"sync"

	"github.com/twodragon/kore-server/database"
	"github.com/twodragon/kore-server/messaging"
	"github.com/twodragon/kore-server/nats"
	"github.com/twodragon/kore-server/utils"
)

var (
	CreateCharMutex sync.RWMutex
)

type CancelCharacterCreationHandler struct {
}

type CharacterCreationHandler struct {
	characterType int
	faction       int
	height        int
	name          string
	headstyle     int64
	facestyle     int64
}

var (
	CHARACTER_CREATED = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x01, 0x03, 0x0A, 0x00, 0x00, 0x00, 0x55, 0xAA}
)

func (ccch *CancelCharacterCreationHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	lch := &ListCharactersHandler{}
	return lch.showCharacterMenu(s)
}

func (cch *CharacterCreationHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	CreateCharMutex.Lock()
	defer CreateCharMutex.Unlock()

	index := 7
	length := int(data[index])
	index += 1

	cch.name = string(data[8 : length+8])
	index += len(cch.name)

	cch.characterType = int(data[index])
	index += 1

	characters, err := database.FindCharactersByUserID(s.User.ID)
	if err != nil {
		return nil, err
	}

	if len(characters) > 0 {
		cch.faction = characters[0].Faction
	} else {
		cch.faction = int(data[index])
	}
	index += 1

	cch.height = int(data[index])
	headint := utils.BytesToInt(data[index:index+4], true)
	cch.headstyle = headint
	index += 4
	faceint := utils.BytesToInt(data[index:index+4], true)
	cch.facestyle = faceint
	headinfo, ok := database.GetItemInfo(headint)
	if !ok || headinfo == nil {
		headint = 0
	}
	faceinfo, ok := database.GetItemInfo(faceint)
	if !ok || faceinfo == nil {
		faceint = 0
	}

	return cch.createCharacter(s)
}

func (cch *CharacterCreationHandler) createCharacter(s *database.Socket) ([]byte, error) {

	ok, err := database.IsValidUsername(cch.name)
	if err != nil {
		return nil, err
	} else if !ok {
		return messaging.SystemMessage(messaging.INVALID_NAME), nil
	} else if cch.faction == 0 {
		return messaging.SystemMessage(messaging.EMPTY_FACTION), nil
	}

	character := &database.Character{
		Type:         cch.characterType,
		UserID:       s.User.ID,
		Name:         cch.name,
		Epoch:        0,
		Faction:      cch.faction,
		Height:       cch.height,
		Level:        1,
		Gold:         0,
		Exp:          0,
		Class:        0,
		IsOnline:     false,
		IsActive:     false,
		Map:          1,
		HTVisibility: 0,
		WeaponSlot:   3,
		RunningSpeed: 5.6,
		GuildID:      -1,
		Slotbar:      []byte{},
		Coordinate:   database.ConvertPointToCoordinate(55.0, 225.0),
		AidTime:      7200,
		HeadStyle:    cch.headstyle,
		FaceStyle:    cch.facestyle,
	}

	err = character.Create()
	if err != nil {
		return nil, err
	}

	if database.StarterItems[cch.characterType] != nil {
		for _, itemID := range database.StarterItems[cch.characterType].ItemIDs {
			//character.AddItem(&database.InventorySlot{ItemID: item, Quantity: 1}, -1, false)
			quantity := int64(1)
			info, _ := database.GetItemInfo(int64(itemID))
			if info == nil {
				continue
			}
			if info.Timer > 0 {
				quantity = int64(info.Timer)
			}
			item := &database.InventorySlot{ItemID: int64(itemID), Quantity: uint(quantity)}

			if info.GetType() == database.PET_TYPE {
				petInfo := database.Pets[itemID]
				petExpInfo := database.PetExps[petInfo.Level-1]
				targetExps := []int{petExpInfo.ReqExpEvo1, petExpInfo.ReqExpEvo2, petExpInfo.ReqExpEvo3, petExpInfo.ReqExpHt, petExpInfo.ReqExpDivEvo1, petExpInfo.ReqExpDivEvo2, petExpInfo.ReqExpDivEvo3, petExpInfo.ReqExpDivEvo4, petExpInfo.ReqExpDivEvo4}
				item.Pet = &database.PetSlot{
					Fullness: 100, Loyalty: 100,
					Exp:   float64(targetExps[petInfo.Evolution-1]),
					HP:    petInfo.BaseHP,
					Level: byte(petInfo.Level),
					Name:  petInfo.Name,
					CHI:   petInfo.BaseChi}
			}
			r, _, err := character.AddItem(item, -1, false)
			if err != nil || r == nil {
				continue
			}
		}
	}

	if cch.characterType == 50 {
		buffinfo := database.BuffInfections[int(277)]
		buff := &database.Buff{ID: int(277), CharacterID: character.ID, Name: buffinfo.Name, BagExpansion: false, StartedAt: 0, Duration: 0, CanExpire: false}
		buff.Create()
		character.Map = 70
		character.Coordinate = database.ConvertPointToCoordinate(database.SavePoints[70].X, database.SavePoints[70].Y)
	} else if cch.characterType == 51 {
		buffinfo := database.BuffInfections[int(280)]
		buff := &database.Buff{ID: int(280), CharacterID: character.ID, Name: buffinfo.Name, BagExpansion: false, StartedAt: 0, Duration: 0, CanExpire: false}
		buff.Create()
		character.Map = 70
		character.Coordinate = database.ConvertPointToCoordinate(database.SavePoints[70].X, database.SavePoints[70].Y)
	}

	character.Update()

	stat := &database.Stat{}
	err = stat.Create(character)
	if err != nil {
		return nil, err
	}

	skills := &database.Skills{}
	err = skills.Create(character)
	if err != nil {
		return nil, err
	}

	resp := CHARACTER_CREATED
	length := int16(len(cch.name)) + 10
	resp.SetLength(length)

	resp.Insert(utils.IntToBytes(uint64(character.ID), 4, true), 9) // character id

	resp[13] = byte(len(cch.name)) // character name length

	resp.Insert([]byte(cch.name), 14) // character name

	lch := &ListCharactersHandler{}
	data, err := lch.showCharacterMenu(s)
	if err != nil {
		return nil, err
	}

	resp.Concat(data)

	ANNOUNCEMENT := utils.Packet{0xAA, 0x55, 0x54, 0x00, 0x71, 0x14, 0x51, 0x55, 0xAA}
	msg := "New Hero hopped into the world. "
	announce := ANNOUNCEMENT
	index := 6
	announce[index] = byte(len(character.Name) + len(msg))
	index++
	announce.Insert([]byte("["+character.Name+"]"), index) // character name
	index += len(character.Name) + 2
	announce.Insert([]byte(msg), index) // character name
	announce.SetLength(int16(binary.Size(announce) - 6))
	p := nats.CastPacket{CastNear: false, Data: announce}
	p.Cast()
	return resp, nil
}
