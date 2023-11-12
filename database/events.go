package database

import (
	"fmt"
	"math"
	"time"

	"github.com/twodragon/kore-server/messaging"
	"github.com/twodragon/kore-server/nats"
	"github.com/twodragon/kore-server/utils"

	null "gopkg.in/guregu/null.v3"
)

var (
	ANNOUNCEMENTa   = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x71, 0x06, 0x00, 0x55, 0xAA}
	mob             *AI
	respawns        = 4
	isCiRunning     = false
	IsDungeonClosed = false
	YY_TIME_LIMIT   = 30
	DUNGEON_TIMER   = utils.Packet{0xAA, 0x55, 0x06, 0x00, 0xC0, 0x17, 0x09, 0x07, 0x00, 0x00, 0x55, 0xAA}
	//6th = 0x19 -> Monster count x/..

	SeasonDungeonCharacters = make(map[int]*Character)
	IsDivineDungeonClosed   = false
	IsSeasonDungeon1Closed  bool
	IsSeasonDungeon2Closed  bool
	IsSeasonDungeon3Closed  bool
	IsSeasonDungeon4Closed  bool

	TIME_LIMIT = 3600

	NcashMobs        []int
	NcashMobsPosIDs  = []int{5335, 5336, 5337, 5338}
	SeasonMobsPosIDs = []int{5344, 5345}

	isActive        = false
	participants    []*Participant
	lotoPrice       = uint64(500)
	minParticipants = 3
	countDownTimer  = 600 //seconds
)

type Participant struct {
	character *Character
	number    int
}

func CiEventSchedule() {

	if !isCiRunning {
		hour, minutes := getHour(null.NewTime(time.Now(), true))
		if (hour == 4 && minutes == 0) || (hour == 12 && minutes == 0) || (hour == 20 && minutes == 0) {
			StartCiEventCountdown(600)
			isCiRunning = true

		}

	}
	time.AfterFunc(time.Minute, func() {
		CiEventSchedule()
	})
}
func StartCiEventCountdown(cd int) {
	if cd >= 120 {
		msg := fmt.Sprintf("Central Island event will start in %d minutes.", cd/60)
		MakeAnnouncementx(msg)
		time.AfterFunc(time.Second*60, func() {
			StartCiEventCountdown(cd - 60)
		})
	} else if cd > 0 {
		msg := fmt.Sprintf("Central Island event will start in %d seconds.", cd)
		MakeAnnouncementx(msg)
		time.AfterFunc(time.Second*10, func() {
			StartCiEventCountdown(cd - 10)
		})
	}
	if cd <= 0 {
		StartCiEvent()
	}
}

func StartCiEvent() {
	if mob == nil {
		spawnMob()
	}
	if respawns <= 0 {
		msg := "Central Island event finished, thank you for participation"
		MakeAnnouncementx(msg)
		respawns = 4
		isCiRunning = false
		return
	} else {
		time.AfterFunc(time.Second*5, func() {
			if mob.IsDead {
				spawnMob()
			}
			StartCiEvent()
		})
	}

}
func spawnMob() {
	respawns--
	id := 77

	npcPos := GetNPCPosByID(id)
	if npcPos == nil {
		return
	}
	npc, ok := GetNpcInfo(npcPos.NPCID)
	if !ok || npc == nil {
		return
	}

	msg := fmt.Sprintf("%s is roaring.", npc.Name)
	MakeAnnouncementx(msg)

	ai := &AI{ID: len(AIs), HP: npc.MaxHp, Map: 10, PosID: npcPos.ID, RunningSpeed: 10, Server: 1, WalkingSpeed: 5, Once: true, CanAttack: true, Faction: 0, IsDead: false}
	ai.OnSightPlayers = make(map[int]interface{})

	points := []string{"105,401", "259,397", "383,369", "409,239", "339,159", "221,137", "129,119", "99,191", "117,273", "189,339", "293,343", "329,271", "273,195", "181,189", "203,345"}
	min := utils.RandInt(0, int64(len(points)-1))
	max := utils.RandInt(0, int64(len(points)-1))
	for min == max {
		min = utils.RandInt(0, int64(len(points)-1))
		max = utils.RandInt(0, int64(len(points)-1))
	}

	npcPos.MinLocation = points[min]
	npcPos.MaxLocation = points[max]

	minLoc := ConvertPointToLocation(npcPos.MinLocation)
	maxLoc := ConvertPointToLocation(npcPos.MaxLocation)
	loc := utils.Location{X: utils.RandFloat(minLoc.X, maxLoc.X), Y: utils.RandFloat(minLoc.Y, maxLoc.Y)}
	ai.NPCpos = npcPos
	ai.Coordinate = loc.String()
	ai.TargetLocation = *ConvertPointToLocation(ai.Coordinate)
	GenerateIDForAI(ai)
	ai.OnSightPlayers = make(map[int]interface{})
	ai.Handler = ai.AIHandler

	AIsByMap[ai.Server][ai.Map] = append(AIsByMap[ai.Server][ai.Map], ai)
	AIs[ai.ID] = ai
	mob = ai
	go ai.Handler()

}

