package auth

import (
	"encoding/binary"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/twodragon/kore-server/database"
	"github.com/twodragon/kore-server/npc"
	"github.com/twodragon/kore-server/player"
	"github.com/twodragon/kore-server/utils"

	dbg "runtime/debug"

	"github.com/thoas/go-funk"
)

type StartGameHandler struct {
}

var (
	GAME_STARTED = utils.Packet{0xAA, 0x55, 0xE6, 0x00, 0x17, 0xE1, 0x00, 0xF3, 0x0C, 0x1F, 0xF1, 0x0C, 0x08, 0x12, 0x00, 0x00, 0x01,
		0x00, 0x0C, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x01, 0x07, 0x01, 0x02, 0x03, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0D, 0x20, 0x01, 0x00,
		0x0C, 0x20, 0x01, 0x00, 0x08, 0x20, 0x01, 0xE0, 0x03, 0x00, 0x00, 0x04, 0xE0, 0x03, 0x0C, 0x60, 0x00, 0x00, 0x64, 0x60, 0x05, 0x06, 0x00, 0x00, 0x10,
		0x0E, 0x00, 0x00, 0x51, 0x20, 0x07, 0x00, 0xCA, 0x20, 0x03, 0x00, 0x24, 0x20, 0x03, 0x00, 0x48, 0x20, 0x03, 0x60, 0x00, 0x01, 0x03, 0x01, 0x20, 0x00,
		0x60, 0x09, 0x60, 0x00, 0x40, 0x74, 0xC0, 0x00, 0x03, 0x74, 0x3B, 0xA4, 0x0B, 0x40, 0x0B, 0x13, 0x05, 0x32, 0x30, 0x31, 0x38, 0x2D, 0x30, 0x34, 0x2D,
		0x33, 0x30, 0x20, 0x30, 0x39, 0x3A, 0x31, 0x37, 0x3A, 0x34, 0x34, 0x40, 0x17, 0xE0, 0x1D, 0x00, 0x09, 0x02, 0x01, 0x00, 0x01, 0x00, 0x01, 0x00, 0xA1,
		0x01, 0x00, 0x60, 0x4C, 0x03, 0x00, 0xC0, 0x75, 0x06, 0x60, 0x0D, 0x00, 0x0C, 0xE0, 0x1D, 0x3E, 0xE0, 0xFF, 0x00, 0xE0, 0xFF, 0x00, 0xE0, 0xFF, 0x00,
		0xE0, 0xFF, 0x00, 0xE0, 0xFF, 0x00, 0xE0, 0xFF, 0x00, 0xE0, 0xFF, 0x00, 0xE0, 0x55, 0x00, 0x00, 0x03, 0xE0, 0x55, 0x5E, 0xE0, 0xFF, 0x00, 0xE0, 0xFF,
		0x00, 0xE0, 0xFF, 0x00, 0xE0, 0xB8, 0x00, 0x01, 0x00, 0x00, 0x55, 0xAA}

	CHARACTER_GONE = utils.Packet{0xAA, 0x55, 0x05, 0x00, 0x21, 0x02, 0x00, 0x55, 0xAA}

	MOB_DISAPPEARED   = utils.Packet{0xAA, 0x55, 0x10, 0x00, 0x31, 0x02, 0x09, 0x00, 0x0A, 0x00, 0x55, 0xAA}
	HOUSE_DISAPPEARED = utils.Packet{}

	AID_ITEM_HANDLE = utils.Packet{0xaa, 0x55, 0x6b, 0x00, 0xa3, 0x03, 0x01, 0x32, 0x30, 0x33, 0x30, 0x2d, 0x31, 0x32, 0x2d, 0x31, 0x38, 0x20, 0x31, 0x30, 0x3a, 0x33, 0x38, 0x3a, 0x30, 0x35, 0x00, 0x01, 0x32, 0x30, 0x34, 0x30, 0x2d, 0x31, 0x32, 0x2d, 0x31, 0x38, 0x20, 0x31, 0x30, 0x3a, 0x33, 0x38, 0x3a, 0x35, 0x30, 0x00, 0x01, 0x33, 0x30, 0x32, 0x30, 0x2d, 0x30, 0x36, 0x2d, 0x31, 0x38, 0x20, 0x31, 0x30, 0x3a, 0x33, 0x38, 0x3a, 0x35, 0x30, 0x00, 0x01, 0x33, 0x30, 0x32, 0x30, 0x2d, 0x30, 0x36, 0x2d, 0x31, 0x38, 0x20, 0x31, 0x30, 0x3a, 0x33, 0x38, 0x3a, 0x35, 0x30, 0x00, 0x01, 0x33, 0x30, 0x32, 0x30, 0x2d, 0x30, 0x36, 0x2d, 0x31, 0x38, 0x20, 0x31, 0x30, 0x3a, 0x33, 0x38, 0x3a, 0x35, 0x30, 0x00, 0x55, 0xaa}

	NPC_APPEARED = utils.Packet{0xAA, 0x55, 0x5D, 0x00, 0x31, 0x01, 0x00, 0x00, 0x00, 0x00, 0x12, 0x47, 0x69, 0x6E, 0x73, 0x65,
		0x6E, 0x67, 0x20, 0x44, 0x69, 0x67, 0x67, 0x65, 0x72, 0x20, 0x44, 0x6F, 0x6E, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x00, 0x00, 0x00, 0x01, 0x01, 0x00, 0x00, 0xA0, 0x41, 0x00, 0x00, 0xA0, 0x41, 0x00, 0x00, 0x00, 0x00, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x64, 0x00, 0x55, 0xAA}

	NPC_DISAPPEARED = MOB_DISAPPEARED

	HOUSE_APPEAR = utils.Packet{0xAA, 0x55, 0x3A, 0x00, 0xAC, 0x03, 0x0A, 0x00, 0xD5, 0x1B, 0x01, 0x00, 0xD5, 0x1B, 0x01, 0x00, 0x90, 0x01,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xAA}
	HOUSE_NPC_APPEAR = utils.Packet{0xaa, 0x55, 0x23, 0x00, 0xac, 0x03, 0x0a, 0x00, 0x3d, 0x07, 0x01, 0x00, 0x3b, 0x07, 0x01, 0x00, 0xc1, 0xd4,
		0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x80, 0xc4, 0x43, 0x8f, 0xc2, 0x75, 0x3d, 0x00, 0x80, 0xd6, 0x43, 0x00, 0x00, 0x00, 0x55, 0xaa}
	HOUSE_FARMING_APPEAR = utils.Packet{0xaa, 0x55, 0x3b, 0x00, 0xac, 0x03, 0x0a, 0x00, 0xc4, 0x17, 0x01, 0x00, 0x3b, 0x07, 0x01, 0x00, 0xd2, 0x86,
		0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xaa}
	StartGameMutex sync.RWMutex
)

