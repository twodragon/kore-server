package player

import (
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/twodragon/kore-server/database"
	"github.com/twodragon/kore-server/messaging"
	"github.com/twodragon/kore-server/nats"
	"github.com/twodragon/kore-server/npc"

	"github.com/twodragon/kore-server/server"
	"github.com/twodragon/kore-server/utils"

	"github.com/thoas/go-funk"
)

type ChatHandler struct {
	chatType  int64
	message   string
	receivers map[int]*database.Character
}

type Emotion struct{}

var (
	CHAT_MESSAGE  = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x00, 0x55, 0xAA}
	SHOUT_MESSAGE = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x71, 0x0E, 0x00, 0x00, 0x55, 0xAA}
	ANNOUNCEMENT  = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x71, 0x06, 0x00, 0x55, 0xAA}
)

func (h *Emotion) Handle(s *database.Socket, data []byte) ([]byte, error) {
	emotID := int(utils.BytesToInt(data[11:12], true))
	emotion := database.Emotions[emotID]
	if emotion == nil {
		return nil, nil
	}

	resp := utils.Packet{0xAA, 0x55, 0x09, 0x00, 0x71, 0x09, 0x00, 0x00, 0x55, 0xAA}
	resp.Insert(utils.IntToBytes(uint64(s.Character.PseudoID), 2, true), 6)
	resp.Insert(utils.IntToBytes(uint64(emotion.Type), 1, true), 8)
	resp.Insert(utils.IntToBytes(uint64(emotion.AnimationID), 2, true), 9)

	p := &nats.CastPacket{CastNear: true, CharacterID: s.Character.ID, Data: resp, Type: nats.CHAT_NORMAL}
	err := p.Cast()
	return nil, err
}

func (h *ChatHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	if s.Character == nil {
		return nil, nil
	}

	user, err := database.FindUserByID(s.Character.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}

	stat := s.Stats
	if stat == nil {
		return nil, nil
	}

	h.chatType = utils.BytesToInt(data[4:6], false)

	switch h.chatType {
	case 28929: // normal chat
		messageLen := utils.BytesToInt(data[6:8], true)
		h.message = string(data[8 : messageLen+8])

		return h.normalChat(s)
	case 28930: // private chat
		index := 6
		recNameLength := int(data[index])
		index++

		recName := string(data[index : index+recNameLength])
		index += recNameLength

		c, err := database.FindCharacterByName(recName)
		if err != nil {
			return nil, err
		} else if c == nil {
			return messaging.SystemMessage(messaging.WHISPER_FAILED), nil
		}

		h.receivers = map[int]*database.Character{c.ID: c}

		messageLen := int(utils.BytesToInt(data[index:index+2], true))
		index += 2

		h.message = string(data[index : index+messageLen])
		return h.chatWithReceivers(s, h.createChatMessage)

	case 28931: // party chat
		party := database.FindParty(s.Character)
		if party == nil {
			return nil, nil
		}

		messageLen := int(utils.BytesToInt(data[6:8], true))
		h.message = string(data[8 : messageLen+8])

		members := funk.Filter(party.GetMembers(), func(m *database.PartyMember) bool {
			return m.Accepted
		}).([]*database.PartyMember)
		members = append(members, &database.PartyMember{Character: party.Leader, Accepted: true})

		h.receivers = map[int]*database.Character{}
		for _, m := range members {
			if m.ID == s.Character.ID {
				continue
			}

			h.receivers[m.ID] = m.Character
		}

		return h.chatWithReceivers(s, h.createChatMessage)

	case 28932: // guild chat
		if s.Character.GuildID > 0 {
			guild, err := database.FindGuildByID(s.Character.GuildID)
			if err != nil {
				return nil, err
			}

			members, err := guild.GetMembers()
			if err != nil {
				return nil, err
			}

			messageLen := int(utils.BytesToInt(data[6:8], true))
			h.message = string(data[8 : messageLen+8])
			h.receivers = map[int]*database.Character{}

			for _, m := range members {
				c, err := database.FindCharacterByID(m.ID)
				if err != nil || c == nil || !c.IsOnline || c.ID == s.Character.ID {
					continue
				}

				h.receivers[m.ID] = c
			}

			return h.chatWithReceivers(s, h.createChatMessage)
		}

	case 28933, 28946: // roar chat
		if stat.CHI < 100 || time.Since(s.Character.LastRoar) < 10*time.Second { //time.Now().Sub(s.Character.LastRoar) < 10*time.Second {
			return nil, nil
		}

		s.Character.LastRoar = time.Now()
		characters, err := database.FindCharactersInServer(user.ConnectedServer)
		if err != nil {
			log.Println(err)
			return nil, err
		}

		//delete(characters, s.Character.ID)
		h.receivers = characters

		stat.CHI -= 100

		index := 6
		messageLen := int(utils.BytesToInt(data[index:index+2], true))
		index += 2

		h.message = string(data[index : index+messageLen])

		resp := utils.Packet{}
		_, err = h.chatWithReceivers(s, h.createChatMessage)
		if err != nil {
			log.Println(err)
			return nil, err
		}

		//resp.Concat(chat)
		resp.Concat(s.Character.GetHPandChi())
		return resp, nil

	case 28935: // commands
		index := 6
		messageLen := int(data[index])
		index++

		h.message = string(data[index : index+messageLen])
		return h.cmdMessage(s, data)

	case 28943: // shout
		return h.Shout(s, data)

	case 28945: // faction chat
		characters, err := database.FindCharactersInServer(user.ConnectedServer)
		if err != nil {
			return nil, err
		}

		//delete(characters, s.Character.ID)
		for _, c := range characters {
			if c.Faction != s.Character.Faction {
				delete(characters, c.ID)
			}
		}

		h.receivers = characters
		index := 6
		messageLen := int(utils.BytesToInt(data[index:index+2], true))
		index += 2

		h.message = string(data[index : index+messageLen])
		resp := utils.Packet{}
		_, err = h.chatWithReceivers(s, h.createChatMessage)
		if err != nil {
			return nil, err
		}

		//resp.Concat(chat)
		return resp, nil

	}

	return nil, nil
}

func (h *ChatHandler) Shout(s *database.Socket, data []byte) ([]byte, error) {
	if time.Since(s.Character.LastRoar) < 10*time.Second {
		return nil, nil
	}

	characters, err := database.FindOnlineCharacters()
	if err != nil {
		log.Println(err)
		return nil, err
	}

	slot, _, err := s.Character.FindItemInInventoryByType(nil, database.SHOUT_ARTS_TYPE)
	if err != nil {
		log.Println(err)
		return nil, err
	} else if slot == -1 {
		return nil, nil
	}

	resp := s.Character.DecrementItem(slot, 1)

	index := 6
	messageLen := int(data[index])
	index++

	h.chatType = 28942
	h.receivers = characters
	h.message = string(data[index : index+messageLen])

	h.message = strings.Replace(h.message, "/shout", "", -1)

	_, err = h.chatWithReceivers(s, h.createShoutMessage)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return *resp, nil
}

func (h *ChatHandler) createChatMessage(s *database.Socket) *utils.Packet {

	resp := CHAT_MESSAGE

	index := 4
	resp.Insert(utils.IntToBytes(uint64(h.chatType), 2, false), index) // chat type
	index += 2

	if h.chatType != 28946 {
		resp.Insert(utils.IntToBytes(uint64(s.Character.PseudoID), 2, true), index) // sender character pseudo id
		index += 2
	}

	resp[index] = byte(len(s.Character.Name)) // character name length
	index++

	resp.Insert([]byte(s.Character.Name), index) // character name
	index += len(s.Character.Name)

	resp.Insert(utils.IntToBytes(uint64(len(h.message)), 2, true), index) // message length
	index += 2

	resp.Insert([]byte(h.message), index) // message
	index += len(h.message)

	length := index - 4
	resp.SetLength(int16(length)) // packet length

	return &resp
}

