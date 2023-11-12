package database

import (
	"fmt"
	"strings"
	"time"

	"github.com/twodragon/kore-server/nats"

	"github.com/twodragon/kore-server/messaging"
	"github.com/twodragon/kore-server/utils"

	null "gopkg.in/guregu/null.v3"
)

var (
	ANNOUNCEMENT      = utils.Packet{0xAA, 0x55, 0x00, 0x00, 0x71, 0x06, 0x00, 0x55, 0xAA}
	FACTION_WAR_START = utils.Packet{
		0xAA, 0x55, 0x23, 0x00, 0x65, 0x01, 0x0a, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xaa}
	FACTION_WAR_UPDATE = utils.Packet{
		0xAA, 0x55, 0x23, 0x00, 0x65, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x55, 0xaa}
	fw_zhuangFactionWarMembersList []*Character
	fw_shaoFactionWarMembersList   []*Character
	fw_zhuangFactionWarPoints      int
	fw_shaoFactionWarPoints        int
	fw_timingFactionWar            int
	fw_isFactionWarEntranceActive  bool
	fw_isFactionWarStarted         bool
	fw_minLevel                    int
	fw_maxLevel                    int
)

func PrepareFactionWar(countdown int) {
	go startFactionWarCounting(countdown)
	fw_minLevel = 40
	fw_maxLevel = 100

}

func startFactionWarCounting(cd int) {
	fw_isFactionWarEntranceActive = true
	if cd >= 120 {
		checkMembersInFactionWarMap()
		msg := fmt.Sprintf("Faction war level 40-100 will start in %d minutes. Enter faction war at Hero Battle Manager", cd/60)
		makeAnnouncement(msg)
		time.AfterFunc(time.Second*60, func() {
			startFactionWarCounting(cd - 60)
		})
	} else if cd > 0 {
		checkMembersInFactionWarMap()
		msg := fmt.Sprintf("Faction war level 40-100 will start in %d seconds.", cd)
		makeAnnouncement(msg)
		time.AfterFunc(time.Second*10, func() {
			startFactionWarCounting(cd - 10)
		})
	}
	if cd <= 0 {
		StartFactionWar()
		fw_isFactionWarEntranceActive = false
	}
}

func StartFactionWar() {

	checkMembersInFactionWarMap()

	if len(fw_zhuangFactionWarMembersList) < 3 {
		msg := "Not enough participants to start war"
		makeAnnouncement(msg)
		return
	} else if len(fw_shaoFactionWarMembersList) < 3 {
		msg := "Not enough participants to start war"
		makeAnnouncement(msg)
		return
	}

	resp := FACTION_WAR_START
	fw_timingFactionWar = 600
	fw_isFactionWarStarted = true

	resp.Overwrite(utils.IntToBytes(uint64(len(fw_zhuangFactionWarMembersList)), 4, true), 8) //Zhuang numbers
	resp.Overwrite(utils.IntToBytes(uint64(fw_zhuangFactionWarPoints), 4, true), 12)          //Zhuang points
	resp.Overwrite(utils.IntToBytes(uint64(len(fw_shaoFactionWarMembersList)), 4, true), 22)  //Shao number
	resp.Overwrite(utils.IntToBytes(uint64(fw_shaoFactionWarPoints), 4, true), 26)            //Shao points
	resp.Overwrite(utils.IntToBytes(uint64(fw_timingFactionWar), 4, true), 35)                //Time

	updateFactionWarBar()
}
func updateFactionWarBar() {

	if fw_timingFactionWar <= 0 {
		return
	}

	checkMembersInFactionWarMap()

	for _, c := range fw_zhuangFactionWarMembersList {
		if c == nil {
			return
		}
		resp := FACTION_WAR_UPDATE
		resp.Overwrite(utils.IntToBytes(uint64(len(fw_zhuangFactionWarMembersList)), 4, true), 7) //Zhuang numbers
		resp.Overwrite(utils.IntToBytes(uint64(fw_zhuangFactionWarPoints), 4, true), 11)          //Zhuang points
		resp.Overwrite(utils.IntToBytes(uint64(len(fw_shaoFactionWarMembersList)), 4, true), 21)  //Shao number///////////////
		resp.Overwrite(utils.IntToBytes(uint64(fw_shaoFactionWarPoints), 4, true), 25)            //Shao points
		resp.Overwrite(utils.IntToBytes(uint64(fw_timingFactionWar), 4, true), 34)                //Time
		c.Socket.Write(resp)
	}
	for _, c := range fw_shaoFactionWarMembersList {
		if c == nil {
			return
		}
		resp := FACTION_WAR_UPDATE
		resp.Overwrite(utils.IntToBytes(uint64(len(fw_zhuangFactionWarMembersList)), 4, true), 7) //Zhuang numbers
		resp.Overwrite(utils.IntToBytes(uint64(fw_zhuangFactionWarPoints), 4, true), 11)          //Zhuang points
		resp.Overwrite(utils.IntToBytes(uint64(len(fw_shaoFactionWarMembersList)), 4, true), 21)  //Shao number///////////////
		resp.Overwrite(utils.IntToBytes(uint64(fw_shaoFactionWarPoints), 4, true), 25)            //Shao points
		resp.Overwrite(utils.IntToBytes(uint64(fw_timingFactionWar), 4, true), 34)                //Time
		c.Socket.Write(resp)
	}

	AddPointsToFactionWarFaction(len(fw_zhuangFactionWarMembersList), 1)
	AddPointsToFactionWarFaction(len(fw_shaoFactionWarMembersList), 2)

	fw_timingFactionWar--
	if fw_timingFactionWar <= 0 {
		finishFactionWar()
		return
	}
	time.AfterFunc(time.Second*2, func() {
		updateFactionWarBar()
	})
}