func (sgh *StartGameHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {
	if s.Character == nil {
		return nil, nil
	}

	if s.Character.IsOnline {
		log.Println(s.Character.Name, "online sorry", s.User.Username)
		return nil, nil
	}

	return sgh.startGame(s)
}

func (csh *StartGameHandler) startGame(s *database.Socket) ([]byte, error) {

	StartGameMutex.Lock()
	defer StartGameMutex.Unlock()
	if s.Character != nil {
		s.Character.IsActive = false
	}
	if s.Stats == nil {
		s.Stats = &database.Stat{}
		s.Stats.Create(s.Character)
	}
	if s.Stats != nil && s.Stats.HP <= 0 {
		s.Stats.HP = s.Stats.MaxHP / 10
	}

	sale := database.FindSale(s.Character.PseudoID)
	if sale != nil {
		sale.Delete()
	}

	trade := database.FindTrade(s.Character)
	if trade != nil {
		trade.Delete()
	}

	_, item, err := s.Character.FindItemInInventory(nil, 99059990, 99059991, 99059992, 99059993, 99059994)
	if item != nil && err == nil {
		_, err = s.Character.RemoveItem(item.SlotID)
		if err != nil {
			return nil, err
		}
	}

	if s.Character.Map == 243 {
		s.Character.Map = 17
		s.Character.Coordinate = database.ConvertPointToCoordinate(37, 453)
	} else if s.Character.Map == 215 {
		s.Character.Map = 24
		s.Character.Coordinate = database.ConvertPointToCoordinate(513, 467)

	} else if s.Character.Map == 233 {
		s.Character.Coordinate = database.ConvertPointToCoordinate(508, 564)
	}
	if s.Character.Map == 255 || s.Character.Map == 212 || s.Character.Map == 249 {
		s.Character.Map = 1
		s.Character.Coordinate = database.ConvertPointToCoordinate(324, 189)
	} else if s.Character.Map == 230 && (!database.WarStarted || !database.CanJoinWar) {
		s.Character.Map = 1
		s.Character.Coordinate = database.ConvertPointToCoordinate(324, 189)
	} else if s.Character.Map == 120 {
		s.Character.Coordinate = database.ConvertPointToCoordinate(269, 247)
	} else if s.Character.IsDungeon || s.Character.Map == 229 {
		s.Character.Map = 1
		s.Character.Coordinate = database.ConvertPointToCoordinate(324, 189)
	}

	if s.Character.Map == 72 || s.Character.Map == 73 || s.Character.Map == 74 || s.Character.Map == 75 {
		f := func(item *database.InventorySlot) bool {
			return item.Activated
		}
		_, item, err := s.Character.FindItemInInventory(f, 200000038, 200000039)
		if err != nil {
			return nil, err
		} else if item == nil {
			s.Character.Map = 1
			s.Character.Coordinate = database.ConvertPointToCoordinate(324, 189)
		}
	}
	s.Character.Socket.Stats.CalculateHonorIDs()
	s.Character.IsDungeon = false
	s.Character.PartyMode = 33
	s.Character.IsinWar = false
	s.Character.LastNPCAction = 0
	s.Character.HasLot = false
	s.Character.IsOnline = true
	s.Character.Respawning = false
	s.Character.Morphed = false
	s.Character.HpRecoveryCooldown = 0
	s.Character.TypeOfBankOpened = 0
	s.Character.DowngradingSkillWarningMessageShowed = false
	s.Character.GroupSettings.ExperienceSharingMethod = 1
	s.Character.GroupSettings.LootDistriburionMethod = 1
	s.Character.KilledMobs = 0
	s.Character.AttackDelay = 0
	s.Character.ShowUpgradingRate = true
	s.Character.Stunned = false
	s.Character.WarContribution = 0
	s.Character.WarKillCount = 0
	s.Character.OmokRequestState = 0

	s.Character.SetInventorySlots(nil)
	s.Character.OnSight.Drops = make(map[int]interface{})
	s.Character.OnSight.NPCs = make(map[int]interface{})
	s.Character.OnSight.Mobs = make(map[int]interface{})
	s.Character.OnSight.Pets = make(map[int]interface{})
	s.Character.OnSight.Players = make(map[int]interface{})
	s.Character.OnSight.BabyPets = make(map[int]interface{})
	s.Character.OnSight.Housingitems = make(map[int]interface{})
	s.Character.House = database.FindHouseByCharId(s.Character.ID)
	if s.Character.Injury > database.MAX_INJURY {
		s.Character.Injury = database.MAX_INJURY
	}
	ip := s.Conn.RemoteAddr().String()
	ip = strings.Split(ip, ":")[0]
	heartbeat := database.GetHeartBeatsByIp(ip)
	if heartbeat == nil {
		heartbeat = &database.HeartBeat{Ip: ip, Count: 0, Last: time.Now()}
		database.SetHeartBeats(heartbeat)
	}

	s.Character.ExploreWorld = func() {
		for {
			if s.Character.ExploreWorld == nil {
				break
			} else {
				exploreWorld(s)
			}

			time.Sleep(time.Second)
		}
	}

	s.Character.CountOnlineHours = s.Character.CountPlayerOnlineHours
	go s.Character.CountOnlineHours()

	s.Character.HandlerCB = s.Character.Handler
	coordinate := database.ConvertPointToLocation(s.Character.Coordinate)
	mapData, err := s.Character.ChangeMap(s.Character.Map, coordinate, true)
	if err != nil {
		return nil, err
	}

	resp := GAME_STARTED
	resp.Insert(utils.IntToBytes(uint64(s.Character.PseudoID), 2, true), 13) // pseudo id
	resp.Insert(utils.IntToBytes(uint64(s.Character.ID), 4, true), 15)       // character id

	index := 20
	resp.Insert([]byte(s.Character.Name), index) // character name
	index += len(s.Character.Name)

	for i := len(s.Character.Name); i < 18; i++ {
		resp.Insert([]byte{0x00}, index)
		index++
	}

	resp[index] = byte(s.Character.Type) // character type
	index += 1

	resp[index] = byte(s.Character.Faction) // character faction
	index += 1

	resp[index] = 4
	index += 1

	resp[index] = byte(s.Character.Map - 1) // character map
	index += 2

	resp.Insert(utils.FloatToBytes(coordinate.X, 4, true), index) // character coordinate-x
	index += 4

	resp.Insert(utils.FloatToBytes(coordinate.Y, 4, true), index) // character coordinate-y
	index += 4
	index += 10

	resp.Overwrite(utils.IntToBytes(uint64(s.Character.Socket.Stats.Honor), 4, true), index)
	index += 4

	//sign := uint16(utils.BytesToInt(resp[4:6], false))
	//log.Print(sign)

	s.Write(resp)
	resp = utils.Packet{}
	//s.Character.ResolveOverlappingItems()
	ggh := &player.GetGoldHandler{}
	gold, _ := ggh.Handle(s)
	s.Write(gold)

	gih := &player.GetInventoryHandler{}
	inventory, err := gih.Handle(s)
	if err != nil {
		return nil, err
	}

	s.Write(inventory)
	s.Write(s.Character.GetPetStats())
	s.Write(mapData)

	honorresp := database.CHANGE_RANK
	honorresp.Insert(utils.IntToBytes(uint64(s.Character.PseudoID), 2, true), 6)
	honorresp.Insert(utils.IntToBytes(uint64(s.Character.HonorRank), 4, true), 8)
	s.Write(honorresp)

	go time.AfterFunc(time.Second*time.Duration(5), func() {
		s.Character.HousingDetails()
	})

	spawnData, err := s.Character.SpawnCharacter()
	if err != nil {
		return nil, err
	}
	s.Write(spawnData)

	gsh := &player.GetStatsHandler{}
	statData, err := gsh.Handle(s)
	if err != nil {
		return nil, err
	}

	s.Write(statData)
	s.Write(s.User.GetTime())

	skillsData, err := s.Skills.GetSkillsData()
	if err != nil {
		return nil, err
	}

	s.Write(skillsData)
	s.Write(s.Character.GetGold())

	r := player.HT_VISIBILITY
	r[9] = byte(s.Character.HTVisibility)
	s.Write(r)

	r = npc.JOB_PROMOTED
	r[6] = byte(s.Character.Class)
	s.Write(r)

	guildData, err := s.Character.GetGuildData()
	if err != nil {
		return nil, err
	}

	s.Write(guildData)

	/*err = s.Character.AddPlayerQuests()
	if err != nil {
		fmt.Printf("Error with load: %s", err)
	}
	//QUEST MOBS LOAD
	s.Character.GetMapQuestMobs()
	QuestList, _ := database.FindQuestsByCharacterID(s.Character.ID)
	for _, quest := range QuestList {
		s.Character.LoadQuests(quest.ID, quest.QuestState)
		quest.Update()
	}*/

	slotData := utils.Packet{}
	slotData.Concat(s.Character.Slotbar)
	s.Write(slotData)

	//AID
	aidbuff, err := database.FindBuffByID(11152, s.Character.ID)
	if err == nil {
		if aidbuff != nil && !s.Character.AidMode {
			go aidbuff.Delete()
		}
	}
	s.Write(AID_ITEM_HANDLE)
	friendresp, err := database.InitFriends(s.Character)
	if err == nil {
		s.Write(friendresp)
	}
	if s.Character.GuildID > 0 {
		guild, err := database.FindGuildByID(s.Character.GuildID)
		if err != nil {
			return nil, err
		} else if guild != nil {
			guild.InformMembers(s.Character)
		}
	}

	mails, err := database.FindMailsByCharacterID(s.Character.ID)
	if err == nil {
		for _, mail := range mails {
			if !mail.IsOpened {
				s.Write(player.MAIL_RECEIVED)
			}
		}
	}

	time.AfterFunc(time.Second*1, func() {
		if s.Character.ExploreWorld != nil {
			go s.Character.ExploreWorld()
		}

		if s.Character.HandlerCB != nil {
			go s.Character.HandlerCB()
		}
	})
	go s.Character.RepurchaseList.Clear()
	go s.Character.ActivityStatus(30)
	//go s.Character.HandleBoxOpener()
	boxes := s.Character.ShowBoxOpenerItems()
	s.Character.ShowEventsDetails()
	s.Write(boxes)
	log.Println(s.Character.Name, "Started the game", s.User.Username)
	return nil, nil
}

func exploreWorld(s *database.Socket) {
	if s == nil {
		return
	}
	if s.User.ID != s.Character.UserID {
		log.Println("hilecimiyiz :D no_explore")
		return
	}
	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
			log.Printf("%+v", string(dbg.Stack()))
		}
	}()

	//if database.Maps[int(s.Character.Map)].HousingMap != 2 {

	exploreMobs(s)
	exploreNPCs(s)
	exploreDrops(s)
	explorePets(s)
	//exploreBabyPets(s)

	//}
	explorePlayers(s)
	if s.Character.Map == 120 {
		exploreHouses(s)
	}

}

