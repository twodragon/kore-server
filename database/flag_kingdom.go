package database

import (
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	"github.com/twodragon/kore-server/messaging"
	"github.com/twodragon/kore-server/utils"

	null "gopkg.in/guregu/null.v3"
)

var (
	zhuangFlagKingdomMembersList []*Character
	shaoFlagKingdomMembersList   []*Character
	zhuangFlagKingdomPoints      int
	shaoFlagKingdomPoints        int
	zhuangFlagKingdomFlags       int
	shaoFlagKingdomFlags         int
	timingFlagKingdom            int
	isFlagKingdomEntranceActive  bool
	isFlagKingdomStarted         bool
	minLevel                     int
	maxLevel                     int
)

func PrepareFlagKingdom(countdown int) {
	go startFlagKingdomCounting(countdown)
	minLevel = 40
	maxLevel = 100

}

func startFlagKingdomCounting(cd int) {
	isFlagKingdomEntranceActive = true
	if cd >= 120 {
		checkMembersInFlagKingdomMap()
		msg := fmt.Sprintf("Flag Kingdom level 40-100 will start in %d minutes. Enter Flag Kingdom at Hero Battle Manager", cd/60)
		makeAnnouncement(msg)
		time.AfterFunc(time.Second*60, func() {
			startFlagKingdomCounting(cd - 60)
		})
	} else if cd > 0 {
		checkMembersInFlagKingdomMap()
		msg := fmt.Sprintf("Flag Kingdom level 40-100 will start in %d seconds.", cd)
		makeAnnouncement(msg)
		time.AfterFunc(time.Second*10, func() {
			startFlagKingdomCounting(cd - 10)
		})
	}
	if cd <= 0 {
		StartFlagKingdom()
		isFlagKingdomEntranceActive = false
	}
}

func StartFlagKingdom() {

	checkMembersInFlagKingdomMap()

	if len(zhuangFlagKingdomMembersList) < 3 {
		msg := "Not enough participants to start Faction war"
		makeAnnouncement(msg)
		return
	} else if len(shaoFlagKingdomMembersList) < 3 {
		msg := "Not enough participants to start Faction war"
		makeAnnouncement(msg)
		return
	}

	resp := FACTION_WAR_START
	timingFlagKingdom = 600
	isFlagKingdomStarted = true

	resp.Overwrite(utils.IntToBytes(uint64(len(zhuangFlagKingdomMembersList)), 4, true), 8) //Zhuang numbers
	resp.Overwrite(utils.IntToBytes(uint64(zhuangFlagKingdomPoints), 4, true), 12)          //Zhuang points
	resp.Overwrite(utils.IntToBytes(uint64(len(shaoFlagKingdomMembersList)), 4, true), 22)  //Shao number
	resp.Overwrite(utils.IntToBytes(uint64(shaoFlagKingdomPoints), 4, true), 26)            //Shao points
	resp.Overwrite(utils.IntToBytes(uint64(timingFlagKingdom), 4, true), 35)                //Time

	updateFlagKingdomBar()
}
func updateFlagKingdomBar() {

	if timingFlagKingdom <= 0 {
		return
	}

	checkMembersInFlagKingdomMap()

	for _, c := range zhuangFlagKingdomMembersList {
		if c == nil {
			return
		}
		resp := FACTION_WAR_UPDATE
		resp.Overwrite(utils.IntToBytes(uint64(len(zhuangFlagKingdomMembersList)), 4, true), 7) //Zhuang numbers
		resp.Overwrite(utils.IntToBytes(uint64(zhuangFlagKingdomPoints), 4, true), 11)          //Zhuang points
		resp.Overwrite(utils.IntToBytes(uint64(zhuangFlagKingdomFlags), 4, true), 15)           //Zhuang flags
		resp.Overwrite(utils.IntToBytes(uint64(len(shaoFlagKingdomMembersList)), 4, true), 21)  //Shao number///////////////
		resp.Overwrite(utils.IntToBytes(uint64(shaoFlagKingdomPoints), 4, true), 25)            //Shao points
		resp.Overwrite(utils.IntToBytes(uint64(shaoFlagKingdomFlags), 4, true), 29)             //Shao flags
		resp.Overwrite(utils.IntToBytes(uint64(timingFlagKingdom), 4, true), 34)                //Time
		c.Socket.Write(resp)
	}
	for _, c := range shaoFlagKingdomMembersList {
		if c == nil {
			return
		}
		resp := FACTION_WAR_UPDATE
		resp.Overwrite(utils.IntToBytes(uint64(len(zhuangFlagKingdomMembersList)), 4, true), 7) //Zhuang numbers
		resp.Overwrite(utils.IntToBytes(uint64(zhuangFlagKingdomPoints), 4, true), 11)          //Zhuang points
		resp.Overwrite(utils.IntToBytes(uint64(zhuangFlagKingdomFlags), 4, true), 15)           //Zhuang flags
		resp.Overwrite(utils.IntToBytes(uint64(len(shaoFlagKingdomMembersList)), 4, true), 21)  //Shao number///////////////
		resp.Overwrite(utils.IntToBytes(uint64(shaoFlagKingdomPoints), 4, true), 25)            //Shao points
		resp.Overwrite(utils.IntToBytes(uint64(shaoFlagKingdomFlags), 4, true), 29)             //Shao flags
		resp.Overwrite(utils.IntToBytes(uint64(timingFlagKingdom), 4, true), 34)                //Time
		c.Socket.Write(resp)
	}

	timingFlagKingdom--
	if timingFlagKingdom <= 0 {
		finishFlagKingdom()
		return
	}
	time.AfterFunc(time.Second*2, func() {
		updateFlagKingdomBar()
	})
}