func MakeAnnouncementx(msg string) {
	length := int16(len(msg) + 3)

	resp := ANNOUNCEMENTa
	resp.SetLength(length)
	resp[6] = byte(len(msg))
	resp.Insert([]byte(msg), 7)

	p := nats.CastPacket{CastNear: false, Data: resp}
	p.Cast()
}
func getHoure(date null.Time) (int, int) {
	if date.Valid {
		hours, minutes, _ := date.Time.Clock()
		return hours, minutes
	}
	return 0, 0
}

func StartYingYang(party *Party) {

	server := 1
	if !IsDungeonClosed {
		IsDungeonClosed = true
	} else {
		msg := messaging.InfoMessage("All dungeons are full at this moment, come back later.")
		party.Leader.Socket.Write(msg)
		return
	}
	party.Leader.Socket.User.ConnectedServer = server
	party.Leader.Socket.User.SelectedServerID = server
	data, _ := party.Leader.ChangeMap(243, nil)
	party.Leader.Socket.Write(data)
	for _, member := range party.Members {

		member.Character.Socket.User.ConnectedServer = server
		member.Character.Socket.User.SelectedServerID = server
		member.IsDungeon = true
		data, _ := member.Character.ChangeMap(243, nil)
		member.Character.Socket.Write(data)
		go StartTimerYingYang(member.Character.Socket, 900)
	}

	go StartTimerYingYang(party.Leader.Socket, YY_TIME_LIMIT)
	go SetDungeonOpenAfterTime(server)
	go CountYingYangMobs(party.Leader.Map)

}
func SetDungeonOpenAfterTime(server int) {
	time.Sleep(time.Minute * time.Duration(YY_TIME_LIMIT))

	IsDungeonClosed = false
}
func StartTimerYingYang(s *Socket, minutes int) {
	resp := DUNGEON_TIMER
	resp.Overwrite(utils.IntToBytes(uint64(minutes*60), 4, true), 6)
	s.Write(DUNGEON_TIMER)

	time.AfterFunc(time.Minute*time.Duration(YY_TIME_LIMIT), func() {
		if s.Character.Map == 243 && s.Character.IsOnline {
			resp := utils.Packet{}
			resp.Concat(messaging.InfoMessage("Your time has ended. Come again when you are stronger. Teleporting to safe zone."))
			coordinate := &utils.Location{X: 37, Y: 453}
			data, _ := s.Character.ChangeMap(17, coordinate)
			resp.Concat(data)
			s.Write(resp)
			s.Character.IsDungeon = false
		}
	})
}

