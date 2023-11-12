package database

import (
	"fmt"
	"time"

	"github.com/twodragon/kore-server/messaging"
	"github.com/twodragon/kore-server/utils"
)

var (
	TIMER_MENU      = utils.Packet{0xAA, 0x55, 0x08, 0x00, 0x65, 0x03, 0x00, 0x00, 0x55, 0xAA}
	WAR_SCOREPANEL  = utils.Packet{0xAA, 0x55, 0x30, 0x00, 0x65, 0x06, 0x55, 0xAA}
	START_WAR       = utils.Packet{0xaa, 0x55, 0x23, 0x00, 0x65, 0x01, 0x00, 0x00, 0x17, 0x00, 0x00, 0x00, 0x10, 0x27, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0d, 0x00, 0x00, 0x00, 0x10, 0x27, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xb0, 0x04, 0x00, 0x00, 0x55, 0xaa}
	OrderCharacters = make(map[int]*Character)
	ShaoCharacters  = make(map[int]*Character)
	LobbyCharacters = make(map[int]*Character)

	WarRequirePlayers = 10
	OrderPoints       = 10000
	ShaoPoints        = 10000
	CanJoinWar        = false
	WarStarted        = false
	WarStonesIDs      = []uint16{}
	WarStones         = make(map[int]*WarStone)
	ActiveWars        = make(map[int]*ActiveWar)
	Level             int
)

type WarStone struct {
	PseudoID      uint16 `db:"-" json:"-"`
	NpcID         int    `db:"-" json:"-"`
	ConqueredID   int    `db:"-" json:"-"`
	ConquereValue int    `db:"-" json:"-"`
	NearbyZuhang  int    `db:"-" json:"-"`
	NearbyShao    int    `db:"-" json:"-"`
	NearbyZuhangV []int  `db:"-" json:"-"`
	NearbyShaoV   []int  `db:"-" json:"-"`
}
type ActiveWar struct {
	WarID         uint16       `db:"-" json:"-"`
	ZuhangPlayers []*Character `db:"-" json:"-"`
	ShaoPlayers   []*Character `db:"-" json:"-"`
}