func (h *ChatHandler) createShoutMessage(s *database.Socket) *utils.Packet {

	resp := SHOUT_MESSAGE
	length := len(s.Character.Name) + len(h.message) + 6
	resp.SetLength(int16(length)) // packet length

	index := 4
	resp.Insert(utils.IntToBytes(uint64(h.chatType), 2, false), index) // chat type
	index += 2

	resp[index] = byte(len(s.Character.Name)) // character name length
	index++

	resp.Insert([]byte(s.Character.Name), index) // character name
	index += len(s.Character.Name)

	resp[index] = byte(len(h.message)) // message length
	index++

	resp.Insert([]byte(h.message), index) // message
	return &resp
}

func (h *ChatHandler) normalChat(s *database.Socket) ([]byte, error) {

	if _, ok := server.MutedPlayers.Get(s.User.ID); ok {
		msg := "Chatting with this account is prohibited. Please contact our customer support service for more information."
		return messaging.InfoMessage(msg), nil
	}

	resp := h.createChatMessage(s)
	p := &nats.CastPacket{CastNear: true, CharacterID: s.Character.ID, Data: *resp, Type: nats.CHAT_NORMAL}
	err := p.Cast()

	return nil, err
}

func (h *ChatHandler) chatWithReceivers(s *database.Socket, msgHandler func(*database.Socket) *utils.Packet) ([]byte, error) {

	if _, ok := server.MutedPlayers.Get(s.User.ID); ok {
		msg := "Chatting with this account is prohibited. Please contact our customer support service for more information."
		return messaging.InfoMessage(msg), nil
	}

	resp := msgHandler(s)

	for _, c := range h.receivers {

		if c == nil || !c.IsOnline {
			if h.chatType == 28930 { // PM
				return messaging.SystemMessage(messaging.WHISPER_FAILED), nil
			}
			continue
		} /*
			friends, err := database.FindFriendsByCharacterID(c.ID)
			if err == nil {
				for _, friend := range friends {
					if friend.FriendID == s.Character.ID && friend.IsBlocked {
						msg := "This Player blocked your incoming conversations."
						return messaging.InfoMessage(msg), nil

					}
				}
			}
		*/
		socket := database.GetSocket(c.UserID)
		if socket != nil {
			err := socket.Write(*resp)
			if err != nil {
				log.Println(err)
				return nil, err
			}
		}
	}

	return *resp, nil
}

func makeAnnouncement(msg string) {
	length := int16(len(msg) + 3)

	resp := ANNOUNCEMENT
	resp.SetLength(length)
	resp[6] = byte(len(msg))
	resp.Insert([]byte(msg), 7)

	p := nats.CastPacket{CastNear: false, Data: resp}
	p.Cast()
}