func AddPointsToFlagKingdomFaction(points int, faction int) {
	if faction == 1 {
		zhuangFlagKingdomPoints += points
		return
	}
	shaoFlagKingdomPoints += points
}
func AddFlagToFlagKingdomFaction(c *Character, itemID int64) {
	faction := c.Faction
	if faction == 1 {
		zhuangFlagKingdomFlags++
		if itemID == 99059990 {
			AddPointsToFlagKingdomFaction(50, faction)
			c.WarContribution += 50
		} else if itemID == 99059991 {
			AddPointsToFlagKingdomFaction(100, faction)
			c.WarContribution += 100
		} else if itemID == 99059992 {
			AddPointsToFlagKingdomFaction(150, faction)
			c.WarContribution += 150
		}
		for _, c := range zhuangFlagKingdomMembersList {
			c.Socket.Write(messaging.SystemMessage(32041)) //Chaos captured the Flag.
		}
		for _, c := range shaoFlagKingdomMembersList {
			c.Socket.Write(messaging.SystemMessage(32041)) //Order captured the Flag.
		}
		return
	}
	shaoFlagKingdomFlags++
	if itemID == 99059992 {
		AddPointsToFlagKingdomFaction(50, faction)
		c.WarContribution += 50
	} else if itemID == 99059991 {
		AddPointsToFlagKingdomFaction(100, faction)
		c.WarContribution += 100
	} else if itemID == 99059990 {
		AddPointsToFlagKingdomFaction(150, faction)
		c.WarContribution += 150
	}
	for _, c := range zhuangFlagKingdomMembersList {
		c.Socket.Write(messaging.SystemMessage(32040)) //Chaos captured the Flag.
	}
	for _, c := range shaoFlagKingdomMembersList {
		c.Socket.Write(messaging.SystemMessage(32040)) //Chaos captured the Flag.
	}

}

func FactionCapturedFlagNotification() {
	for _, c := range shaoFlagKingdomMembersList {
		c.Socket.Write(messaging.SystemMessage(32038)) //Flag has be taken
	}
	for _, c := range zhuangFlagKingdomMembersList {
		c.Socket.Write(messaging.SystemMessage(32038)) //Flag has be taken
	}
}

func IsFlagKingdomEntranceActive() bool {
	return isFlagKingdomEntranceActive
}
func IsFlagKingdomStarted() bool {
	return isFlagKingdomStarted
}