func AddPlayerToGreatWar(c *Character) {
	if !CanJoinWar {
		c.Socket.Write(messaging.SystemMessage(messaging.CANNOT_MOVE_TO_AREA))
		return
	}
	if (c.Level < 40 || c.Level > 100) && Level == 40 {
		c.Socket.Write(messaging.SystemMessage(messaging.NO_LEVEL_REQUIREMENT))
		return
	} else if (c.Level < 101 || c.Level > 201) && Level == 101 {
		c.Socket.Write(messaging.SystemMessage(messaging.NO_LEVEL_REQUIREMENT))
		return
	}

	if c.Faction == 1 {
		x := 75.0
		y := 45.0
		c.IsinWar = true
		OrderCharacters[c.ID] = c
		data, _ := c.ChangeMap(230, ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", x, y)))
		c.WarKillCount = 0
		c.WarContribution = 0
		c.Socket.Write(data)
	}
	if c.Faction == 2 {
		x := 81.0
		y := 475.0
		c.IsinWar = true
		ShaoCharacters[c.ID] = c
		data, _ := c.ChangeMap(230, ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", x, y)))
		c.WarKillCount = 0
		c.WarContribution = 0
		c.Socket.Write(data)
	}

}

func StartWarTimer(prepareWarStart int, level int) {
	Level = level
	min, sec := secondsToMinutes(prepareWarStart)
	msg := fmt.Sprintf("%d minutes %d second after the Great War will start.", min, sec)
	msg2 := fmt.Sprintln("Please participate war by Battle Guard")
	makeAnnouncement(msg)
	makeAnnouncement(msg2)
	if prepareWarStart > 0 {
		time.AfterFunc(time.Second*10, func() {
			StartWarTimer(prepareWarStart-10, level)
		})
	} else {
		StartWar()
	}
}
func secondsToMinutes(inSeconds int) (int, int) {
	minutes := inSeconds / 60
	seconds := inSeconds % 60
	return minutes, seconds
}

type countdown struct {
	t int
	d int
	h int
	m int
	s int
}

func JoinToWarLobby(char *Character) {
	LobbyCharacters[char.ID] = char
	warReady := false
	zuhangPlayers := 0
	shaoPlayers := 0
	if len(LobbyCharacters) >= WarRequirePlayers {
		newWar := &ActiveWar{WarID: 1}
		ActiveWars[int(1)] = newWar
		for _, char := range LobbyCharacters {
			if char.Faction == 1 {
				if zuhangPlayers < WarRequirePlayers/2 {
					zuhangPlayers++
					ActiveWars[int(1)].ZuhangPlayers = append(ActiveWars[int(1)].ZuhangPlayers, char)
				}
			} else {
				if shaoPlayers < WarRequirePlayers/2 {
					shaoPlayers++
					ActiveWars[int(1)].ShaoPlayers = append(ActiveWars[int(1)].ShaoPlayers, char)
				}
			}
			if zuhangPlayers >= WarRequirePlayers/2 && shaoPlayers >= WarRequirePlayers/2 {
				warReady = true
				continue
			}
		}
	}
	if warReady {
		for _, char := range ActiveWars[int(1)].ShaoPlayers {
			char.Socket.Write(messaging.InfoMessage(fmt.Sprintln("Your war is ready. /accept war ")))
		}
		for _, char := range ActiveWars[int(1)].ZuhangPlayers {
			char.Socket.Write(messaging.InfoMessage(fmt.Sprintln("Your war is ready. /accept war ")))
		}
		CheckALlPlayersReady()
	}
}

func ResetWar() {
	time.AfterFunc(time.Second*10, func() {
		for _, char := range OrderCharacters {
			delete(OrderCharacters, char.ID)
			char.WarKillCount = 0
			char.WarContribution = 0
			char.IsinWar = false
			gomap, _ := char.ChangeMap(1, nil)
			char.Socket.Write(gomap)
			delete(OrderCharacters, char.ID)
		}
		for _, char := range ShaoCharacters {
			delete(ShaoCharacters, char.ID)
			char.WarKillCount = 0
			char.WarContribution = 0
			char.IsinWar = false
			gomap, _ := char.ChangeMap(1, nil)
			char.Socket.Write(gomap)
			delete(ShaoCharacters, char.ID)
		}
		for _, stones := range WarStones {
			stones.ConquereValue = 0
			stones.ConqueredID = 0
		}
	})
}

func StartWar() {
	resp := START_WAR
	byteOrders := utils.IntToBytes(uint64(len(OrderCharacters)), 4, false)
	byteShaos := utils.IntToBytes(uint64(len(ShaoCharacters)), 4, false)
	resp.Overwrite(byteOrders, 8)
	resp.Overwrite(byteShaos, 22)
	for _, char := range OrderCharacters {
		char.Socket.Write(resp)
	}
	for _, char := range ShaoCharacters {
		char.Socket.Write(resp)
	}

	CanJoinWar = false
	WarStarted = true
	StartInWarTimer()
}

func (selfz *WarStone) RemoveZuhang(id int) {
	for i, other := range selfz.NearbyZuhangV {
		if other == id {
			selfz.NearbyZuhangV = append(selfz.NearbyZuhangV[:i], selfz.NearbyZuhangV[i+1:]...)
			break
		}
	}
}

func (selfs *WarStone) RemoveShao(id int) {
	for i, other := range selfs.NearbyShaoV {
		if other == id {
			selfs.NearbyShaoV = append(selfs.NearbyShaoV[:i], selfs.NearbyShaoV[i+1:]...)
			break
		}
	}
}

func CheckALlPlayersReady() {
	timein := time.Now().Add(time.Minute * 1)
	deadtime := timein.Format(time.RFC3339)

	v, err := time.Parse(time.RFC3339, deadtime)
	if err != nil {
		fmt.Println(err)

	}

	for range time.Tick(1 * time.Second) {
		readyShaoPlayers := 0
		readyZuhangPlayers := 0
		timeRemaining := getTimeRemaining(v)
		if timeRemaining.t <= 0 {
			for _, char := range ActiveWars[int(1)].ShaoPlayers {
				if !char.IsAcceptedWar {
					ActiveWars[int(1)].RemoveLobbyShao(char)
					delete(LobbyCharacters, char.ID)
					char.Socket.Write(messaging.InfoMessage(fmt.Sprintln("You missed the ready check. You kicked from the lobby")))
				}
				char.IsAcceptedWar = false
			}
			for _, char := range ActiveWars[int(1)].ZuhangPlayers {
				if !char.IsAcceptedWar {
					ActiveWars[int(1)].RemoveLobbyZuhang(char)
					delete(LobbyCharacters, char.ID)
					char.Socket.Write(messaging.InfoMessage(fmt.Sprintln("You missed the ready check. You kicked from the lobby")))
				}
				char.IsAcceptedWar = false
			}
			break
		}
		for _, char := range ActiveWars[int(1)].ShaoPlayers {
			if char.IsAcceptedWar {
				readyShaoPlayers++
			}
		}
		for _, char := range ActiveWars[int(1)].ZuhangPlayers {
			if char.IsAcceptedWar {
				readyZuhangPlayers++
			}
		}
		if readyZuhangPlayers >= WarRequirePlayers/2 && readyShaoPlayers >= WarRequirePlayers/2 {
			for _, char := range ActiveWars[int(1)].ZuhangPlayers {
				x := 75.0
				y := 45.0
				char.IsinWar = true
				OrderCharacters[char.ID] = char
				data, _ := char.ChangeMap(230, ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", x, y)))
				char.Socket.Write(data)
			}
			for _, char := range ActiveWars[int(1)].ShaoPlayers {
				x := 81.0
				y := 475.0
				char.IsinWar = true
				ShaoCharacters[char.ID] = char
				data, _ := char.ChangeMap(230, ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", x, y)))
				char.Socket.Write(data)
			}
			StartWarTimer(int(60), Level)
			break
		}
	}
}

func (selfs *ActiveWar) RemoveLobbyShao(char *Character) {
	for i, other := range selfs.ShaoPlayers {
		if other == char {
			selfs.ShaoPlayers = append(selfs.ShaoPlayers[:i], selfs.ShaoPlayers[i+1:]...)
			break
		}
	}
}
func (selfz *ActiveWar) RemoveLobbyZuhang(char *Character) {
	for i, other := range selfz.ZuhangPlayers {
		if other == char {
			selfz.ZuhangPlayers = append(selfz.ZuhangPlayers[:i], selfz.ZuhangPlayers[i+1:]...)
			break
		}
	}
}

func StartInWarTimer() {
	timein := time.Now().Add(time.Minute * 10)
	deadtime := timein.Format(time.RFC3339)

	v, err := time.Parse(time.RFC3339, deadtime)
	if err != nil {
		fmt.Println(err)

	}

	ticker := time.NewTicker(time.Second * 1)
	for range ticker.C {
		//for range time.Tick(1 * time.Second) {
		timeRemaining := getTimeRemaining(v)
		if timeRemaining.t <= 0 || OrderPoints <= 0 || ShaoPoints <= 0 { //|| len(OrderCharacters) == 0 || len(ShaoCharacters) == 0
			WarScorePanelgwar()
			ResetWar()
			break
		}
		timeCounters := CalculateWarCountdown(timeRemaining)
		data := utils.IntToBytes(uint64(timeCounters), 4, true)
		shaoStones := 0
		ZuhangStones := 0
		index := 8
		resp := TIMER_MENU
		byteOrders := utils.IntToBytes(uint64(len(OrderCharacters)), 4, true)
		ordersPoint := utils.IntToBytes(uint64(OrderPoints), 4, true)
		byteShaos := utils.IntToBytes(uint64(len(ShaoCharacters)), 4, true)
		shaoPoint := utils.IntToBytes(uint64(ShaoPoints), 4, true)
		for _, stones := range WarStones {
			if stones.ConqueredID == 1 {
				ShaoPoints -= 2
				ZuhangStones++
			} else if stones.ConqueredID == 2 {
				OrderPoints -= 2
				shaoStones++
			}
		}
		resp.Insert(byteOrders, index)
		index += 4
		resp.Insert(ordersPoint, index)
		index += 4
		resp.Insert([]byte{0x00, 0x00, 0x00, 0x00}, index)
		index += 4
		if ZuhangStones > 0 {
			resp.Insert(utils.IntToBytes(uint64(ZuhangStones), 1, false), index)
			index++
			for _, stones := range WarStones {
				if stones.ConqueredID == 1 {
					resp.Insert(utils.IntToBytes(uint64(stones.NpcID), 4, true), index)
					index += 4
				}
			}
			resp.Insert([]byte{0x00}, index)
			index++
		} else {
			resp.Insert([]byte{0x00, 0x00}, index) //IDE JÖN MAJD HOGY KINEK HÁNY KÖVE VAN
			index += 2
		}
		resp.Insert(byteShaos, index)
		index += 4
		resp.Insert(shaoPoint, index)
		index += 4
		resp.Insert([]byte{0x00, 0x00, 0x00, 0x00}, index)
		index += 4
		if shaoStones >= 1 {
			resp.Insert(utils.IntToBytes(uint64(shaoStones), 1, false), index)
			index++
			for _, stones := range WarStones {
				if stones.ConqueredID == 2 {
					resp.Insert(utils.IntToBytes(uint64(stones.NpcID), 4, true), index)
					index += 4
				}
			}
		} else {
			resp.Insert([]byte{0x00}, index-2)
			index++
		}
		resp.Insert(data, index)
		index += 4
		/*resp.Insert(data2, index)
		index++*/
		length := index - 4
		resp.SetLength(int16(length))
		for _, char := range OrderCharacters {
			if char.IsOnline {
				char.Socket.Write(resp)
			} else {
				delete(OrderCharacters, char.ID)
			}
		}
		for _, char := range ShaoCharacters {
			if char.IsOnline {
				char.Socket.Write(resp)
			} else {
				delete(OrderCharacters, char.ID)
			}
		}
		for _, stones := range WarStones {
			if len(stones.NearbyZuhangV) > len(stones.NearbyShaoV) {
				if stones.ConquereValue > 0 {
					stones.ConquereValue--
				}
				if stones.ConquereValue >= 0 && stones.ConquereValue <= 30 {
					stones.ConqueredID = 1
				} else if stones.ConquereValue > 170 {
					stones.ConqueredID = 0
				}
			} else if len(stones.NearbyShaoV) > len(stones.NearbyZuhangV) {
				if stones.ConquereValue < 200 {
					stones.ConquereValue++
				}
				if stones.ConquereValue >= 170 && stones.ConquereValue <= 200 {
					stones.ConqueredID = 2
				} else if stones.ConquereValue < 30 {
					stones.ConqueredID = 0
				}
			}
		}
	}
}

func WarScorePanelgwar() {
	zuhang_nyert := true
	resp := WAR_SCOREPANEL
	index := 6
	if OrderPoints > ShaoPoints {
		resp.Insert([]byte{0x00, 0x28, 0x00}, index)
	} else {
		resp.Insert([]byte{0x01, 0x28, 0x00}, index)
	}
	if OrderPoints > ShaoPoints {
		zuhang_nyert = true
	}
	index += 3
	for _, char := range OrderCharacters {
		resp.Insert(utils.IntToBytes(uint64(len(char.Name)), 1, false), index)
		index++
		resp.Insert([]byte(char.Name), index)
		index += len(char.Name)
		resp.Insert(utils.IntToBytes(uint64(char.Faction), 1, false), index)
		index++
		data := utils.IntToBytes(uint64(char.WarContribution), 2, true)
		resp.Insert(data, index)
		index += 2
		resp.Insert([]byte{0x00, 0x00}, index)
		index += 2
		data2 := utils.IntToBytes(uint64(char.WarKillCount), 2, true)
		resp.Insert(data2, index)
		index += 2
		resp.Insert([]byte{0x00, 0x00}, index)
		index += 2
		if zuhang_nyert {
			item := &InventorySlot{ItemID: 99009117, Quantity: uint(1)}
			r, _, err := char.AddItem(item, -1, false)
			if err == nil {
				char.Socket.Write(*r)
			}
		} else {
			item := &InventorySlot{ItemID: 99009118, Quantity: uint(1)}
			r, _, err := char.AddItem(item, -1, false)
			if err == nil {
				char.Socket.Write(*r)
			}
		}
	}
	for _, char := range ShaoCharacters {
		resp.Insert(utils.IntToBytes(uint64(len(char.Name)), 1, false), index)
		index++
		resp.Insert([]byte(char.Name), index)
		index += len(char.Name)
		resp.Insert(utils.IntToBytes(uint64(char.Faction), 1, false), index)
		index++
		data := utils.IntToBytes(uint64(char.WarContribution), 2, true)
		resp.Insert(data, index)
		index += 2
		resp.Insert([]byte{0x00, 0x00}, index)
		index += 2
		data2 := utils.IntToBytes(uint64(char.WarKillCount), 2, true)
		resp.Insert(data2, index)
		index += 2
		resp.Insert([]byte{0x00, 0x00}, index)
		index += 2
		if !zuhang_nyert {
			item := &InventorySlot{ItemID: 99009117, Quantity: uint(1)}
			r, _, err := char.AddItem(item, -1, false)
			if err == nil {
				char.Socket.Write(*r)
			}
		} else {
			item := &InventorySlot{ItemID: 99009118, Quantity: uint(1)}
			r, _, err := char.AddItem(item, -1, false)
			if err == nil {
				char.Socket.Write(*r)
			}
		}
	}
	length := index - 4
	resp.SetLength(int16(length))
	for _, char := range ShaoCharacters {
		if char.IsOnline {
			char.Socket.Write(resp)
		} else {
			delete(OrderCharacters, char.ID)
		}
	}
	for _, char := range OrderCharacters {
		if char.IsOnline {
			char.Socket.Write(resp)
		} else {
			delete(OrderCharacters, char.ID)
		}
	}
}
func CalculateResult(number int) []int {
	remaining := number
	divCount := []int{0, 0, 0, 0}
	divNumbers := []int{1, 16, 256, 4096, 65536, 1048576}
	for i := len(divNumbers) - 1; i >= 0; i-- {
		if remaining < divNumbers[i] || remaining == 0 {
			continue
		}
		test := remaining / divNumbers[i]
		if test > 15 {
			test = 15
		}
		divCount[i] = test
		test2 := test * divNumbers[i]
		remaining -= test2
	}
	return divCount
}

func CalculateWarCountdown(time countdown) int {
	//remaining := time.t
	return time.t
}

/*func divmod(numerator, denominator int64) (quotient, remainder int64) {
	quotient = numerator / denominator // integer division, decimals are truncated
	remainder = numerator % denominator
	return
}*/

func getTimeRemaining(t time.Time) countdown {
	currentTime := time.Now()
	difference := t.Sub(currentTime)

	total := int(difference.Seconds())
	days := int(total / (60 * 60 * 24))
	hours := int(total / (60 * 60) % 24)
	minutes := int(total/60) % 60
	seconds := int(total % 60)
	return countdown{
		t: total,
		d: days,
		h: hours,
		m: minutes,
		s: seconds,
	}
}