func explorePlayers(s *database.Socket) {
	c := s.Character
	if c == nil {
		return
	}

	characters, err := c.GetNearbyCharacters()
	if err != nil {
		log.Println(err)
		return
	}

	for _, character := range characters {

		if character.IsMounting {
			delete(c.OnSight.Players, character.ID)
		}

		c.OnSight.PlayerMutex.RLock()
		_, ok := c.OnSight.Players[character.ID]
		c.OnSight.PlayerMutex.RUnlock()

		if !ok {
			c.OnSight.PlayerMutex.Lock()
			c.OnSight.Players[character.ID] = character.PseudoID
			c.OnSight.PlayerMutex.Unlock()

			opData, err := character.SpawnCharacter()
			if err != nil || opData == nil || len(opData) < 13 {
				continue
			}

			r := utils.Packet{}
			r.Concat(opData)

			if c.CanAttack(character) {
				r.Overwrite(utils.IntToBytes(1, 1, true), 14) // duel state
			}

			resp := utils.Packet{}

			resp.Concat(r)
			if !c.CanAttack(character) {
				resp.Concat([]byte{0xAA, 0x55, 0x02, 0x00, 0x2A, 0x04, 0x55, 0xAA})
			}

			err = s.Write(resp)
			if err != nil {
				log.Print(err)
			}
			go time.AfterFunc(time.Second*time.Duration(1), func() {
				c.Socket.Write((character.GetHPandChi()))
			})

		}
	}

	ids := funk.Map(characters, func(c *database.Character) int {
		return c.ID
	}).([]int)

	c.OnSight.PlayerMutex.RLock()
	losers := utils.SliceDiff(utils.Keys(c.OnSight.Players), ids)
	c.OnSight.PlayerMutex.RUnlock()

	for _, id := range losers {

		loser, err := database.FindCharacterByID(id)
		if err != nil {
			log.Println(err)
			return
		}

		c.OnSight.PlayerMutex.RLock()
		pseudoID := c.OnSight.Players[loser.ID].(uint16)
		c.OnSight.PlayerMutex.RUnlock()

		d := CHARACTER_GONE
		d.Insert(utils.IntToBytes(uint64(pseudoID), 2, true), 6)
		err = s.Write(d)
		if err != nil {
			log.Print(err)
		}

		c.OnSight.PlayerMutex.Lock()
		delete(c.OnSight.Players, id)
		c.OnSight.PlayerMutex.Unlock()
	}
}
func Tester(s []uint16, e uint16) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func exploreMobs(s *database.Socket) {
	c := s.Character
	if c == nil {
		return
	}
	ids, err := c.GetNearbyAIIDs()
	if err != nil {
		log.Println(err)
		return
	}

	for _, id := range ids {
		mob, ok := database.AIs[id]
		if !ok {
			log.Println("!mob ", mob)
			continue
		}
		if c.IsinWar {
			isStone := Tester(database.WarStonesIDs, mob.PseudoID)
			if isStone {
				delete(c.OnSight.Mobs, id)
			}
		}

		c.OnSight.MobMutex.RLock()
		_, ok = c.OnSight.Mobs[id]
		c.OnSight.MobMutex.RUnlock()

		if mob.IsDead && ok {
			c.OnSight.MobMutex.Lock()
			delete(c.OnSight.Mobs, id)
			c.OnSight.MobMutex.Unlock()

			mob.PlayersMutex.Lock()
			delete(mob.OnSightPlayers, c.ID)
			mob.PlayersMutex.Unlock()

		} else if !mob.IsDead && !ok {
			c.OnSight.MobMutex.Lock()
			c.OnSight.Mobs[id] = struct{}{}
			//			log.Println("!err 535: ", mob.ID)
			c.OnSight.MobMutex.Unlock()

			mob.PlayersMutex.Lock()
			mob.OnSightPlayers[c.ID] = struct{}{}
			mob.PlayersMutex.Unlock()

			npcID := uint64(database.GetNPCPosByID(mob.PosID).NPCID)
			npc, ok := database.GetNpcInfo(int(npcID))
			//			log.Println("ok npc", npc.Name, mob.ID)
			if !ok {
				log.Println("!ok npc", npc)
				continue
			}
			coordinate := database.ConvertPointToLocation(mob.Coordinate)

			r := database.MOB_APPEARED
			if (mob.Faction != 0 && mob.Faction == c.Faction) || mob.Faction == 3 { //faction 3 = neutral
				r.Overwrite(utils.IntToBytes(uint64(1), 4, true), 6)
				npc.Level = 1
			} else {
				npc2, ok := database.GetNpcInfo(int(npcID))
				if !ok {
					continue
				}
				npc.Level = npc2.Level
				r.Overwrite([]byte{0xFF, 0xFF, 0xFF, 0xFF}, 6)
			}

			//r.Overwrite([]byte{0xf1,0xd8,0xff,0xff}, 6)  mineral

			rotation := database.GetNPCPosByID(mob.PosID).Rotation
			r.Overwrite(utils.IntToBytes(uint64(rotation), 2, true), 24) // rotation

			r.Insert(utils.IntToBytes(uint64(mob.PseudoID), 2, true), 6) // mob pseudo id
			r.Insert(utils.IntToBytes(npcID, 4, true), 8)                // mob npc id
			r.Insert(utils.IntToBytes(uint64(npc.Level), 4, true), 12)   // mob level
			chi := mob.CHI
			if npc.Type == 29 {
				chi = (npc.MaxHp / 2) / 10
			}
			index := 20
			r.Insert(utils.IntToBytes(uint64(len(npc.Name)), 1, true), index)
			index++
			r.Insert([]byte(npc.Name), index) // mob name
			index += len(npc.Name)
			r.Insert(utils.IntToBytes(uint64(mob.HP), 4, true), index) // mob hp
			index += 4
			r.Insert(utils.IntToBytes(uint64(chi), 4, true), index) // mob chi
			index += 4
			r.Insert(utils.IntToBytes(uint64(npc.MaxHp), 4, true), index) // mob max hp
			index += 4
			r.Insert(utils.IntToBytes(uint64(npc.MaxChi), 4, true), index) // mob max chi
			index += 6
			r.Insert(utils.FloatToBytes(coordinate.X, 4, true), index) // coordinate-x
			index += 4
			r.Insert(utils.FloatToBytes(coordinate.Y, 4, true), index) // coordinate-y
			index += 8
			r.Insert(utils.FloatToBytes(coordinate.X, 4, true), index) // coordinate-x
			index += 4
			r.Insert(utils.FloatToBytes(coordinate.Y, 4, true), index) // coordinate-y
			index += 4
			r.SetLength(int16(index + 16))

			//LOADMOBSBUFFS

			buffs, err := database.FindBuffsByAiPseudoID(mob.PseudoID)
			if err == nil && len(buffs) > 0 {
				index := 5
				br := database.DEAL_BUFF_AI
				br.Insert(utils.IntToBytes(uint64(mob.PseudoID), 2, true), index) // ai pseudo id
				index += 2
				br.Insert(utils.IntToBytes(uint64(mob.PseudoID), 2, true), index) // ai pseudo id
				index += 2
				br.Insert(utils.IntToBytes(uint64(mob.HP), 4, true), index) // ai current hp
				index += 4
				br.Insert(utils.IntToBytes(uint64(chi), 4, true), index)        // ai current chi
				br.Overwrite(utils.IntToBytes(uint64(len(buffs)), 1, true), 21) //BUFF ID
				index = 22
				count := 0
				for _, buff := range buffs {
					br.Insert(utils.IntToBytes(uint64(buff.ID), 4, true), index) //BUFF ID
					index += 4
					if count < len(buffs)-1 {
						br.Insert(utils.IntToBytes(uint64(0), 2, true), index) //BUFF ID
						index += 2
					}
					count++
				}
				index += 4
				br.SetLength(int16(index))
				r.Concat(br)
			} else if err != nil && len(buffs) != 0 {
				fmt.Printf("LoadBuffsToMob: %s", err.Error())
			}

			if c.IsinWar {
				isStone := Tester(database.WarStonesIDs, mob.PseudoID)
				if isStone {
					if c.Faction == 1 {
						if ok, _ := utils.Contains(database.WarStones[int(mob.PseudoID)].NearbyZuhangV, c.ID); !ok {
							database.WarStones[int(mob.PseudoID)].NearbyZuhangV = append(database.WarStones[int(mob.PseudoID)].NearbyZuhangV, c.ID)
						}
					} else {
						if ok, _ := utils.Contains(database.WarStones[int(mob.PseudoID)].NearbyShaoV, c.ID); !ok {
							database.WarStones[int(mob.PseudoID)].NearbyShaoV = append(database.WarStones[int(mob.PseudoID)].NearbyShaoV, c.ID)
						}
					}
					if c.Socket.Stats.HP <= 0 {
						if c.Faction == 1 {
							if ok, _ := utils.Contains(database.WarStones[int(mob.PseudoID)].NearbyZuhangV, c.ID); ok {
								database.WarStones[int(mob.PseudoID)].RemoveZuhang(c.ID)
							}
						} else {
							if ok, _ := utils.Contains(database.WarStones[int(mob.PseudoID)].NearbyShaoV, c.ID); ok {
								database.WarStones[int(mob.PseudoID)].RemoveShao(c.ID)
							}
						}
					}
					resp := database.STONE_APPEARED
					resp.Insert(utils.IntToBytes(uint64(mob.PseudoID), 2, true), 6) // mob pseudo id
					resp.Insert(utils.IntToBytes(npcID, 4, true), 8)                // mob npc id
					resp.Insert(utils.IntToBytes(uint64(npc.Level), 4, true), 12)   // mob level
					resp.Insert(utils.IntToBytes(uint64(mob.HP), 8, true), 33)      // mob hp
					resp.Insert(utils.IntToBytes(uint64(npc.MaxHp), 8, true), 41)   // mob max hp
					resp.Insert(utils.FloatToBytes(coordinate.X, 4, true), 51)      // coordinate-x
					resp.Insert(utils.FloatToBytes(coordinate.Y, 4, true), 55)      // coordinate-y
					resp.Insert(utils.FloatToBytes(coordinate.X, 4, true), 63)      // coordinate-x
					resp.Insert(utils.FloatToBytes(coordinate.Y, 4, true), 67)      // coordinate-y
					resp.Overwrite(utils.IntToBytes(uint64(database.WarStones[int(mob.PseudoID)].ConquereValue), 1, false), 37)
					resp.Overwrite([]byte{0xc8}, 45)
					s.Conn.Write(resp)
					continue
				}
			}

			err = s.Write(r)
			if err != nil {
				log.Print(err)
			}

			//resp.Concat(r)
		}
	}

	c.OnSight.MobMutex.RLock()
	losers := utils.SliceDiff(utils.Keys(c.OnSight.Mobs), ids)
	c.OnSight.MobMutex.RUnlock()
	//losers = append(losers, utils.SliceDiff(ids, utils.Keys(c.OnSight.Mobs))...)

	for _, id := range losers {
		loser := database.AIs[id]
		coordinate := database.ConvertPointToLocation(loser.Coordinate)
		r := MOB_DISAPPEARED
		r.Insert(utils.IntToBytes(uint64(loser.PseudoID), 2, true), 6) // mob pseudo id
		r.Insert(utils.FloatToBytes(coordinate.X, 4, true), 12)        // coordinate-x
		r.Insert(utils.FloatToBytes(coordinate.Y, 4, true), 16)        // coordinate-y

		err = s.Write(r)
		if err != nil {
			log.Print(err)
		}

		//resp.Concat(r)

		c.OnSight.MobMutex.Lock()
		delete(c.OnSight.Mobs, loser.ID)
		c.OnSight.MobMutex.Unlock()

		loser.PlayersMutex.Lock()
		delete(loser.OnSightPlayers, c.ID)
		loser.PlayersMutex.Unlock()
	}
}