func StartSeasonDungeon(s *Socket) {

	server := 1
	if !IsSeasonDungeon1Closed {
		IsSeasonDungeon1Closed = true
	} else if !IsSeasonDungeon2Closed {
		server = 2
		IsSeasonDungeon2Closed = true
	} else if !IsSeasonDungeon3Closed {
		server = 3
		IsSeasonDungeon3Closed = true
	} else if !IsSeasonDungeon4Closed {
		server = 4
		IsSeasonDungeon4Closed = true
	} else {
		msg := messaging.InfoMessage("All dungeons are full at this moment, come back later. ")
		s.Write(msg)
		return
	}
	s.Character.IsDungeon = true
	s.Character.Socket.User.ConnectedServer = server
	s.User.SelectedServerID = server
	data, _ := s.Character.ChangeMap(212, nil)
	s.Write(data)

	go StartSeasonTimer(s, 212, TIME_LIMIT)
	go CountSeasonCave(server)
	go SetSeasonDungeonOpenAfterTime(server)

}
func SetSeasonDungeonOpenAfterTime(server int) {
	time.Sleep(time.Second * time.Duration(TIME_LIMIT))
	if server == 1 {
		IsSeasonDungeon1Closed = false
	} else if server == 2 {
		IsSeasonDungeon2Closed = false
	} else if server == 3 {
		IsSeasonDungeon3Closed = false
	} else if server == 4 {
		IsSeasonDungeon4Closed = false
	}
}

func StartSeasonTimer(s *Socket, mapID int16, seconds int) {
	resp := DUNGEON_TIMER
	resp.Overwrite(utils.IntToBytes(uint64(seconds), 4, true), 6)
	s.Write(DUNGEON_TIMER)

	time.AfterFunc(time.Second*time.Duration(seconds), func() {
		if s.Character.Map == 212 && s.Character.IsOnline {
			resp := utils.Packet{}
			resp.Concat(messaging.InfoMessage("Your time has ended. Come again when you are stronger. Teleporting to safe zone."))
			data, _ := s.Character.ChangeMap(1, nil)
			resp.Concat(data)
			s.Write(resp)
			s.Character.IsDungeon = false
		}
	})

}

func StartDivineYingYang(party *Party) {

	server := 1
	if !IsDivineDungeonClosed {
		IsDivineDungeonClosed = true
	} else {
		msg := messaging.InfoMessage("All dungeons are full at this moment, come back later. ")
		party.Leader.Socket.Write(msg)
		return
	}
	party.Leader.Socket.User.ConnectedServer = server
	party.Leader.Socket.User.SelectedServerID = server
	data, _ := party.Leader.ChangeMap(215, nil)
	party.Leader.Socket.Write(data)
	for _, member := range party.Members {

		member.Character.Socket.User.ConnectedServer = server
		member.Character.Socket.User.SelectedServerID = server
		member.IsDungeon = true
		data, _ := member.Character.ChangeMap(215, nil)
		member.Character.Socket.Write(data)
		go StartTimerDivineYingYang(member.Character.Socket, 900)
	}

	go StartTimerDivineYingYang(party.Leader.Socket, 1800)
	go SetDivineDungeonOpenAfterTime(server)
	go CountYingYangMobs(party.Leader.Map)

}
func SetDivineDungeonOpenAfterTime(server int) {
	time.Sleep(time.Minute * 30)
	IsDivineDungeonClosed = false

}
func StartTimerDivineYingYang(s *Socket, seconds int) {
	resp := DUNGEON_TIMER
	resp.Overwrite(utils.IntToBytes(uint64(seconds), 4, true), 6)
	s.Write(DUNGEON_TIMER)

	time.AfterFunc(time.Minute*30, func() {
		if s.Character.Map == 243 && s.Character.IsOnline {
			resp := utils.Packet{}
			resp.Concat(messaging.InfoMessage("Your time has ended. Come again when you are stronger. Teleporting to safe zone."))
			coordinate := &utils.Location{X: 513, Y: 467}
			data, _ := s.Character.ChangeMap(24, coordinate)
			resp.Concat(data)
			s.Write(resp)
			s.Character.IsDungeon = false
		}
	})
}

