package player

import (
	"time"

	"github.com/twodragon/kore-server/database"
	"github.com/twodragon/kore-server/utils"

	"github.com/thoas/go-funk"
)

type (
	SendPartyRequestHandler    struct{}
	RespondPartyRequestHandler struct{}
	LeavePartyHandler          struct{}
	ExpelFromPartyHandler      struct{}
)

var (
	PARTY_REQUEST          = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x52, 0x01, 0x0A, 0x00, 0x00, 0x00, 0x55, 0xAA}
	PARTY_REQUEST_REJECTED = utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x52, 0x02, 0x52, 0x03, 0x55, 0xAA}
	EXPEL_PARTY_MEMBER     = utils.Packet{0xAA, 0x55, 0x08, 0x00, 0x52, 0x06, 0x0A, 0x00, 0x55, 0xAA}
	PARTY_SETTINGS         = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x72, 0x0D, 0x11, 0x55, 0xAA}
)

func (h *SendPartyRequestHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	c := s.Character
	pseudoID := uint16(utils.BytesToInt(data[6:8], true))
	member := database.FindCharacter(s.User.ConnectedServer, pseudoID)
	if member == nil || database.FindParty(member) != nil {
		return nil, nil
	}

	party := database.FindParty(c)
	if party == nil {
		party = &database.Party{}
		party.Leader = c
		c.PartyID = c.UserID
		party.Create()
		//s.Write(database.GetPartyMemberData(party.Leader))
	} else if len(party.GetMembers()) >= 4 {
		return nil, nil
	}

	resp := PARTY_REQUEST
	setting := 33
	if c.GroupSettings.ExperienceSharingMethod == 1 {
		if c.GroupSettings.LootDistriburionMethod == 2 {
			setting = 40
		} else if c.GroupSettings.LootDistriburionMethod == 3 {
			setting = 34
		} else if c.GroupSettings.LootDistriburionMethod == 4 {
			setting = 36
		}
	} else if c.GroupSettings.ExperienceSharingMethod == 2 {
		if c.GroupSettings.LootDistriburionMethod == 1 {
			setting = 17
		} else if c.GroupSettings.LootDistriburionMethod == 2 {
			setting = 24
		} else if c.GroupSettings.LootDistriburionMethod == 3 {
			setting = 18
		} else if c.GroupSettings.LootDistriburionMethod == 4 {
			setting = 20
		}
	}
	resp[9] = byte(setting)
	length := int16(len(c.Name) + 6)
	resp.SetLength(length)

	resp[8] = byte(len(c.Name))
	resp.Insert([]byte(c.Name), 9)

	member.Socket.Write(resp)

	if member == nil {
		return nil, nil
	}
	member.PartyID = c.UserID
	m := &database.PartyMember{Character: member, Accepted: false}
	party.AddMember(m)

	time.AfterFunc(30*time.Second, func() {
		if !m.Accepted {
			party.RemoveMember(m)
		}
	})

	return nil, nil
}

func (h *RespondPartyRequestHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	accepted := data[6] == 1
	c := s.Character

	party := database.FindParty(c)
	if party == nil {
		return nil, nil
	}

	resp := utils.Packet{}
	if accepted {
		if m := party.GetMember(c.ID); m != nil {
			m.Accepted = true
			m.GroupSettings.ExperienceSharingMethod = party.Leader.GroupSettings.ExperienceSharingMethod
			m.GroupSettings.LootDistriburionMethod = party.Leader.GroupSettings.LootDistriburionMethod
			setting := 33
			if m.GroupSettings.ExperienceSharingMethod == 1 {
				if m.GroupSettings.LootDistriburionMethod == 2 {
					setting = 40
				} else if m.GroupSettings.LootDistriburionMethod == 3 {
					setting = 34
				} else if m.GroupSettings.LootDistriburionMethod == 4 {
					setting = 36
				}
			} else if m.GroupSettings.ExperienceSharingMethod == 2 {
				if m.GroupSettings.LootDistriburionMethod == 1 {
					setting = 17
				} else if m.GroupSettings.LootDistriburionMethod == 2 {
					setting = 24
				} else if m.GroupSettings.LootDistriburionMethod == 3 {
					setting = 18
				} else if m.GroupSettings.LootDistriburionMethod == 4 {
					setting = 20
				}
			}
			rr := PARTY_SETTINGS
			rr[6] = byte(setting)
			m.Socket.Write(rr)
		}

		members := party.GetMembers()
		members = funk.Filter(members, func(m *database.PartyMember) bool {
			return m.Accepted
		}).([]*database.PartyMember)

		r := utils.Packet{0xAA, 0x55, 0x04, 0x00, 0x52, 0x0B, 0x00, 0x55, 0xAA} //0xAA, 0x55, 0x04, 0x00, 0x52, 0x02, 0x0A, 0x00, 0x55, 0xAA
		if len(members) == 1 {
			r.Concat(database.GetPartyMemberData(party.Leader))
			party.Leader.Socket.Write(r)
		}
		//	partyLeaderMessage := r
		party.WelcomeMember(c)                                 // send all party members mine data
		resp.Concat(database.GetPartyMemberData(party.Leader)) // get party leader

		for _, member := range members { // get all party members
			resp.Concat(database.GetPartyMemberData(member.Character))
		}
	} else {
		m := party.GetMember(c.ID)
		party.RemoveMember(m)
		c.PartyID = ""
		if len(party.GetMembers()) == 0 {
			party.Leader.PartyID = ""
		}

		r := PARTY_REQUEST_REJECTED
		party.Leader.Socket.Write(r)
	}

	return resp, nil
}

func (h *LeavePartyHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	c := s.Character
	c.LeaveParty()
	party := database.FindParty(c)
	if party == nil {
		return nil, nil
	}
	for _, m := range party.Members {
		if m.Character.Map == 243 || m.Character.Map == 229 {
			coordinate := &utils.Location{X: 37, Y: 453}
			gomap, _ := m.Character.ChangeMap(17, coordinate)
			m.Character.Socket.Write(gomap)
		}
	}
	return nil, nil
}

func (h *ExpelFromPartyHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	c := s.Character
	party := database.FindParty(c)
	if party == nil || party.Leader.ID != c.ID { // if no party or no authorization
		return nil, nil
	}

	characterID := int(utils.BytesToInt(data[6:10], true))
	character, err := database.FindCharacterByID(characterID)
	if err != nil {
		return nil, err
	} else if characterID == c.ID { // expel yourself
		return nil, nil
	}

	resp := EXPEL_PARTY_MEMBER
	resp.Insert(utils.IntToBytes(uint64(characterID), 4, true), 8) // member character id

	character.Socket.Write(resp)

	member := party.GetMember(characterID)
	member.PartyID = ""
	party.RemoveMember(member)

	members := party.GetMembers()
	members = funk.Filter(members, func(m *database.PartyMember) bool {
		return m.Accepted
	}).([]*database.PartyMember)

	for _, m := range members {
		m.Socket.Write(resp)
	}

	if len(party.GetMembers()) == 0 {
		c.PartyID = ""
		resp.Concat(database.PARTY_DISBANDED)
		party.Delete()
	}

	return resp, nil
}
