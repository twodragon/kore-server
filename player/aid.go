package player

import (
	"github.com/thoas/go-funk"
	"github.com/twodragon/kore-server/database"
	"github.com/twodragon/kore-server/messaging"
	"github.com/twodragon/kore-server/nats"
	"github.com/twodragon/kore-server/utils"
)

type (
	AidHandler struct{}
)

func (h *AidHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	if len(data) < 281 {
		return nil, nil
	}

	if sale := database.FindSale(s.Character.PseudoID); sale != nil {
		return nil, nil
	}
	c := s.Character

	if funk.Contains(database.DisabledAIDMaps, c.Map) {
		return nil, nil
	}
	if !c.CanMove {
		msg := "Unequip unusable items to start AID."
		s.Write(messaging.InfoMessage(msg))
		return nil, nil
	}

	activated := data[5] == 1

	if len(data) > 20 {
		petfood1 := utils.BytesToInt(data[269:273], true)
		petfood1percent := utils.BytesToInt(data[265:269], true)
		petchi := utils.BytesToInt(data[281:285], true)
		petchipercent := utils.BytesToInt(data[277:281], true)
		s.Character.PlayerAidSettings = &database.AidSettings{PetChiItemID: petchi, PetChiPercent: uint(petchipercent), PetFood1ItemID: petfood1, PetFood1Percent: uint(petfood1percent)}
	}
	resp := utils.Packet{}

	s.Character.AidMode = activated

	if activated == true {
		characterCoordinate := database.ConvertPointToLocation(c.Coordinate)
		c.AidStartingPosition = characterCoordinate.String()
	}

	resp.Concat(s.Character.AidStatus())
	p := &nats.CastPacket{CastNear: true, CharacterID: s.Character.ID, Data: s.Character.GetHPandChi()}
	p.Cast()

	return resp, nil
}