func SpawnRandomNcashBosses() {
	minutes := utils.RandInt(600, 1440)
	posId := int(utils.RandInt(0, int64(len(NcashMobsPosIDs))))
	posId = NcashMobsPosIDs[posId]

	time.AfterFunc(time.Minute*time.Duration(minutes), func() {
		go SpawnNcashBoss("", posId, true)
		go SpawnRandomNcashBosses()
	})
}

func SpawnSeasonBoss() {
	minutes := utils.RandInt(1350, 1440)
	posId := int(utils.RandInt(0, int64(len(SeasonMobsPosIDs))))
	posId = SeasonMobsPosIDs[posId]
	go SpawnNcashBoss("", posId, true)

	time.AfterFunc(time.Minute*time.Duration(minutes), func() {
		go SpawnNcashBoss("", posId, true)
		go SpawnRandomNcashBosses()
	})
}

func SpawnNcashBoss(coordinate string, posId int, announce bool) {
	npcPos := GetNPCPosByID(int(posId))
	if npcPos == nil {
		return
	}
	npc, ok := GetNpcInfo(npcPos.NPCID)
	if !ok {
		return
	}

	ai := &AI{ID: len(AIs), HP: npc.MaxHp, Map: npcPos.MapID, PosID: npcPos.ID, RunningSpeed: 10, Server: 1, WalkingSpeed: 5, Once: true}
	GenerateIDForAI(ai)
	ai.OnSightPlayers = make(map[int]interface{})

	minLoc := ConvertPointToLocation(npcPos.MinLocation)
	maxLoc := ConvertPointToLocation(npcPos.MaxLocation)
	loc := utils.Location{X: utils.RandFloat(minLoc.X, maxLoc.X), Y: utils.RandFloat(minLoc.Y, maxLoc.Y)}
	if coordinate != "" {
		ai.Coordinate = coordinate
	}
	ai.Coordinate = loc.String()
	ai.Handler = ai.AIHandler
	go ai.Handler()

	if announce {
		msg := fmt.Sprintf("%s is roaring.", npc.Name)
		makeAnnouncement(msg)

	}

	AIsByMap[ai.Server][npcPos.MapID] = append(AIsByMap[ai.Server][npcPos.MapID], ai)
	AIs[ai.ID] = ai

	NcashMobs = append(NcashMobs, ai.ID)
}

func StartLoto() {
	time.AfterFunc(time.Hour*3, func() {
		msg := "Lottery event has started. Buy loto ticket by typing /loto. Price : 500nC"
		makeAnnouncement(msg)
		go StartLoto()
		go CountLoto(countDownTimer)
	})
}
func CountLoto(cd int) {
	isActive = true
	if cd >= 120 {
		checkMembersInFactionWarMap()
		msg := fmt.Sprintf("Lottery number will be extracted in %d minutes.", cd/60)
		makeAnnouncement(msg)
		time.AfterFunc(time.Second*60, func() {
			CountLoto(cd - 60)
		})
	} else if cd > 0 {
		checkMembersInFactionWarMap()
		msg := fmt.Sprintf("Lottery number will be extracted in %d seconds.", cd)
		makeAnnouncement(msg)
		time.AfterFunc(time.Second*10, func() {
			CountLoto(cd - 10)
		})
	}
	if cd <= 0 {
		endLoto()
		participants = nil
		isActive = false
	}
}