func AddPointsToFactionWarFaction(points int, faction int) {
	if faction == 1 {
		fw_zhuangFactionWarPoints += points
		return
	}
	fw_shaoFactionWarPoints += points
}

func IsFactionWarEntranceActive() bool {
	return fw_isFactionWarEntranceActive
}
func IsFactionWarStarted() bool {
	return fw_isFactionWarStarted
}

func AddMemberToFactionWar(char *Character) {
	if !fw_isFactionWarEntranceActive {
		return
	}
	if char.Level < 40 || char.Level > 100 {
		return
	}
	checkMembersInFactionWarMap()

	for _, player := range fw_zhuangFactionWarMembersList {
		user, err := FindUserByID(player.UserID)
		if err != nil {
			continue
		}
		user2, err := FindUserByID(char.UserID)
		if err != nil {
			return
		}
		ip1 := strings.Split(user.ConnectedIP, ":")
		ip1x := ip1[0]
		ip2 := strings.Split(user2.ConnectedIP, ":")
		ip2x := ip2[0]

		if ip1x == ip2x {
			char.Socket.Write(messaging.InfoMessage("You cannot enter with more than one character!"))
			return
		}
	}
	for _, player := range fw_shaoFactionWarMembersList {

		user, err := FindUserByID(player.UserID)
		if err != nil {
			continue
		}
		user2, err := FindUserByID(char.UserID)
		if err != nil {
			return
		}
		ip1 := strings.Split(user.ConnectedIP, ":")
		ip1x := ip1[0]
		ip2 := strings.Split(user2.ConnectedIP, ":")
		ip2x := ip2[0]

		if ip1x == ip2x {
			char.Socket.Write(messaging.InfoMessage("You cannot enter with more than one character!"))
			return
		}

	}
	coordinate := &utils.Location{X: 325, Y: 465}
	data, _ := char.ChangeMap(255, coordinate)
	if char.Faction == 2 {
		coordinate := &utils.Location{X: 179, Y: 45}
		data, _ = char.ChangeMap(255, coordinate)
	}
	char.IsinWar = true
	char.Socket.Write(data)
}