func explorePets(s *database.Socket) {
	c := s.Character
	if c == nil {
		return
	}

	characters, err := c.GetNearbyCharacters()
	if err != nil {
		log.Println(err)
		return
	}

	characters = append(characters, c)
	petSlots := make(map[int]*database.InventorySlot)
	petIDs := []int{}

	characters = funk.Filter(characters, func(ch *database.Character) bool {
		slots, err := ch.InventorySlots()
		if err != nil {
			return false
		}

		petSlot := slots[0x0A]
		if petSlot.Pet == nil || !petSlot.Pet.IsOnline {
			return false
		}

		petIDs = append(petIDs, petSlot.Pet.PseudoID)
		petSlots[ch.ID] = petSlot
		return true
	}).([]*database.Character)

	resp := utils.Packet{}
	for _, character := range characters {

		petSlot := petSlots[character.ID]
		pet := petSlot.Pet

		c.OnSight.PetsMutex.RLock()
		_, ok := c.OnSight.Pets[pet.PseudoID]
		c.OnSight.PetsMutex.RUnlock()

		if pet.HP <= 0 || !character.IsActive || character.Respawning {

			c.OnSight.PetsMutex.Lock()
			delete(c.OnSight.Pets, pet.PseudoID)
			c.OnSight.PetsMutex.Unlock()

		} else if !ok {

			c.OnSight.PetsMutex.Lock()
			c.OnSight.Pets[pet.PseudoID] = struct{}{}
			c.OnSight.PetsMutex.Unlock()

			r := database.PET_APPEARED
			r.Insert(utils.IntToBytes(uint64(pet.PseudoID), 2, true), 6)          // pet pseudo id
			r.Insert(utils.IntToBytes(uint64(petSlot.ItemID), 4, true), 8)        // pet npc id
			r.Insert(utils.IntToBytes(uint64(pet.Level), 4, true), 12)            // pet level
			r.Overwrite(utils.IntToBytes(uint64(character.Faction), 4, true), 16) //Pets to neutral
			//r.Insert([]byte{0x09, 0x57, 0x69, 0x6C, 0x64, 0x20, 0x42, 0x6F, 0x61, 0x72}, 20)
			//r.Insert(utils.IntToBytes(uint64(len(pet.Name)), 1, true), 20)
			//	index++
			index := 0
			if pet.Name != "" {
				r.Insert(utils.IntToBytes(uint64(len(character.Name+"|"+pet.Name)), 1, true), 20)
				r.Insert([]byte(character.Name+"|"+pet.Name), 21) // pet name
				index = len(character.Name+"|"+pet.Name) + 21
			} else {
				r.Insert(utils.IntToBytes(uint64(len(character.Name)), 1, true), 20)
				r.Insert([]byte(character.Name), 21) // pet name
				index = len(character.Name) + 21
			}
			r.Insert(utils.IntToBytes(uint64(pet.HP), 4, true), index)        // pet hp
			r.Insert(utils.IntToBytes(uint64(pet.CHI), 4, true), index+4)     // pet chi
			r.Insert(utils.IntToBytes(uint64(pet.MaxHP), 4, true), index+8)   // pet max hp
			r.Insert(utils.IntToBytes(uint64(pet.MaxCHI), 4, true), index+12) // pet max chi
			r.Insert(utils.IntToBytes(3, 2, true), index+16)                  //
			r.Insert(utils.FloatToBytes(pet.Coordinate.X, 4, true), index+18) // coordinate-x
			r.Insert(utils.FloatToBytes(pet.Coordinate.Y, 4, true), index+22) // coordinate-y
			r.Insert(utils.FloatToBytes(12, 4, true), index+26)               // z?
			r.Insert(utils.FloatToBytes(pet.Coordinate.X, 4, true), index+30) // coordinate-x
			r.Insert(utils.FloatToBytes(pet.Coordinate.Y, 4, true), index+34) // coordinate-y
			r.Insert(utils.FloatToBytes(12, 4, true), index+38)               // z?
			r.Insert([]byte{0x00, 0x00, 0x00, 0x00}, index+42)
			//r = append(r[:index+42], r[index+50:]...)
			//r.Overwrite(utils.Packet{0x00, 0x00, 0x00, 0x00, 0xFF, 0xFF, 0x00, 0xE8, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, index+42)

			r.SetLength(int16(binary.Size(r) - 6))
			resp.Concat(r)
		}
	}

	c.OnSight.PetsMutex.RLock()
	losers := utils.SliceDiff(utils.Keys(c.OnSight.Pets), petIDs)
	c.OnSight.PetsMutex.RUnlock()

	for _, id := range losers {
		/*/
		loser, ok := database.GetFromRegister(c.Socket.User.ConnectedServer, c.Map, uint16(id)).(*database.PetSlot)
		if !ok {
			continue
		}
		*/

		r := MOB_DISAPPEARED
		r.Insert(utils.IntToBytes(uint64(id), 2, true), 6) // mob pseudo id
		r.Insert(utils.FloatToBytes(0, 4, true), 12)       // coordinate-x
		r.Insert(utils.FloatToBytes(0, 4, true), 16)       // coordinate-y

		resp.Concat(r)
		c.OnSight.PetsMutex.Lock()
		delete(c.OnSight.Pets, id)
		c.OnSight.PetsMutex.Unlock()
	}

	err = s.Write(resp)
	if err != nil {
		log.Print(err)
	}

}