func endLoto() {
	if len(participants) == 0 {
		return
	}
	if len(participants) < minParticipants {
		for _, participant := range participants {
			user, err := FindUserByID(participant.character.UserID)
			if err != nil {
				return
			} else if user == nil {
				return
			}
			user.NCash += lotoPrice
			user.Update()
			msg := "Lottery event canceled because not enough participants to the event."
			makeAnnouncement(msg)
			msg = "Not enough participants to lottery. The ticket value has been restored to your account."
			participant.character.Socket.Write(messaging.InfoMessage(msg))
		}
	} else {
		rand := int(utils.RandInt(0, 100))
		difference := 100
		winner := &Participant{}
		for _, participant := range participants {
			if int(math.Abs(float64(rand)-float64(participant.number))) < difference {
				difference = int(math.Abs(float64(rand) - float64(participant.number)))
				winner = participant
			}
		}
		user, err := FindUserByID(winner.character.UserID)
		if err != nil {
			return
		} else if user == nil {
			return
		}
		winnings := lotoPrice * uint64(len(participants))
		user.NCash += winnings
		user.Update()
		msg := fmt.Sprintf("Lottery: %s has aquired %d ncash by winning the Lottery.", winner.character.Name, winnings)
		makeAnnouncement(msg)
		msg = fmt.Sprintf("Extracted number : %d. Congratulations, you won the Lottery with number : %d. %d Ncash has been added to your account.", rand, winner.number, winnings)
		winner.character.Socket.Write(messaging.InfoMessage(msg))
		for _, participant := range participants {
			if participant != winner {
				msg = fmt.Sprintf("Extracted number : %d. You chose %d and lost :( Good luck next time.", rand, participant.number)
				participant.character.Socket.Write(messaging.InfoMessage(msg))
			}
		}
	}
}
func AddPlayer(s *Socket, number int) {
	if !isActive {
		msg := "Loto event is not active at this moment."
		s.Write(messaging.InfoMessage(msg))
		return
	}
	if number < 0 || number > 100 {
		msg := "You must choose a number between 0 and 100!"
		s.Write(messaging.InfoMessage(msg))
		return
	}
	for _, participant := range participants {
		if participant.character == s.Character {
			msg := "You already have a ticket. Wait for loto number extraction. Good Luck!"
			s.Write(messaging.InfoMessage(msg))
			return
		}
		if participant.number == number {
			msg := "This number was already picked, choose another number."
			s.Write(messaging.InfoMessage(msg))
			return
		}
	}
	if s.User.NCash < lotoPrice {
		msg := fmt.Sprintf("You don't have enough cash to buy loto ticket. Price : %d", int(lotoPrice))
		s.Write(messaging.InfoMessage(msg))
		return
	} else {
		s.User.NCash -= lotoPrice
		s.User.Update()
		participant := &Participant{
			character: s.Character,
			number:    number,
		}
		participants = append(participants, participant)

		for _, participant := range participants {
			msg := fmt.Sprintf("%s bought a lotery ticket with number :%d", s.Character.Name, number)
			participant.character.Socket.Write(messaging.InfoMessage(msg))
		}
	}
}

func (c *Character) ShowEventsDetails() {
	EVENT_DETAILS := utils.Packet{0xAA, 0x55, 0x0A, 0x00, 0xE7, 0x00, 0x55, 0xAA}
	amount := float64((EXP_RATE))
	EVENT_DETAILS.Insert(utils.FloatToBytes(amount, 4, true), 6)
	time := uint64(EXP_RATE_TIME) //time-seconds
	EVENT_DETAILS.Insert(utils.IntToBytes(time, 4, true), 10)
	c.Socket.Write(EVENT_DETAILS)
	EVENT_DETAILS = utils.Packet{0xAA, 0x55, 0x0A, 0x00, 0xE7, 0x01, 0x55, 0xAA}
	amount = float64((DROP_RATE))
	EVENT_DETAILS.Insert(utils.FloatToBytes(amount, 4, true), 6)
	time = uint64(DROP_RATE_TIME) //time-seconds
	EVENT_DETAILS.Insert(utils.IntToBytes(time, 4, true), 10)
	c.Socket.Write(EVENT_DETAILS)
	EVENT_DETAILS = utils.Packet{0xAA, 0x55, 0x0A, 0x00, 0xE7, 0x02, 0x55, 0xAA}
	amount = float64((GOLD_RATE))
	EVENT_DETAILS.Insert(utils.FloatToBytes(amount, 4, true), 6)
	time = uint64(GOLD_RATE_TIME) //time-seconds
	EVENT_DETAILS.Insert(utils.IntToBytes(time, 4, true), 10)
	c.Socket.Write(EVENT_DETAILS)
}