func finishFactionWar() {
	fw_isFactionWarStarted = false
	fw_isFactionWarEntranceActive = false
	WarScorePanel(fw_zhuangFactionWarMembersList, fw_shaoFactionWarMembersList, fw_zhuangFactionWarPoints, fw_shaoFactionWarPoints)

	if fw_zhuangFactionWarPoints > fw_shaoFactionWarPoints { //zhuang won
		msg := "Zhuang faction won the faction war!"
		makeAnnouncement(msg)

		for _, c := range fw_zhuangFactionWarMembersList { //give item to all zhuangs
			if c == nil {
				return
			}
			resp := utils.Packet{}

			item := &InventorySlot{ItemID: 99009117, Quantity: uint(1)}
			r, _, err := c.AddItem(item, -1, false)
			if err == nil && c.Socket != nil {
				resp.Concat(*r)
			}
			item = &InventorySlot{ItemID: 100080294, Quantity: uint(1)}
			r, _, err = c.AddItem(item, -1, false)
			if err == nil && c.Socket != nil {
				resp.Concat(*r)
			}
			item = &InventorySlot{ItemID: 92000063, Quantity: uint(1)}
			r, _, err = c.AddItem(item, -1, false)
			if err == nil && c.Socket != nil {
				resp.Concat(*r)
			}

			c.Socket.Stats.Honor += 50
			c.Socket.Stats.Update()
			c.Socket.Write(messaging.InfoMessage("You acquired 50 Honor points."))
			stat, _ := c.GetStats()
			c.Socket.Write(stat)

		}
		for _, c := range fw_shaoFactionWarMembersList { //give item to all shaos
			if c == nil {
				return
			}
			resp := utils.Packet{}
			item := &InventorySlot{ItemID: 99009118, Quantity: uint(1)}
			r, _, err := c.AddItem(item, -1, false)
			if err == nil && c.Socket != nil {
				resp.Concat(*r)
			}
			item = &InventorySlot{ItemID: 100080294, Quantity: uint(1)}
			r, _, err = c.AddItem(item, -1, false)
			if err == nil && c.Socket != nil {
				resp.Concat(*r)
			}
			item = &InventorySlot{ItemID: 92000063, Quantity: uint(1)}
			r, _, err = c.AddItem(item, -1, false)
			if err == nil && c.Socket != nil {
				resp.Concat(*r)
			}

		}

	} else { // shao won
		msg := "Shao faction won the faction war!"
		makeAnnouncement(msg)
		for _, c := range fw_zhuangFactionWarMembersList { //give item to all zhuangs
			if c == nil {
				return
			}
			resp := utils.Packet{}
			item := &InventorySlot{ItemID: 99009118, Quantity: uint(1)}
			r, _, err := c.AddItem(item, -1, false)
			if err == nil && c.Socket != nil {
				resp.Concat(*r)
			}
			item = &InventorySlot{ItemID: 100080294, Quantity: uint(1)}
			r, _, err = c.AddItem(item, -1, false)
			if err == nil && c.Socket != nil {
				resp.Concat(*r)
			}
			item = &InventorySlot{ItemID: 92000063, Quantity: uint(1)}
			r, _, err = c.AddItem(item, -1, false)
			if err == nil && c.Socket != nil {
				resp.Concat(*r)
			}

			c.IsinWar = false

		}
		for _, c := range fw_shaoFactionWarMembersList { //give item to all shaos
			if c == nil {
				return
			}
			resp := utils.Packet{}
			item := &InventorySlot{ItemID: 99009117, Quantity: uint(1)}
			r, _, err := c.AddItem(item, -1, false)
			if err == nil && c.Socket != nil {
				resp.Concat(*r)
			}
			item = &InventorySlot{ItemID: 100080294, Quantity: uint(1)}
			r, _, err = c.AddItem(item, -1, false)
			if err == nil && c.Socket != nil {
				resp.Concat(*r)
			}
			item = &InventorySlot{ItemID: 92000063, Quantity: uint(1)}
			r, _, err = c.AddItem(item, -1, false)
			if err == nil && c.Socket != nil {
				resp.Concat(*r)
			}

			c.IsinWar = false
			c.Socket.Stats.Honor += 50
			c.Socket.Stats.Update()
			c.Socket.Write(messaging.InfoMessage("You acquired 50 Honor points."))
			stat, _ := c.GetStats()
			c.Socket.Write(stat)

		}
	}
	fw_zhuangFactionWarPoints = 0
	fw_shaoFactionWarPoints = 0
}
func checkMembersInFactionWarMap() {
	fw_zhuangFactionWarMembersList = nil
	fw_shaoFactionWarMembersList = nil
	for _, member := range FindCharactersInMap(255) {
		if member.Faction == 1 {
			fw_zhuangFactionWarMembersList = append(fw_zhuangFactionWarMembersList, member)
			member.IsinWar = true
		}
		if member.Faction == 2 {
			fw_shaoFactionWarMembersList = append(fw_shaoFactionWarMembersList, member)
			member.IsinWar = true
		}
	}
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
func getHour(date null.Time) (int, int) {
	if date.Valid {
		hours, minutes, _ := date.Time.Clock()
		return hours, minutes
	}
	return 0, 0
}
func FactionWarSchedule() {
	if !fw_isFactionWarEntranceActive && !fw_isFactionWarStarted {
		if time.Now().Day()%2 == 0 {
			hour, minutes := getHour(null.NewTime(time.Now(), true))
			if hour == 20 && minutes == 0 {
				CanJoinWar = true
				PrepareFactionWar(600)
			}
		}
	}
	time.AfterFunc(time.Minute, func() {
		FactionWarSchedule()
	})
}