func exploreBabyPets(s *database.Socket) {
	c := s.Character
	if c == nil {
		return
	}
	ids, err := c.GetNearbyBabyPetsIDs()
	if err != nil {
		log.Println(err)
		return
	}

	for _, id := range ids {
		babypet, ok := database.BabyPets[id]
		if !ok {
			continue
		}

		c.OnSight.BabyPetsMutex.RLock()
		_, ok = c.OnSight.BabyPets[id]
		c.OnSight.BabyPetsMutex.RUnlock()

		if babypet.IsDead && ok {
			c.OnSight.BabyPetsMutex.Lock()
			delete(c.OnSight.BabyPets, id)
			c.OnSight.BabyPetsMutex.Unlock()

			babypet.PlayersMutex.Lock()
			delete(babypet.OnSightPlayers, c.ID)
			babypet.PlayersMutex.Unlock()

		} else if !babypet.IsDead && !ok {
			c.OnSight.BabyPetsMutex.Lock()
			c.OnSight.BabyPets[id] = struct{}{}
			c.OnSight.BabyPetsMutex.Unlock()

			babypet.PlayersMutex.Lock()
			babypet.OnSightPlayers[c.ID] = struct{}{}
			babypet.PlayersMutex.Unlock()

			r, err := babypet.SpawnBabyPet()
			if err != nil {
				log.Println(err)

			}
			s.Write(r)
			continue
		}
	}

	c.OnSight.BabyPetsMutex.RLock()
	losers := utils.SliceDiff(utils.Keys(c.OnSight.BabyPets), ids)
	c.OnSight.BabyPetsMutex.RUnlock()
	//losers = append(losers, utils.SliceDiff(ids, utils.Keys(c.OnSight.Mobs))...)

	for _, id := range losers {
		loser := database.BabyPets[id]
		coordinate := database.ConvertPointToLocation(loser.Coordinate)
		r := MOB_DISAPPEARED
		r.Insert(utils.IntToBytes(uint64(loser.PseudoID), 2, true), 6) // mob pseudo id
		r.Insert(utils.FloatToBytes(coordinate.X, 4, true), 12)        // coordinate-x
		r.Insert(utils.FloatToBytes(coordinate.Y, 4, true), 16)        // coordinate-y

		err = s.Write(r)
		if err != nil {
			log.Print(err)
		}

		//resp.Concat(r)

		c.OnSight.BabyPetsMutex.Lock()
		delete(c.OnSight.BabyPets, loser.ID)
		c.OnSight.BabyPetsMutex.Unlock()

		loser.PlayersMutex.Lock()
		delete(loser.OnSightPlayers, c.ID)
		loser.PlayersMutex.Unlock()
	}
}
func exploreNPCs(s *database.Socket) {
	c := s.Character
	if c == nil {
		return
	}

	ids, err := c.GetNearbyNPCIDs()
	if err != nil {
		log.Println(err)
		return
	}

	npcPosIds := []int{}
	resp := utils.Packet{}
	for _, id := range ids {
		npcPos := database.GetNPCPosByID(id)
		if npcPos == nil { //
			continue
		}
		npc, ok := database.GetNpcInfo(npcPos.NPCID)
		if !ok || npc == nil {
			continue
		}
		npcPosIds = append(npcPosIds, npcPos.ID)

		c.OnSight.NpcMutex.RLock()
		_, ok = c.OnSight.NPCs[npcPos.ID]
		c.OnSight.NpcMutex.RUnlock()

		if !ok {
			c.OnSight.NpcMutex.Lock()
			c.OnSight.NPCs[npcPos.ID] = struct{}{}
			c.OnSight.NpcMutex.Unlock()

			minLocation := database.ConvertPointToLocation(npcPos.MinLocation)
			maxLocation := database.ConvertPointToLocation(npcPos.MaxLocation)
			coordinate := &utils.Location{X: (minLocation.X + maxLocation.X) / 2, Y: (minLocation.Y + maxLocation.Y) / 2}

			r := NPC_APPEARED
			r.Insert(utils.IntToBytes(uint64(npcPos.PseudoID), 2, true), 6) // npc pseudo id
			r.Insert(utils.IntToBytes(uint64(npc.ID), 4, true), 8)          // npc id
			r.Insert(utils.IntToBytes(uint64(npc.Level), 4, true), 12)      // npc level
			r.Insert(utils.IntToBytes(uint64(npc.MaxHp), 4, true), 39)      // npc hp
			r.Insert(utils.IntToBytes(uint64(npc.MaxHp), 4, true), 47)      // npc hp
			r.Insert(utils.FloatToBytes(coordinate.X, 4, true), 57)         // coordinate-x
			r.Insert(utils.FloatToBytes(coordinate.Y, 4, true), 61)         // coordinate-y
			r.Insert(utils.FloatToBytes(coordinate.X, 4, true), 69)         // coordinate-x
			r.Insert(utils.FloatToBytes(coordinate.Y, 4, true), 73)         // coordinate-y

			resp.Concat(r)
		}
	}

	c.OnSight.NpcMutex.RLock()
	losers := utils.SliceDiff(utils.Keys(c.OnSight.NPCs), npcPosIds)
	c.OnSight.NpcMutex.RUnlock()

	for _, id := range losers {
		loserPos := database.GetNPCPosByID(id)

		if loserPos == nil {
			continue
		}

		minLocation := database.ConvertPointToLocation(loserPos.MinLocation)
		maxLocation := database.ConvertPointToLocation(loserPos.MaxLocation)
		coordinate := &utils.Location{X: (minLocation.X + maxLocation.X) / 2, Y: (minLocation.Y + maxLocation.Y) / 2}

		r := NPC_DISAPPEARED
		r.Insert(utils.IntToBytes(uint64(loserPos.PseudoID), 2, true), 6) // mob pseudo id
		r.Insert(utils.FloatToBytes(coordinate.X, 4, true), 12)           // coordinate-x
		r.Insert(utils.FloatToBytes(coordinate.Y, 4, true), 16)           // coordinate-y

		resp.Concat(r)
		c.OnSight.NpcMutex.Lock()
		delete(c.OnSight.NPCs, loserPos.ID)
		c.OnSight.NpcMutex.Unlock()
	}

	err = s.Write(resp)
	if err != nil {
		log.Print(err)
	}

}