func (h *ChatHandler) cmdMessage(s *database.Socket, data []byte) ([]byte, error) {

	var (
		err  error
		resp utils.Packet
	)

	text := fmt.Sprintf("Name: "+s.Character.Name+"("+s.Character.UserID+") used command: (%s)", h.message)
	utils.NewLog("logs/cmd_logs.txt", text)

	if parts := strings.Split(h.message, " "); len(parts) > 0 {
		cmd := strings.ToLower(strings.TrimPrefix(parts[0], "/"))

		switch cmd {
		case "shout":
			return h.Shout(s, data)

		case "announce":
			if s.User.UserType < server.GAL_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			msg := strings.Join(parts[1:], " ")
			makeAnnouncement(msg)
		case "deleteitemslot":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}
			slotID, err := strconv.ParseInt(parts[1], 10, 16)
			slotMax := int16(slotID)
			if err != nil {
				return nil, err
			}
			ch := s.Character
			if len(parts) >= 3 {
				chr, _ := database.FindCharacterByName(parts[2])
				ch = chr
			}
			r, err := ch.RemoveItem(slotMax)
			if err != nil {
				return nil, err
			}
			ch.Socket.Write(r)
		case "skill":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}
			if len(parts) < 2 {
				return nil, nil
			}
			c := s.Character

			skillID, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}
			c.DoAnimation(int(skillID))
		case "boxinfo":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}
			if len(parts) < 2 {
				return nil, nil
			}
			itemID := int64(utils.StringToInt(parts[1]))
			title, msg, _ := database.GetBoxContentInfo(itemID)
			log.Print(title + msg)
		case "item":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			itemID := int64(utils.StringToInt(parts[1]))

			quantity := int64(1)
			info, ok := database.GetItemInfo(int64(itemID))
			if !ok {
				msg := "Item not found"
				return messaging.InfoMessage(msg), nil
			}
			if info.Timer > 0 {
				quantity = int64(info.Timer)
			}
			if len(parts) >= 3 {
				quantity, err = strconv.ParseInt(parts[2], 10, 64)
				if err != nil {
					return nil, err
				}
			}
			ch := s.Character
			if s.User.UserType >= server.HGM_USER {
				if len(parts) >= 4 {
					chr, err := database.FindCharacterByName(parts[3])
					if err != nil {
						return nil, err
					} else {
						ch = chr
					}
				}
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

			r, _, err := ch.AddItem(item, -1, false)
			if err != nil || r == nil {
				return nil, err
			}
			sItemID := fmt.Sprint(item.ItemID)
			text := "Name: " + s.Character.Name + "(" + s.Character.UserID + ") give item(" + fmt.Sprint(item.ID) + ") ItemID: " + sItemID + " Quantity: " + fmt.Sprint(item.Quantity)
			utils.NewLog("logs/admin_log.txt", text)
			ch.Socket.Write(*r)

		case "attackspeed":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			amount, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}
			s.Stats.AdditionalAttackSpeed = int(amount)

			s.Stats.Update()
			statData, err := s.Character.GetStats()
			if err != nil {
				return nil, err
			}
			resp := utils.Packet{}
			resp.Concat(statData)
			resp.Concat(messaging.InfoMessage(fmt.Sprintf("Attack speed set to %d", amount)))

		case "class":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}

			if len(parts) < 3 {
				return nil, nil
			}

			c, err := database.FindCharacterByName(parts[1])
			if err != nil {
				return nil, err
			}
			if c == nil {
				return nil, nil
			}

			t, err := strconv.Atoi(parts[2])
			if err != nil {
				return nil, err
			}
			c.Class = t
			c.Update()
			//resp := utils.Packet{}
			resp = utils.Packet{0xAA, 0x55, 0x03, 0x00, 0x57, 0x09, 0x00, 0x55, 0xAA}
			resp[6] = byte(c.Class)
			return resp, nil

		case "gold":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			amount, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}

			resp := s.Character.LootGold(uint64(amount))
			s.Write(resp)
		case "god":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}
			if s.Stats.STR >= 99999 && s.Stats.DEX >= 99999 {
				s.Stats.STR -= 99999
				s.Stats.DEX -= 99999
				s.Stats.INT -= 99999
				s.Stats.Fire -= 99999
				s.Stats.Water -= 99999
				s.Stats.Wind -= 99999
			} else {

				s.Stats.STR += 99999
				s.Stats.DEX += 99999
				s.Stats.INT += 99999
				s.Stats.Fire += 99999
				s.Stats.Water += 99999
				s.Stats.Wind += 99999
			}

			s.Stats.Calculate()
			s.Stats.HP = s.Stats.MaxHP
			s.Stats.CHI = s.Stats.MaxCHI

			s.Stats.Update()
			statData, err := s.Character.GetStats()
			if err != nil {
				return nil, err
			}
			resp := utils.Packet{}
			resp.Concat(statData)
		case "shop":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}

			shopNo, err := strconv.ParseInt(parts[1], 10, 32)
			if err != nil {
				return nil, err
			}

			resp = utils.Packet{0xAA, 0x55, 0x07, 0x00, 0x57, 0x03, 0x01, 0x55, 0xAA}
			resp.Insert(utils.IntToBytes(uint64(shopNo), 4, true), 7) // shop id

		case "upgrade":
			if s.User.UserType < server.HGM_USER || len(parts) < 3 {
				return nil, nil
			}

			slots, err := s.Character.InventorySlots()
			if err != nil {
				return nil, err
			}

			slotID, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}

			code, err := strconv.ParseInt(parts[2], 10, 64)
			if err != nil {
				return nil, err
			}

			count := int64(1)
			if len(parts) > 3 {
				count, err = strconv.ParseInt(parts[3], 10, 64)
				if err != nil {
					return nil, err
				}
			}

			codes := []byte{}
			for i := 0; i < int(count); i++ {
				codes = append(codes, byte(code))
			}

			item := slots[slotID]
			return item.Upgrade(int16(slotID), codes...), nil
		case "clearinv":
			if s.User.UserType < server.GAL_USER {
				return nil, nil
			}
			s.Character.ClearInventory()

		case "playerskillreset":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			if len(parts) < 2 {
				return nil, nil
			}
			c, err := database.FindCharacterByName(parts[1])
			if err != nil {
				return nil, err
			}
			if c == nil {
				return nil, nil
			}
			go c.ResetPlayerSkillBook()
		case "alljobsreset":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			chars, _ := database.FindAllCharacter()
			for _, char := range chars {
				char.ResetPlayerSkillBook()
			}

		case "addstatpoints":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			if len(parts) < 3 {
				return nil, nil
			}
			c, err := database.FindCharacterByName(parts[1])
			if err != nil {
				return nil, err
			}
			points, err := strconv.ParseInt(parts[2], 10, 64)
			if err != nil {
				return nil, err
			}
			c.Socket.Stats.StatPoints += int(points)
			spawnData, err := c.SpawnCharacter()
			if err == nil {
				c.Socket.Write(spawnData)
				c.Update()
			}

		case "exp":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			amount, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}

			ch := s.Character
			if len(parts) > 2 {
				c, err := database.FindCharacterByName(parts[2])
				if err != nil {
					return nil, err
				}
				ch = c
			}

			data, levelUp := ch.AddExp(amount)
			if levelUp {
				statData, err := ch.GetStats()
				if err == nil && ch.Socket != nil {
					ch.Socket.Write(statData)
				}
			}

			if ch.Socket != nil {
				ch.Socket.Write(data)
			}

			return nil, nil
		case "petexp":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			amount, err := strconv.ParseUint(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}
			slots, err := s.Character.InventorySlots()
			if err != nil {
				log.Println(err)
				return nil, nil
			}

			petSlot := slots[0x0A]
			pet := petSlot.Pet
			if pet == nil || petSlot.ItemID == 0 || !pet.IsOnline {
				return nil, nil
			}
			pet.AddExp(s.Character, float64(amount))

		case "map":
			if s.User.UserType < server.GA_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			mapID, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}

			if len(parts) >= 3 {
				c, err := database.FindCharacterByName(parts[2])
				if err != nil {
					return nil, err
				}
				if c == nil {
					return nil, nil
				}

				data, err := c.ChangeMap(int16(mapID), nil)
				if err != nil {
					return nil, err
				}

				database.GetSocket(c.UserID).Write(data)
				return nil, nil
			}

			return s.Character.ChangeMap(int16(mapID), nil)
		case "event":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			if len(parts) < 2 {
				return nil, nil
			}
			cmdEvents(parts[1])
		case "eventprob":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			if len(parts) < 2 {
				return nil, nil
			}
			count, _ := strconv.ParseInt(parts[1], 10, 64)
			database.EventProb = int(count)
			s.Write(messaging.InfoMessage(fmt.Sprintf("Succesfully change the new value is %d !", count)))
		case "kill":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			ids, err := s.Character.GetNearbyAIIDs()
			if err != nil {
				log.Println(err)
				return nil, nil
			}

			ai := &database.AI{}
			for _, id := range ids {
				mob := database.AIs[id]
				if int(mob.PseudoID) == s.Character.Selection {
					ai = mob
				}
			}
			if ai == nil {
				return nil, nil
			}
			npcPos := database.GetNPCPosByID(ai.PosID)
			if npcPos == nil {
				log.Println("npc pos is nil")
				return nil, nil
			}
			npc, ok := database.GetNpcInfo(npcPos.NPCID)
			if !ok || npc == nil {
				log.Println("npc is nil")
				return nil, nil
			}
			targetPos := database.GetNPCPosByID(ai.PosID)
			if targetPos == nil {
				return nil, nil
			}

			ai.HP = 1

		case "generate":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			if len(parts) < 3 { //
				return nil, nil
			}
			n, _ := strconv.ParseInt(parts[1], 10, 64)
			m, _ := strconv.ParseInt(parts[2], 10, 64)
			generateMobs(int(n), int(m))
		case "del":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			chars, err := database.FindAllCharacter()
			if err != nil {
				log.Print(err)
			}
			for _, char := range chars {
				user, err := database.FindUserByID(char.UserID)
				if err != nil {
					log.Print(err)
				} else if user == nil {
					char.Delete()
				}
			}
		case "delcons":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
		case "addmobs":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			npcId, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}
			count, err := strconv.ParseInt(parts[2], 10, 64)
			if err != nil {
				return nil, err
			}
			mapID := s.Character.Map
			server := s.User.ConnectedServer
			NPCsSpawnPoint := []string{s.Character.Coordinate}
			cmdSpawnMobs(int(count), int(npcId), int(server), int(mapID), NPCsSpawnPoint)

		case "cash":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			if len(parts) < 3 {
				return nil, nil
			}

			amount, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}

			c, err := database.FindCharacterByName(parts[2])
			if err != nil {
				return nil, err
			}
			if c == nil {
				return nil, nil
			}
			user, err := database.FindUserByID(c.UserID)
			if err != nil {
				return nil, err
			} else if user == nil {
				return nil, nil
			}
			user.NCash += uint64(amount)
			user.Update()
			text := fmt.Sprintf("Name: "+s.Character.Name+"("+s.Character.UserID+") give cash(%d) To: "+c.Name, amount)
			utils.NewLog("logs/admin_log.txt", text)

			return messaging.InfoMessage(fmt.Sprintf("%d nCash loaded to %s (%s).", amount, user.Username, user.ID)), nil

			/*	case "transfer":
				if len(parts) < 3 {
					return nil, nil
				}

				amount, err := strconv.ParseInt(parts[1], 10, 64)
				if err != nil {
					return nil, err
				}
				if amount <= 0 {
					return nil, nil
				}
				if s.User.NCash < uint64(amount) {
					return messaging.InfoMessage(fmt.Sprintf("Not enough ncash to transfer.")), nil
				}
				c, err := database.FindCharacterByName(parts[2])
				if err != nil {
					return nil, err
				}
				if c == nil {
					return nil, nil
				}
				user, err := database.FindUserByID(c.UserID)
				if err != nil {
					return nil, err
				} else if user == nil {
					return nil, nil
				}
				s.User.NCash -= uint64(amount)
				s.User.Update()
				user.NCash += uint64(amount)
				user.Update()
				text := fmt.Sprintf("Name: "+s.Character.Name+"("+s.Character.UserID+") transfered cash(%d) To: "+c.Name, amount)
				utils.NewLog("logs/nc_transfers.txt", text)

				return messaging.InfoMessage(fmt.Sprintf("%d nCash transfered to %s.", amount, c.Name)), nil*/

		case "goldrate":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}

			if len(parts) < 3 {
				return nil, nil
			}
			if len(parts) > 2 {
				if am, err := strconv.ParseFloat(parts[1], 64); err == nil {
					database.GOLD_RATE = am
				}
				minute, err := strconv.ParseInt(parts[2], 10, 64)
				if err != nil {
					return nil, err
				}
				database.GOLD_RATE_TIME = minute * 60
				//every second check if gold rate time is 0, if so, set gold rate to default
				go func() {
					for {
						database.GOLD_RATE_TIME--
						if database.GOLD_RATE_TIME <= 0 {
							database.GOLD_RATE = database.DEFAULT_GOLD_RATE
							break
						}
						time.Sleep(time.Second)
					}
				}()
				s.Character.ShowEventsDetails()
			}
			return messaging.InfoMessage(fmt.Sprintf("Gold Rate now: %f, Default is : %f", database.GOLD_RATE, database.DEFAULT_GOLD_RATE)), nil
		case "exprate":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}

			if len(parts) < 3 {
				return nil, nil
			}
			if len(parts) > 2 {
				if am, err := strconv.ParseFloat(parts[1], 64); err == nil {
					database.EXP_RATE = am
				}
				minute, err := strconv.ParseInt(parts[2], 10, 64)
				if err != nil {
					return nil, err
				}
				database.EXP_RATE_TIME = minute * 60
				go func() {
					for {
						database.EXP_RATE_TIME--
						if database.EXP_RATE_TIME <= 0 {
							database.EXP_RATE = database.DEFAULT_EXP_RATE
							break
						}
						time.Sleep(time.Second)
					}
				}()
				s.Character.ShowEventsDetails()
			}
			return messaging.InfoMessage(fmt.Sprintf("EXP Rate now: %f", database.EXP_RATE)), nil

		case "relicdrops":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			if len(parts) < 2 {
				return nil, nil
			}
			tru, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}
			if tru == 1 {
				database.RELIC_DROP_ENABLED = true
				s.Write(messaging.InfoMessage("Relic drops enabled."))
			} else if tru == 0 {
				database.RELIC_DROP_ENABLED = false
				s.Write(messaging.InfoMessage("Relic drops disabled."))
			}
		case "droprate":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}

			if len(parts) < 3 {
				return nil, nil
			}
			if len(parts) > 2 {
				if s, err := strconv.ParseFloat(parts[1], 64); err == nil {
					database.DROP_RATE = s
				}
				minute, err := strconv.ParseInt(parts[2], 10, 64)
				if err != nil {
					return nil, err
				}
				database.DROP_RATE_TIME = minute * 60
				go func() {
					for {
						database.DROP_RATE_TIME--
						if database.DROP_RATE_TIME <= 0 {
							database.DROP_RATE = database.DEFAULT_DROP_RATE
							break
						}
						time.Sleep(time.Second)
					}
				}()
				s.Character.ShowEventsDetails()
			}
			return messaging.InfoMessage(fmt.Sprintf("Drop Rate now: %f", database.DROP_RATE)), nil

		case "honor":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			if len(parts) < 2 {
				return nil, nil
			}
			ch := s.Character
			if len(parts) >= 3 {
				c, err := database.FindCharacterByName(parts[1])
				if err != nil {
					return nil, err
				}
				if c == nil {
					return nil, nil
				}
				ch = c
			}
			honorpoints, err := strconv.ParseInt(parts[2], 10, 32)
			if err != nil {
				return nil, err
			}
			ch.Socket.Stats.Honor = int(honorpoints)
			ch.Socket.Stats.Update()
		case "debug":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}
			if len(parts) < 2 {
				return nil, nil
			}
			dbg, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}
			database.DEBUG_FACTORY = int(dbg)
		case "devlog":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}
			if len(parts) < 2 {
				return nil, nil
			}
			log, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}
			database.DEVLOG = int(log)
		case "rank":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			rankID, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}
			s.Character.HonorRank = rankID
			s.Character.Update()
			resp := database.CHANGE_RANK
			resp.Insert(utils.IntToBytes(uint64(s.Character.PseudoID), 2, true), 6)
			resp.Insert(utils.IntToBytes(uint64(s.Character.HonorRank), 4, true), 8)
			statData, _ := s.Character.GetStats()
			resp.Concat(statData)
			s.Write(resp)

		case "addguild":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			if len(parts) < 2 {
				return nil, nil
			}
			guildid, err := strconv.ParseInt(parts[1], 10, 32)
			if err != nil {
				return nil, err
			}
			ch := s.Character
			if len(parts) >= 3 {
				c, err := database.FindCharacterByName(parts[2])
				if err != nil {
					return nil, err
				}
				if c == nil {
					return nil, nil
				}
				ch = c
			}
			removeid := -1
			if int(guildid) == removeid {
				guild, err := database.FindGuildByID(ch.GuildID)
				if err != nil {
					return nil, err
				}
				err = guild.RemoveMember(ch.ID)
				if err != nil {
					return nil, err
				}
				guild.Update()
				ch.GuildID = int(guildid)
			} else {
				guild, err := database.FindGuildByID(int(guildid))
				if err != nil {
					return nil, err
				}
				guild.AddMember(&database.GuildMember{ID: ch.ID, Role: database.GROLE_MEMBER})
				guild.Update()
				ch.GuildID = int(guildid)
			}
			spawnData, err := s.Character.SpawnCharacter()
			if err == nil {
				p := nats.CastPacket{CastNear: true, CharacterID: s.Character.ID, Type: nats.PLAYER_SPAWN, Data: spawnData}
				p.Cast()
				ch.Socket.Write(spawnData)
			}
			resp.Concat(messaging.InfoMessage(fmt.Sprintf("Player new guild id: %d", ch.GuildID)))
			return resp, nil
		case "war":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			time, _ := strconv.ParseInt(parts[1], 10, 32)
			database.PrepareFactionWar(int(time))
		case "fgwar":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			time, _ := strconv.ParseInt(parts[1], 10, 32)
			database.PrepareFlagKingdom(int(time))
		case "gwar":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			if len(parts) < 2 {
				return nil, nil
			}
			time, _ := strconv.ParseInt(parts[1], 10, 32)
			database.CanJoinWar = true
			database.StartWarTimer(int(time), 40)
		case "divine":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			if len(parts) < 2 {
				return nil, nil
			}
			ch, err := database.FindCharacterByName(parts[1])
			if err != nil {
				return nil, err
			}

			data, levelUp := ch.AddExp(233332051411)
			if levelUp {
				statData, err := ch.GetStats()
				if err == nil && ch.Socket != nil {
					ch.Socket.Write(statData)
				}
			}
			if ch.Socket != nil {
				ch.Socket.Write(data)
			}

			ch.GoDivine()
		case "reborn":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			if len(parts) < 2 {
				return nil, nil
			}
			ch, err := database.FindCharacterByName(parts[1])
			if err != nil {
				return nil, err
			}

			data, levelUp := ch.AddExp(233332051411)
			if levelUp {
				statData, err := ch.GetStats()
				if err == nil && ch.Socket != nil {
					ch.Socket.Write(statData)
				}
			}
			if ch.Socket != nil {
				ch.Socket.Write(data)
			}

			ch.Reborn()

		case "dungeon":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			data, err := s.Character.ChangeMap(243, nil)
			if err != nil {
				return nil, err
			}
			resp.Concat(data)
			x := "377,246"
			coord := s.Character.Teleport(database.ConvertPointToLocation(x))
			resp.Concat(coord)
			return resp, nil
		case "verifyconsignment":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			database.IsHideBannedUserItems = true

		case "refresh":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			if len(parts) < 2 {
				return nil, nil
			}
			command := parts[1]
			switch command {
			case "relics":
				database.GetRelics()
			case "savepoints":
				database.GetAllSavePoints()
			case "scripts":
				database.RefreshScripts()
			case "maps":
			//	database.GetMaps()
			case "htshop":
				database.GetHTItems()
			case "items":
				database.GetAllItems()
			case "buffinf":
				database.GetBuffInfections()
			case "advancedfusions":
				database.GetAdvancedFusions()
			case "gamblings":
				database.GetGamblings()
			case "craftitems":
				database.GetCraftItem()
			case "productions":
				database.GetProductions()
			case "drops":
				database.ReadAllDropsInfo()
			case "shopitems":
				database.GetShopItems()
			case "itemsets":
				database.ReadItemSets()
			case "shoptable":
				database.GetShopsTable()
			case "exp":
				database.GetExps()
			case "skills":
				database.GetSkills()
			case "npcs":
				database.ReadAllNPCsInfo()
			case "gates":
				database.GetGates()
			case "houses":
				database.GetHouseItems()
			case "all":
				callBacks := []func() error{database.RefreshScripts, database.GetGates, database.GetShopsTable,
					database.GetGamblings, database.ReadAllNPCsInfo, database.GetExps}
				for _, cb := range callBacks {
					if err := cb(); err != nil {
						fmt.Println("Error: ", err)
					}
				}
			}

		case "cleanitemsdb":
			allitems := database.ReadAllItems()

			allusers := database.Users
			for _, item := range allitems {
				if allusers[item.UserID.String] == nil {
					item.Delete()
				}

			}

		case "startci":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			database.StartCiEventCountdown(600)
		case "setinjury":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			if len(parts) < 3 {
				return nil, nil
			}
			ch, err := database.FindCharacterByName(parts[1])
			if err != nil {
				return nil, err
			}
			if ch == nil {
				return messaging.InfoMessage("Character not found"), nil
			}
			injury, _ := strconv.ParseFloat(parts[2], 64)
			ch.Injury = injury
			ch.Update()

		case "checkin":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			c, err := database.FindCharacterByName(parts[1])
			if err != nil {
				return nil, err
			}
			checknum, err := strconv.ParseInt(parts[2], 10, 64)
			if err != nil {
				return nil, err
			}
			user, err := database.FindUserByID(c.UserID)
			if err == nil {
				user.CheckinCounter = int(checknum)
			}

		case "discitem":

			if s.User.UserType < server.GM_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			itemID := int64(utils.StringToInt(parts[1]))

			quantity := int64(1)
			itemtype := int64(0)
			judgestat := int64(0)
			info, ok := database.GetItemInfo(int64(itemID))
			if !ok {
				msg := "Item not found"
				return messaging.InfoMessage(msg), nil
			}
			if info.Timer > 0 {
				quantity = int64(info.Timer)
			}
			if len(parts) >= 3 {
				quantity, err = strconv.ParseInt(parts[2], 10, 64)
				if err != nil {
					return nil, err
				}
			}
			if len(parts) >= 4 {
				itemtype, err = strconv.ParseInt(parts[3], 10, 64)
				if err != nil {
					return nil, err
				}
			}
			if len(parts) >= 5 {
				judgestat, err = strconv.ParseInt(parts[4], 10, 64)
				if err != nil {
					return nil, err
				}
			}
			ch := s.Character
			if s.User.UserType >= server.HGM_USER {
				if len(parts) >= 6 {
					chID, err := strconv.ParseInt(parts[5], 10, 64)
					if err == nil {
						chr, err := database.FindCharacterByID(int(chID))
						if err == nil {
							ch = chr
						}
					}
				}
			}

			item := &database.InventorySlot{ItemID: int64(itemID), Quantity: uint(quantity), ItemType: int16(itemtype), JudgementStat: int64(judgestat)}

			if info.GetType() == database.PET_TYPE {
				petInfo := database.Pets[itemID]
				petExpInfo := database.PetExps[petInfo.Level-1]
				targetExps := []int{petExpInfo.ReqExpEvo1, petExpInfo.ReqExpEvo2, petExpInfo.ReqExpEvo3, petExpInfo.ReqExpHt, petExpInfo.ReqExpDivEvo1, petExpInfo.ReqExpDivEvo2, petExpInfo.ReqExpDivEvo3, petExpInfo.ReqExpDivEvo4, petExpInfo.ReqExpDivEvo4}
				item.Pet = &database.PetSlot{
					Fullness: 100, Loyalty: 100,
					Exp:   float64(targetExps[petInfo.Evolution-1]),
					HP:    petInfo.BaseHP,
					Level: byte(petInfo.Level),
					Name:  "",
					CHI:   petInfo.BaseChi}
			}

			r, _, err := ch.AddItem(item, -1, false)
			if err != nil {
				return nil, err
			}
			//sItemID := fmt.Sprint(item.ItemID)
			//text := "Name: " + s.Character.Name + "(" + s.Character.UserID + ") give item(" + fmt.Sprint(item.ID) + ") ItemID: " + sItemID + " Quantity: " + fmt.Sprint(item.Quantity)
			//adminlogger.Println(text)
			ch.Socket.Write(*r)
			return nil, nil

		case "buff":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}
			if len(parts) < 3 {
				return nil, nil
			}
			infID, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}
			duration, err := strconv.ParseInt(parts[2], 10, 64)
			if err != nil {
				return nil, err
			}
			infection := database.BuffInfections[int(infID)]
			s.Character.AddBuff(infection, duration)

		case "r":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			if len(parts) < 2 {
				return nil, nil
			}
			data, err := hex.DecodeString(parts[1])
			if err != nil {
				return nil, nil
			}
			log.Print(data)
			s.Character.Socket.Write(data)

		case "mob":
			if s.User.UserType < server.GAL_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			posId, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}

			npcPos := database.GetNPCPosByID(int(posId))
			if npcPos == nil {
				return nil, nil
			}
			npc, ok := database.GetNpcInfo(npcPos.NPCID)
			if !ok {
				return nil, nil
			}

			ai := &database.AI{ID: len(database.AIs), HP: npc.MaxHp, Map: npcPos.MapID, PosID: npcPos.ID, RunningSpeed: 10, Server: 1, WalkingSpeed: 5, Once: true, Faction: 0, CanAttack: true}
			database.GenerateIDForAI(ai)
			ai.OnSightPlayers = make(map[int]interface{})

			minLoc := database.ConvertPointToLocation(npcPos.MinLocation)
			maxLoc := database.ConvertPointToLocation(npcPos.MaxLocation)
			loc := utils.Location{X: utils.RandFloat(minLoc.X, maxLoc.X), Y: utils.RandFloat(minLoc.Y, maxLoc.Y)}
			ai.NPCpos = npcPos
			ai.Coordinate = loc.String()
			ai.Handler = ai.AIHandler
			go ai.Handler()

			makeAnnouncement(fmt.Sprintf("%s is roaring. %s", npc.Name, ai.Coordinate))

			database.AIsByMap[ai.Server][npcPos.MapID] = append(database.AIsByMap[ai.Server][npcPos.MapID], ai)
			database.AIs[ai.ID] = ai

		case "spawn":
			posId, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}
			npcPos := database.GetNPCPosByID(int(posId))
			if npcPos == nil {
				return nil, nil
			}
			database.SpawnCreep(2, npcPos)
		case "relic":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			itemID := utils.StringToInt(parts[1])

			ch := s.Character
			if len(parts) >= 3 {
				if err == nil {
					chr, err := database.FindCharacterByName(parts[2])
					if err != nil || chr == nil {
						return nil, nil
					}
					ch = chr
				}
			}

			slot, err := ch.FindFreeSlot()
			if err != nil {
				return nil, nil
			}

			itemtype := int(0)
			if len(parts) >= 4 {
				if err == nil {
					itemtype = utils.StringToInt(parts[3])
					if itemtype < 0 || itemtype > 2 {
						itemtype = 0
					}
				}
			}

			itemData, _, _ := ch.AddItem(&database.InventorySlot{ItemID: int64(itemID), Quantity: 1, ItemType: int16(itemtype), JudgementStat: int64(0)}, slot, true)
			if itemData != nil {
				ch.Socket.Write(*itemData)
				relicDrop := ch.RelicDrop(int64(itemID))
				p := nats.CastPacket{CastNear: false, Data: relicDrop, Type: nats.ITEM_DROP}
				p.Cast()
			}

		case "main":
			if s.User.UserType < server.GAL_USER {
				return nil, nil
			}

			countMaintenance(60)

		case "ban":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}

			if len(parts) < 3 {
				return nil, nil
			}

			userID := parts[1]
			user, err := database.FindUserByID(userID)
			if err != nil {
				return nil, err
			} else if user == nil {
				return nil, nil
			}

			hours, err := strconv.ParseInt(parts[2], 10, 64)
			if err != nil {
				return nil, err
			}

			user.UserType = 0
			user.DisabledUntil = time.Now().Add(time.Hour * time.Duration(hours)).Format("2006-01-02 15:04:05")
			user.Update()

			database.GetSocket(userID).Conn.Close()

		case "mute":
			if s.User.UserType < server.GA_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			dumb, err := database.FindCharacterByName(parts[1])
			if err != nil {
				return nil, err
			}

			server.MutedPlayers.Set(dumb.UserID, struct{}{})

		case "unmute":
			if s.User.UserType < server.GA_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			dumb, err := database.FindCharacterByName(parts[1])
			if err != nil {
				return nil, err
			}

			server.MutedPlayers.Remove(dumb.UserID)

		case "uid":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			c, err := database.FindCharacterByName(parts[1])
			if err != nil {
				return nil, err
			} else if c == nil {
				return nil, nil
			}

			resp = messaging.InfoMessage(c.UserID)

		case "uuid":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			c, err := database.FindCharacterByName(parts[1])
			if err != nil {
				return nil, err
			} else if c == nil {
				return nil, nil
			}

			resp = messaging.InfoMessage(string(rune(c.ID)))

		case "visibility":
			if s.User.UserType < server.GA_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}
			if parts[1] == "1" {
				data := database.BUFF_INFECTION
				data.Insert(utils.IntToBytes(uint64(70), 4, true), 6)     // infection id
				data.Insert(utils.IntToBytes(uint64(99999), 4, true), 11) // buff remaining time

				s.Write(data)
			} else {
				r := database.BUFF_EXPIRED
				r.Insert(utils.IntToBytes(uint64(70), 4, true), 6) // buff infection id
				r.Concat(data)

				s.Write(r)
			}
			s.Character.Invisible = parts[1] == "1"

		case "kick":
			if s.User.UserType < server.GAL_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			dumb, err := database.FindCharacterByName(parts[1])
			if err != nil {
				return nil, nil
			}

			database.GetSocket(dumb.UserID).Conn.Close()

		case "tp":
			if s.User.UserType < server.GA_USER {
				return nil, nil
			}

			if len(parts) < 3 {
				return nil, nil
			}

			x, err := strconv.ParseFloat(parts[1], 32)
			if err != nil {
				return nil, err
			}

			y, err := strconv.ParseFloat(parts[2], 32)
			if err != nil {
				return nil, err
			}

			return s.Character.Teleport(database.ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", x, y))), nil

		case "tpp":
			if s.User.UserType < server.GAL_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			c, err := database.FindCharacterByName(parts[1])
			if err != nil {
				return nil, err
			}
			if c.Socket.User.ConnectedServer != s.User.ConnectedServer {
				s.User.ConnectedServer = c.Socket.User.ConnectedServer
				s.User.SelectedServerID = c.Socket.User.ConnectedServer
			}
			mapID, _ := s.Character.ChangeMap(c.Map, database.ConvertPointToLocation(c.Coordinate))
			s.Write(mapID)

			return nil, nil

		case "summon":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			c, err := database.FindCharacterByName(parts[1])
			if err != nil {
				return nil, err
			}
			if c == nil {
				return nil, nil
			}

			coordinate := database.ConvertPointToLocation(s.Character.Coordinate)
			gomap, _ := c.ChangeMap(s.Character.Map, coordinate)
			c.Socket.Write(gomap)
			s.Write(gomap)

		case "speed":
			if s.User.UserType < server.GAL_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			speed, err := strconv.ParseFloat(parts[1], 32)
			if err != nil {
				return nil, err
			}

			s.Character.RunningSpeed = speed
			statdata, err := s.Character.GetStats()
			if err != nil {
				return nil, err
			}
			s.Write(statdata)

		case "online":
			if s.User.UserType < server.GA_USER {
				return nil, nil
			}

			characters, err := database.FindOnlineCharacters()
			if err != nil {
				return nil, err
			}

			online := funk.Values(characters).([]*database.Character)
			sort.Slice(online, func(i, j int) bool {
				return online[i].Name < online[j].Name
			})

			resp.Concat(messaging.InfoMessage(fmt.Sprintf("%d player(s) online.", len(characters))))

			for _, c := range online {
				u, _ := database.FindUserByID(c.UserID)
				if u == nil {
					continue
				}

				resp.Concat(messaging.InfoMessage(fmt.Sprintf("%s is in map %d (Dragon%d) at %s.", c.Name, c.Map, u.ConnectedServer, c.Coordinate)))
			}
		case "resetdungkeys":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			err := database.RefreshYingYangKeys()
			if err != nil {
				fmt.Print(err)
			}

		case "giveitemtoall":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			if len(parts) < 2 {
				return nil, nil
			}

			characters, err := database.FindOnlineCharacters()
			if err != nil {
				return nil, err
			}

			online := funk.Values(characters).([]*database.Character)
			sort.Slice(online, func(i, j int) bool {
				return online[i].Name < online[j].Name
			})

			itemID := int64(utils.StringToInt(parts[1]))

			quantity := int64(1)
			info, ok := database.GetItemInfo(int64(itemID))
			if !ok || info == nil {
				return nil, nil
			}
			if info.Timer > 0 {
				quantity = int64(info.Timer)
			}
			if len(parts) >= 3 {
				quantity, err = strconv.ParseInt(parts[2], 10, 64)
				if err != nil {
					return nil, err
				}
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

		case "solve":
			if len(parts) < 1 {
				return nil, nil
			}
			resolveOverlappingItems(parts[1])

		case "name":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}

			if len(parts) < 3 {
				return nil, nil
			}

			id, _ := strconv.Atoi(parts[1])
			c, _ := database.FindCharacterByID(int(id))
			if c == nil {
				return nil, nil
			}

			c2, _ := database.FindCharacterByName(parts[2])
			if c2 != nil {
				return nil, nil
			}

			c.Name = parts[2]
			c.Update()

		case "role":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}

			if len(parts) < 3 {
				return nil, nil
			}

			id, _ := strconv.Atoi(parts[1])
			c, err := database.FindCharacterByID(int(id))
			if err != nil {
				return nil, err
			}

			user, err := database.FindUserByID(c.UserID)
			if err != nil {
				return nil, err
			}

			role, err := strconv.Atoi(parts[2])
			if err != nil {
				return nil, err
			}

			user.UserType = int8(role)
			user.Update()
		case "skillpoint":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}

			if len(parts) < 3 {
				return nil, nil
			}

			character, err := database.FindCharacterByName(parts[1])
			if err != nil {
				return nil, err
			}
			num, err := strconv.Atoi(parts[2])
			if err != nil {
				return nil, err
			}
			character.Socket.Skills.SkillPoints += num
			if character.Socket != nil {

				character.Socket.Write(character.GetExpAndSkillPts())
			}
		case "resetmobs":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}

			posId, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}
			npcPos := database.GetNPCPosByID(int(posId))
			if npcPos == nil {
				return nil, nil
			}
			npc, ok := database.GetNpcInfo(npcPos.NPCID)
			if !ok {
				log.Print("Error")
			}
			for i := 0; i < int(npcPos.Count); i++ {
				if npc.ID == 0 {
					continue
				}

				newai := &database.AI{ID: len(database.AIs), HP: npc.MaxHp, Map: npcPos.MapID, PosID: npcPos.ID, RunningSpeed: 10, Server: 1, WalkingSpeed: 5, Once: true}
				database.GenerateIDForAI(newai)
				newai.OnSightPlayers = make(map[int]interface{})

				minLoc := database.ConvertPointToLocation(npcPos.MinLocation)
				maxLoc := database.ConvertPointToLocation(npcPos.MaxLocation)
				loc := utils.Location{X: utils.RandFloat(minLoc.X, maxLoc.X), Y: utils.RandFloat(minLoc.Y, maxLoc.Y)}
				newai.Coordinate = loc.String()
				log.Print(newai.Coordinate)
				newai.Handler = newai.AIHandler
				database.AIsByMap[newai.Server][npcPos.MapID] = append(database.AIsByMap[newai.Server][npcPos.MapID], newai)
				database.AIs[newai.ID] = newai
				log.Print("New mob created", len(database.AIs))
				newai.Create()
				go newai.Handler()
			}
			log.Print("Finished")
			return nil, nil
		case "resetallmobs":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}

			for _, npcPos := range database.GetNPCPostions() {
				npc, ok := database.GetNpcInfo(npcPos.NPCID)
				if !ok {
					log.Print("Error")
					continue
				}

				for i := 0; i < int(npc.Test); i++ {
					if npc.ID == 0 || npcPos.IsNPC || !ok || !npcPos.Attackable {
						continue
					}
					minLoc := database.ConvertPointToLocation(npcPos.MinLocation)
					maxLoc := database.ConvertPointToLocation(npcPos.MaxLocation)
					loc := utils.Location{X: utils.RandFloat(minLoc.X, maxLoc.X), Y: utils.RandFloat(minLoc.Y, maxLoc.Y)}
					//newai.Coordinate = loc.String()
					x := len(database.AIs)
					x++
					newai := &database.AI{
						ID: x,
						//	HP:           npc.MaxHp,
						Map:   npcPos.MapID,
						PosID: npcPos.ID,
						//	RunningSpeed: float64(npc.RunningSpeed),
						Server: 1,
						//	WalkingSpeed: float64(npc.WalkingSpeed),
						//	Once:         false,
						CanAttack:  true,
						Faction:    0,
						IsDead:     false,
						Coordinate: loc.String(),
					}

					//	newai.OnSightPlayers = make(map[int]interface{})
					if err := newai.Create(); err != nil {
						fmt.Printf(err.Error())
					}
					newai.Handler = newai.AIHandler
					database.AIsByMap[newai.Server][npcPos.MapID] = append(database.AIsByMap[newai.Server][npcPos.MapID], newai)
					database.AIs[newai.ID] = newai
					fmt.Println("New mob created", newai.ID)

				}

			}
			log.Print("Finished")
			return nil, nil
		case "spawnmob":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}
			posId, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}

			npcPos := database.GetNPCPosByID(int(posId))
			if npcPos == nil {
				return nil, nil
			}
			npc, ok := database.GetNpcInfo(npcPos.NPCID)
			if !ok {
				return nil, nil
			}
			//npcPos := &database.NpcPosition{ID: len(database.NPCPos), NPCID: int(action), MapID: s.Character.Map, Rotation: 0, Attackable: true, IsNPC: false, RespawnTime: 30, Count: 30, MinLocation: "120,120", MaxLocation: "150,150"}
			//newPos := database.NPCPos[int(action)]
			database.SetNPCPos(npcPos.ID, npcPos)
			//npcPos := database.NPCPos[int(action)]
			newai := &database.AI{ID: len(database.AIs), HP: npc.MaxHp, Map: npcPos.MapID, PosID: npcPos.ID, RunningSpeed: 10, Server: 1, WalkingSpeed: 5, Once: true}
			newai.OnSightPlayers = make(map[int]interface{})
			coordinate := database.ConvertPointToLocation(s.Character.Coordinate)
			randomLocX := randFloats(coordinate.X, coordinate.X+30)
			randomLocY := randFloats(coordinate.Y, coordinate.Y+30)
			loc := utils.Location{X: randomLocX, Y: randomLocY}
			npcPos.MinLocation = fmt.Sprintf("%.1f,%.1f", randomLocX, randomLocY)
			maxX := randomLocX + 50
			maxY := randomLocY + 50
			npcPos.MaxLocation = fmt.Sprintf("%.1f,%.1f", maxX, maxY)
			newai.Coordinate = loc.String()
			log.Print(newai.Coordinate)
			newai.Handler = newai.AIHandler

			database.AIsByMap[newai.Server][newai.Map] = append(database.AIsByMap[newai.Server][newai.Map], newai)
			database.AIs[newai.ID] = newai
			database.GenerateIDForAI(newai)
			//ai.Init()
			if newai.WalkingSpeed > 0 {
				go newai.Handler()
			}
		case "form":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			if len(parts) < 1 {
				return nil, nil
			}
			npcid, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, err
			}

			c := s.Character

			if npcid > 0 {
				c.Morphed = true

				c.MorphedNPCID = int(npcid)
				r := utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x37, 0x55, 0xAA}
				r.Insert(utils.IntToBytes(uint64(npcid), 4, true), 5) // form npc id
				data, err := c.GetStats()
				if err == nil {
					r.Concat(data)
				}
				c.Socket.Write(r)
				characters, err := c.GetNearbyCharacters()
				if err != nil {
					log.Println(err)
					return nil, nil
				}
				//test := utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x21, 0x02, 0x00, 0x55, 0xAA}
				for _, chars := range characters {
					delete(chars.OnSight.Players, c.ID)
				}
			} else {
				c.Morphed = false
				c.MorphedNPCID = 0
				FORM_DEACTIVATED := utils.Packet{0xAA, 0x55, 0x01, 0x00, 0x38, 0x55, 0xAA}
				c.Socket.Write(FORM_DEACTIVATED)
				characters, err := c.GetNearbyCharacters()
				if err != nil {
					log.Println(err)
					//return
				}
				//test := utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x21, 0x02, 0x00, 0x55, 0xAA}
				for _, chars := range characters {
					delete(chars.OnSight.Players, c.ID)
				}
			}

		case "npc":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}
			npcID, _ := strconv.Atoi(parts[1])
			actID, _ := strconv.Atoi(parts[2])
			resp := npc.GetNPCMenu(npcID, 999993, 0, []int{actID})
			return resp, nil

			/*		case "loto":
					if len(parts) < 2 {
						return messaging.InfoMessage(fmt.Sprintf("You have to choose a number. Ex: /loto 34")), nil
					}
					number, err := strconv.ParseInt(parts[1], 10, 64)
					if err != nil {
						return nil, err
					}
					database.AddPlayer(s, int(number))*/
		case "info":
			if s.User.UserType < server.GM_USER {
				return nil, nil
			}

			if len(parts) < 2 {
				return nil, nil
			}
			c, err := database.FindCharacterByName(parts[1])
			if err != nil {
				return nil, err
			}

			resp.Concat(messaging.InfoMessage(fmt.Sprintf("%s player details:", c.Name)))
			resp.Concat(messaging.InfoMessage(fmt.Sprintf("CharID: %d | UserName: %s", c.Socket.Character.ID, c.Socket.User.Username)))
			resp.Concat(messaging.InfoMessage(fmt.Sprintf("Map: %d | Location: %s", c.Map, c.Coordinate)))
			resp.Concat(messaging.InfoMessage(fmt.Sprintf("Level: %d | Exp: %d", c.Level, c.Exp)))
			resp.Concat(messaging.InfoMessage(fmt.Sprint("Gold: ", c.Gold)))
			resp.Concat(messaging.InfoMessage(fmt.Sprint("Bank Gold: ", c.Socket.User.BankGold)))
			resp.Concat(messaging.InfoMessage(fmt.Sprint("Ncash: ", c.Socket.User.NCash)))
			resp.Concat(messaging.InfoMessage(fmt.Sprintf("AID: %d | AID-:%t", c.AidTime, c.AidMode)))
			resp.Concat(messaging.InfoMessage(fmt.Sprint("SkillPoints: ", c.Socket.Skills.SkillPoints)))

		case "type":
			if s.User.UserType < server.HGM_USER {
				return nil, nil
			}

			if len(parts) < 3 {
				return nil, nil
			}

			id, _ := strconv.Atoi(parts[1])
			c, err := database.FindCharacterByID(int(id))
			if err != nil {
				return nil, err
			}

			t, err := strconv.Atoi(parts[2])
			if err != nil {
				return nil, err
			}
			c.Type = t
			c.Update()

		}

	}

	return resp, err
}