func AddMemberToFlagKingdom(char *Character) {
	if !isFlagKingdomEntranceActive {
		return
	}
	if char.Level < 40 || char.Level > 100 {
		return
	}
	checkMembersInFlagKingdomMap()

	for _, player := range zhuangFlagKingdomMembersList {
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
	for _, player := range shaoFlagKingdomMembersList {

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
	coordinate := &utils.Location{X: 439, Y: 51}
	data, _ := char.ChangeMap(249, coordinate)
	if char.Faction == 2 {
		coordinate := &utils.Location{X: 67, Y: 481}
		data, _ = char.ChangeMap(249, coordinate)
	}
	char.IsinWar = true
	char.Socket.Write(data)
}

func finishFlagKingdom() {
	isFlagKingdomStarted = false
	isFlagKingdomEntranceActive = false

	if zhuangFlagKingdomPoints > shaoFlagKingdomPoints { //zhuang won
		msg := "Zhuang faction won the Flag Kingdom!"
		makeAnnouncement(msg)

		for _, c := range zhuangFlagKingdomMembersList { //give item to all zhuangs
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
		for _, c := range shaoFlagKingdomMembersList { //give item to all shaos
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
		msg := "Shao faction won the Flag Kingdom!"
		makeAnnouncement(msg)

		for _, c := range zhuangFlagKingdomMembersList { //give item to all zhuangs
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

			data, _ := c.ChangeMap(1, nil)
			c.IsinWar = false
			c.Socket.Write(data)
			infection := BuffInfections[60029]
			if infection != nil {
				c.AddBuff(infection, 7200)
			}
		}
		for _, c := range shaoFlagKingdomMembersList { //give item to all shaos
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

			data, _ := c.ChangeMap(1, nil)
			c.IsinWar = false
			c.Socket.Write(data)
			c.Socket.Stats.Honor += 50
			c.Socket.Stats.Update()
			c.Socket.Write(messaging.InfoMessage("You acquired 50 Honor points."))
			stat, _ := c.GetStats()
			c.Socket.Write(stat)
			infection := BuffInfections[60028]
			if infection != nil {
				c.AddBuff(infection, 7200)
			}
		}
	}

	zhuangFlagKingdomPoints = 0
	shaoFlagKingdomPoints = 0
	zhuangFlagKingdomFlags = 0
	shaoFlagKingdomFlags = 0

	WarScorePanel(zhuangFlagKingdomMembersList, shaoFlagKingdomMembersList, zhuangFlagKingdomPoints, shaoFlagKingdomPoints)

}
func checkMembersInFlagKingdomMap() {
	zhuangFlagKingdomMembersList = nil
	shaoFlagKingdomMembersList = nil
	for _, member := range FindCharactersInMap(249) {
		if member.Faction == 1 {
			zhuangFlagKingdomMembersList = append(zhuangFlagKingdomMembersList, member)
			member.IsinWar = true
		}
		if member.Faction == 2 {
			shaoFlagKingdomMembersList = append(shaoFlagKingdomMembersList, member)
			member.IsinWar = true
		}
	}
}
func FlagKingdomSchedule() {
	if !isFlagKingdomEntranceActive && !isFlagKingdomStarted {
		hour, minutes := getHour(null.NewTime(time.Now(), true))
		if hour == 20 && minutes == 0 {
			CanJoinWar = true
			PrepareFlagKingdom(600)
		}

	}
	time.AfterFunc(time.Minute, func() {
		FlagKingdomSchedule()
	})
}
func DropFlag(c *Character, flag int64) {
	baseLocation := ConvertPointToLocation(c.Coordinate)

	drop := NewSlot()
	drop.ItemID = flag
	drop.Quantity = 1
	drop.Plus = 0
	dr := &Drop{Server: 1, Map: 249, Claimer: nil, Item: drop,
		Location: *baseLocation}

	dr.GenerateIDForDrop(1, 249)

	dropID := uint16(dr.ID)

	time.AfterFunc(DROP_LIFETIME, func() { // remove drop after timeout
		RemoveDrop(1, 249, dropID)
	})
}

func WarScorePanel(zhuangs []*Character, shaos []*Character, zhuangScore int, shaoScore int) {
	resp := WAR_SCOREPANEL
	index := 6
	fmt.Println("OrderPoints: ", zhuangScore, " ShaoPoints: ", shaoScore)
	if zhuangScore > shaoScore {
		resp.Insert([]byte{0x00, 0x28, 0x00}, index)
	} else {
		resp.Insert([]byte{0x01, 0x28, 0x00}, index)
	}
	index += 3
	for _, char := range zhuangs {
		resp.Insert(utils.IntToBytes(uint64(len(char.Name)), 1, false), index)
		index++
		resp.Insert([]byte(char.Name), index)
		index += len(char.Name)
		resp.Insert(utils.IntToBytes(uint64(char.Faction), 1, false), index)
		index++
		data := utils.IntToBytes(uint64(char.WarContribution), 3, true)
		resp.Insert(data, index)
		index += 3
		resp.Insert([]byte{0x00}, index)
		index++
		data2 := utils.IntToBytes(uint64(char.WarKillCount), 3, true)
		resp.Insert(data2, index)
		index += 3
		resp.Insert([]byte{0x00}, index)
		index++
	}
	for _, char := range shaos {
		resp.Insert(utils.IntToBytes(uint64(len(char.Name)), 1, false), index)
		index++
		resp.Insert([]byte(char.Name), index)
		index += len(char.Name)
		resp.Insert(utils.IntToBytes(uint64(char.Faction), 1, false), index)
		index++
		data := utils.IntToBytes(uint64(char.WarContribution), 3, true)
		resp.Insert(data, index)
		index += 3
		resp.Insert([]byte{0x00}, index)
		index++
		data2 := utils.IntToBytes(uint64(char.WarKillCount), 3, true)
		resp.Insert(data2, index)
		index += 3
		resp.Insert([]byte{0x00}, index)
		index++
	}
	resp.SetLength(int16(binary.Size(resp) - 6))

	for _, char := range shaos {
		char.WarContribution = 0
		char.Socket.Write(resp)
	}
	for _, char := range zhuangs {
		char.WarContribution = 0
		char.Socket.Write(resp)
	}

}
func FlagKingdomWarSchedule() {
	if !isFlagKingdomEntranceActive && !isFlagKingdomStarted {
		if time.Now().Day()%2 != 0 {
			hour, minutes := getHour(null.NewTime(time.Now(), true))
			if hour == 20 && minutes == 0 {
				CanJoinWar = true
				PrepareFlagKingdom(600)
			}
		}
	}
	time.AfterFunc(time.Minute, func() {
		FactionWarSchedule()
	})
}