func exploreHouses(s *database.Socket) {
	c := s.Character
	if c == nil {
		return
	}
	if c.Map != 120 {
		return
	}
	ids, err := c.GetNearbyHousesIDs()
	if err != nil {
		log.Println(err)
		return
	}

	for _, id := range ids {
		database.HousingItemsMutex.RLock()
		house, ok := database.HousingItems[id]
		database.HousingItemsMutex.RUnlock()
		if !ok || house == nil {
			continue
		}

		c.OnSight.HousingitemsMutex.RLock()
		_, ok = c.OnSight.Housingitems[id]
		c.OnSight.HousingitemsMutex.RUnlock()

		if !ok {
			c.OnSight.HousingitemsMutex.Lock()
			c.OnSight.Housingitems[id] = struct{}{}
			c.OnSight.HousingitemsMutex.Unlock()

			owner, err := database.FindCharacterByID(house.OwnerID)
			if err != nil {
				continue
			}
			main := database.FindHouseByCharId(house.OwnerID)
			if database.HouseItemsInfos[house.HouseID].Type == 1 {

				resp := HOUSE_APPEAR

				index := 24
				resp.Insert(utils.FloatToBytes(house.PosX, 4, true), index)
				index += 4
				resp.Insert(utils.FloatToBytes(house.PosZ, 4, true), index)
				index += 4
				resp.Insert(utils.FloatToBytes(house.PosY, 4, true), index)
				index += 4
				resp.Insert(utils.IntToBytes(uint64(house.IsPublic), 1, true), index)
				index++

				formatdate := house.ExpirationDate.Time.Format("2006-01-02 15:04:05")

				resp.Insert(utils.IntToBytes(uint64(len(formatdate)), 1, true), index)
				index++
				resp.Insert([]byte(formatdate), index)
				index += len(formatdate)

				resp.Insert(utils.IntToBytes(uint64(len(owner.Name)), 1, true), index)
				index += 1
				resp.Insert([]byte(owner.Name), index)
				index += len(owner.Name)

				resp.SetLength(int16(binary.Size(resp) - 6))

				resp.Overwrite(utils.IntToBytes(uint64(house.PseudoID), 4, true), 8)
				resp.Overwrite(utils.IntToBytes(uint64(main.PseudoID), 4, true), 12)
				resp.Overwrite(utils.IntToBytes(uint64(house.HouseID), 4, true), 16)

				err = s.Write(resp)
				if err != nil {
					log.Print(err)
				}
			} else {

				resp := HOUSE_NPC_APPEAR
				resp.Overwrite(utils.IntToBytes(uint64(house.PseudoID), 4, true), 8)
				resp.Overwrite(utils.IntToBytes(uint64(main.PseudoID), 4, true), 12)
				resp.Overwrite(utils.IntToBytes(uint64(house.HouseID), 4, true), 16)
				resp.Overwrite(utils.FloatToBytes(house.PosX, 4, true), 24)
				resp.Overwrite(utils.FloatToBytes(house.PosZ, 4, true), 28)
				resp.Overwrite(utils.FloatToBytes(house.PosY, 4, true), 32)

				err = s.Write(resp)
				if err != nil {
					log.Print(err)
				}
			}

		}
	}

	c.OnSight.HousingitemsMutex.RLock()
	losers := utils.SliceDiff(utils.Keys(c.OnSight.Housingitems), ids)
	c.OnSight.HousingitemsMutex.RUnlock()

	for _, id := range losers {
		database.HousingItemsMutex.RLock()
		loser, ok := database.HousingItems[id]
		database.HousingItemsMutex.RUnlock()
		if !ok {
			continue
		}
		r := utils.Packet{0xaa, 0x55, 0x08, 0x00, 0xac, 0x04, 0x0a, 0x00, 0x3e, 0x07, 0x01, 0x00, 0x55, 0xaa}
		r.Overwrite(utils.IntToBytes(uint64(loser.PseudoID), 4, true), 8)

		err = s.Write(r)
		if err != nil {
			log.Print(err)
		}

		//resp.Concat(r)

		c.OnSight.HousingitemsMutex.Lock()
		delete(c.OnSight.Housingitems, loser.ID)
		c.OnSight.HousingitemsMutex.Unlock()

		loser.PlayersMutex.Lock()
		delete(loser.OnSightPlayers, c.ID)
		loser.PlayersMutex.Unlock()
	}

}