func countMaintenance(cd int) {
	msg := fmt.Sprintf("There will be maintenance after %d seconds. Please log out in order to prevent any inconvenience.", cd)
	makeAnnouncement(msg)

	if cd > 0 {
		time.AfterFunc(time.Second*10, func() {
			countMaintenance(cd - 10)
		})
	} else {
		characters, err := database.FindOnlineCharacters()
		if err == nil {
			for _, char := range characters {
				char.Update()
				if char.Socket != nil {
					char.Socket.OnClose()
				}
			}
		}
		os.Exit(0)
	}
}

/*func startGuildWar(sourceG, enemyG *database.Guild) []byte {
	challengerGuild := sourceG
	enemyGuild := enemyG
	makeAnnouncement(fmt.Sprintf("%s has declare war to %s.", challengerGuild.Name, enemyGuild.Name))

	return nil
}*/

func randFloats(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

func RemoveIndex(a []string, index int) []string {
	a[index] = a[len(a)-1] // Copy last element to index i.
	a[len(a)-1] = ""       // Erase last element (write zero value).
	a = a[:len(a)-1]       // Truncate slice.
	return a
}

func resolveOverlappingItems(userid string) { //67-306

	bankSlots, _ := database.FindBankSlotsByUserID(userid)
	freeSlots := make(map[int16]struct{})
	for _, s := range bankSlots {
		freeSlots[s.SlotID] = struct{}{}
	}

	findSlot := func() int16 {
		for i := int16(67); i <= 306; i++ {
			if _, ok := freeSlots[i]; !ok {
				return i
			}
		}
		return -1
	}

	for i := 0; i < len(bankSlots)-1; i++ {
		for j := i; true; j++ {
			if len(bankSlots) == j+1 || bankSlots[i].SlotID != bankSlots[j+1].SlotID {
				break
			}

			free := findSlot()
			if free == -1 {
				continue
			}

			fmt.Printf("%d => %d\n", bankSlots[j+1].SlotID, free)
			freeSlots[free] = struct{}{}
			bankSlots[j+1].SlotID = free
			bankSlots[j+1].Update()
		}
	}

}

func cmdSpawnMobs(count, npcID, sv int, mapID int, NPCsSpawnPoint []string) {
	for i := 0; i < int(count); i++ {
		randomInt := rand.Intn(len(NPCsSpawnPoint))
		npcPos := &database.NpcPosition{ID: len(database.GetNPCPostions()), NPCID: int(npcID), MapID: int16(mapID), Attackable: true, IsNPC: false, RespawnTime: 30, Count: 30, MinLocation: "120,120", MaxLocation: "150,150"}
		database.SetNPCPos(npcPos.ID, npcPos)

		npc, ok := database.GetNpcInfo(npcID)
		if !ok {
			continue
		}
		newai := &database.AI{ID: len(database.AIs), HP: npc.MaxHp, Map: int16(mapID), PosID: npcPos.ID, RunningSpeed: 7, Server: sv, WalkingSpeed: 3, Once: false}
		newai.OnSightPlayers = make(map[int]interface{})
		coordinate := database.ConvertPointToLocation(NPCsSpawnPoint[randomInt])
		randomLocX := randFloats(coordinate.X-6, coordinate.X+6)
		randomLocY := randFloats(coordinate.Y-6, coordinate.Y+6)
		loc := utils.Location{X: randomLocX, Y: randomLocY}
		npcPos.MinLocation = fmt.Sprintf("%.1f,%.1f", randomLocX, randomLocY)
		maxX := randomLocX + 15
		maxY := randomLocY + 15
		npcPos.MaxLocation = fmt.Sprintf("%.1f,%.1f", maxX, maxY)
		npcPos.Create()
		newai.Coordinate = loc.String()
		log.Print(newai.Coordinate)
		newai.Handler = newai.AIHandler
		database.AIsByMap[newai.Server][newai.Map] = append(database.AIsByMap[newai.Server][newai.Map], newai)
		database.DungeonsAiByMap[newai.Server][newai.Map] = append(database.AIsByMap[newai.Server][newai.Map], newai)
		database.AIs[newai.ID] = newai

		database.GenerateIDForAI(newai)
		newai.Create()
		//ai.Init()
		if newai.WalkingSpeed > 0 {
			go newai.Handler()
		}
	}
}
func generateMobs(n int, m int) {
	for i := n; i <= m; i++ {
		npcPos := database.GetNPCPosByID(i)
		if npcPos == nil {
			continue
		}
		if npcPos.IsNPC {
			continue
		}
		for j := 0; j < int(npcPos.Count); j++ {
			npcID := npcPos.NPCID
			npc, ok := database.GetNpcInfo(npcID)
			if !ok || npc == nil {
				log.Print("Null npc:")
				log.Print(npcPos.NPCID)
				continue
			}
			newai := &database.AI{
				ID:           len(database.AIs),
				HP:           npc.MaxHp,
				Map:          npcPos.MapID,
				PosID:        npcPos.ID,
				RunningSpeed: float64(npc.RunningSpeed),
				Server:       1,
				WalkingSpeed: float64(npc.WalkingSpeed),
				Once:         false,
				CanAttack:    true,
				Faction:      0,
				IsDead:       false,
				NPCpos:       npcPos,
			}

			newai.OnSightPlayers = make(map[int]interface{})
			minLoc := database.ConvertPointToLocation(npcPos.MinLocation)
			maxLoc := database.ConvertPointToLocation(npcPos.MaxLocation)
			loc := utils.Location{X: utils.RandFloat(minLoc.X, maxLoc.X), Y: utils.RandFloat(minLoc.Y, maxLoc.Y)}
			newai.Coordinate = loc.String()
			log.Print(newai.Coordinate)
			newai.Handler = newai.AIHandler
			database.GenerateIDForAI(newai)
			database.AIsByMap[newai.Server][newai.Map] = append(database.AIsByMap[newai.Server][newai.Map], newai)
			database.AIs[newai.ID] = newai
			newai.Create()
			if newai.WalkingSpeed > 0 {
				go newai.Handler()
			}
		}

	}
}
func RemoveEventItem(id int) {
	for i, other := range database.EventItems {
		if other == id {
			database.EventItems = append(database.EventItems[:i], database.EventItems[i+1:]...)
			break
		}
	}
}
func cmdEvents(event string) {
	switch event {
	case "happyhour":
		if database.STRHappyHourRate == 0.00 {
			rate := utils.RandFloat(0.05, 0.10)
			database.STRHappyHourRate = rate
			msg := fmt.Sprintf("Upgrading happy hour started, %d%% upgrading bonus for one hour.", int(rate*100))
			makeAnnouncement(msg)
			//msg = "@here " + msg
			time.AfterFunc(time.Hour, func() {
				database.STRHappyHourRate = 0.00
				msg := "Upgrading happy hour ended."
				makeAnnouncement(msg)
			})
		} else {
			database.STRHappyHourRate = 0.00
			msg := "Upgrading happy hour ended."
			makeAnnouncement(msg)
		}
	case "loto":
		database.CountLoto(600)
	case "dragonbox":
		if funk.Contains(database.EventItems, 13370000) {
			makeAnnouncement("Dragon box drop event deactivated")
			RemoveEventItem(13370000)
		} else {
			makeAnnouncement("Dragon box drop event activated")
			database.EventItems = append(database.EventItems, 13370000)
		}
	case "xmas":
		if funk.Contains(database.EventItems, 200000031) {
			makeAnnouncement("X-mas drop event deactivated")
			RemoveEventItem(200000031)
		} else {
			makeAnnouncement("X-mas drop event activated")
			database.EventItems = append(database.EventItems, 200000031)
		}
	}
}