func exploreDrops(s *database.Socket) {
	c := s.Character
	if c == nil {
		return
	}

	ids, err := c.GetNearbyDrops()
	if err != nil {
		log.Println(err)
		return
	}
	func() {
		for _, id := range ids {
			drop := database.GetDrop(s.User.ConnectedServer, c.Map, uint16(id))
			if drop == nil {
				continue
			}

			c.OnSight.DropsMutex.RLock()
			_, ok := c.OnSight.Drops[id]
			c.OnSight.DropsMutex.RUnlock()
			claimer := drop.Claimer
			if claimer == nil {
				claimer = s.Character
			}
			if claimer.PartyID != "" {
				party := database.FindParty(claimer)
				if party != nil {
					m := party.GetMember(s.Character.ID)
					if m != nil && (party.PartyMode == 18 || party.PartyMode == 34) {
						claimer = m.Character
					}
				}

			}

			if !ok {
				c.OnSight.DropsMutex.Lock()
				c.OnSight.Drops[id] = struct{}{}
				c.OnSight.DropsMutex.Unlock()
				r := database.ITEM_DROPPED
				r.Insert(utils.IntToBytes(uint64(id), 2, true), 6) // drop id

				r.Insert(utils.FloatToBytes(drop.Location.X, 4, true), 10) // drop coordinate-x
				r.Insert(utils.FloatToBytes(drop.Location.Y, 4, true), 18) // drop coordinate-y

				r.Insert(utils.IntToBytes(uint64(drop.Item.ItemID), 4, true), 22) // item id
				if drop.Item.Plus > 0 {
					r[27] = 0xA2
					r.Insert(drop.Item.GetUpgrades(), 32)                             // item upgrades
					r.Insert([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, 47) // item sockets
					r.Insert(utils.IntToBytes(uint64(claimer.PseudoID), 2, true), 66) // claimer id
					r.SetLength(0x42)
				} else {
					r[27] = 0xA1
					r.Insert(drop.Item.GetUpgrades(), 32)                             // item upgrades
					r.Insert([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, 47) // item sockets
					r.Insert(utils.IntToBytes(uint64(claimer.PseudoID), 2, true), 66) // claimer id
					r.SetLength(0x42)
				}
				err = s.Write(r)
				if err != nil {
					log.Print(err)
				}

			}
		}
	}()

	c.OnSight.DropsMutex.RLock()
	losers := utils.SliceDiff(utils.Keys(c.OnSight.Drops), ids)
	c.OnSight.DropsMutex.RUnlock()

	func() {
		for _, id := range losers {

			r := database.DROP_DISAPPEARED
			r.Insert(utils.IntToBytes(uint64(id), 2, true), 6) //drop id

			err = s.Write(r)
			if err != nil {
				log.Print(err)
			}

			c.OnSight.DropsMutex.Lock()
			delete(c.OnSight.Drops, id)
			c.OnSight.DropsMutex.Unlock()
		}
	}()
}
