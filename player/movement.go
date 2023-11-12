package player

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/twodragon/kore-server/database"
	"github.com/twodragon/kore-server/messaging"
	"github.com/twodragon/kore-server/nats"
	"github.com/twodragon/kore-server/utils"
)

type MovementHandler struct {
}

var (
	CHARACTER_MOVEMENT = utils.Packet{0xAA, 0x55, 0x22, 0x00, 0x22, 0x01, 0x00, 0x00, 0x00, 0x00, 0xC8, 0xB0, 0xFE, 0xBE, 0x00, 0x00, 0x55, 0xAA}
)

func (h *MovementHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	c := s.Character
	if c == nil {
		return nil, nil
	}
	if c.Stunned {
		resp := messaging.SystemMessage(10025)
		point := database.ConvertPointToLocation(c.Coordinate)
		teleportData := c.Teleport(point)
		s.Write(teleportData)
		return resp, nil
	}
	if !c.CanMove {
		msg := "Unequip unusable items to be able to move."
		s.Write(messaging.InfoMessage(msg))
		resp, _ := c.ChangeMap(1, nil)
		return resp, nil
	}

	if !c.IsActive {
		c.IsActive = true
		go c.ActivityStatus(0)
	}
	if c.Respawning {
		return nil, nil
	}

	if len(data) < 26 {
		return nil, nil
	}

	if c.Map == 255 && database.IsFactionWarEntranceActive() {
		parts := strings.Split(c.Coordinate, ",")
		y := strings.Trim(parts[1], ")")
		py := strings.Split(y, ".")
		y = py[0]
		posY, _ := strconv.ParseInt(y, 10, 64)
		if c.Faction == 1 && posY < 440 {
			coordinate := &utils.Location{X: 325, Y: 465}
			teleportData := c.Teleport(coordinate)
			s.Write(teleportData)
		}
		if c.Faction == 2 && posY > 80 {
			coordinate := &utils.Location{X: 325, Y: 465}
			teleportData := c.Teleport(coordinate)
			s.Write(teleportData)
		}
	}

	movType := utils.BytesToInt(data[4:6], false)
	speed := float64(0.0)
	movMasodik := byte(0x00)
	if c.Map == 249 {

		coordinate := database.ConvertPointToLocation(c.Coordinate)
		if database.IsFlagKingdomEntranceActive() {
			if c.Faction == 1 {
				if coordinate.X < 400 || coordinate.Y > 127 {
					coordinate := &utils.Location{X: 439, Y: 51}
					teleportData := c.Teleport(coordinate)
					s.Write(teleportData)
				}
			}
			if c.Faction == 2 && (coordinate.X > 110 || coordinate.Y < 400) {
				coordinate := &utils.Location{X: 67, Y: 481}
				teleportData := c.Teleport(coordinate)
				s.Write(teleportData)
			}
		} else {
			if c.Faction == 1 && (coordinate.X < 91 && coordinate.Y > 423) {
				coordinate := &utils.Location{X: 105, Y: 405}
				teleportData := c.Teleport(coordinate)
				s.Write(teleportData)
			}
			if c.Faction == 2 && (coordinate.X > 417 && coordinate.Y < 103) {
				coordinate := &utils.Location{X: 405, Y: 111}
				teleportData := c.Teleport(coordinate)
				s.Write(teleportData)
			}
		}
		_, item, err := c.FindItemInInventory(nil, 99059990, 99059991, 99059992)
		if item != nil && err == nil && movType != 8705 {
			point := database.ConvertPointToLocation(c.Coordinate)
			teleportData := c.Teleport(point)
			s.Write(teleportData)
			s.Write(messaging.SystemMessage(32039)) //Flag only walk message.
			return nil, nil
		}
	}
	if !s.Character.IsAllowedInMap(s.Character.Map) {
		gomap, _ := s.Character.ChangeMap(1, nil)
		s.Write(gomap)
	}

	if movType == 8705 { // movement
		speed = 5.6
		movMasodik = data[5]
	} else if movType == 8706 || movType == 9732 { // running or flying
		speed = c.RunningSpeed + c.Socket.Stats.AdditionalRunningSpeed
		movMasodik = data[5]
	}

	if c.IsMounting {
		speed = 500.5
		movMasodik = byte(0x01)
	}

	resp := CHARACTER_MOVEMENT
	resp.Insert(utils.IntToBytes(uint64(s.Character.PseudoID), 2, true), 5) // character pseudo id

	coordinate := &utils.Location{X: utils.BytesToFloat(data[6:10], true), Y: utils.BytesToFloat(data[10:14], true)}
	target := &utils.Location{X: utils.BytesToFloat(data[18:22], true), Y: utils.BytesToFloat(data[22:26], true)}
	distance := utils.CalculateDistance(coordinate, target)

	resp[4] = data[4]
	resp[7] = movMasodik //data[5]            // running mode

	resp.Insert(utils.FloatToBytes(coordinate.X, 4, true), 8) // coordinate-x
	resp.Insert(utils.FloatToBytes(coordinate.Y, 4, true), 12)

	resp.Insert(utils.FloatToBytes(target.X, 4, true), 20) // coordinate-x
	resp.Insert(utils.FloatToBytes(target.Y, 4, true), 24)

	resp.Insert(utils.FloatToBytes(speed, 4, true), 32) // speed

	p := &nats.CastPacket{CastNear: true, CharacterID: s.Character.ID, Data: resp, Type: nats.PLAYER_MOVEMENT}
	err := p.Cast()
	if err != nil {
		return nil, err
	}

	c.SetCoordinate(coordinate)
	token := utils.RandInt(0, math.MaxInt64)
	c.MovementToken = token

	if c.IsinWar && !database.WarStarted && c.Map == 230 {
		if coordinate.X >= 155 && c.Faction == 1 && target.X > 155 || target.Y > 65 && c.Faction == 1 {
			target.X = 155
			target.Y = coordinate.Y
			if target.Y > 65 {
				target.Y = 65
			}
			c.SetCoordinate(target)
			mapID, _ := s.Character.ChangeMap(c.Map, target)
			s.Write(mapID)
			return nil, nil
		} else if coordinate.X >= 147 && c.Faction == 2 && target.X > 147 || target.Y < 457 && c.Faction == 2 {
			target.X = 147
			target.Y = coordinate.Y
			if target.Y < 457 {
				target.Y = 457
			}
			c.SetCoordinate(target)
			mapID, _ := s.Character.ChangeMap(c.Map, target)
			s.Write(mapID)
			return nil, nil
		}
	}

	delay := distance * 1000 / speed // delay (ms)
	time.AfterFunc(time.Duration(delay)*time.Millisecond, func() {
		if c.MovementToken == token {
			c.SetCoordinate(target)
		}
	})

	if speed > 5.6 && s.User.UserType != 5 {
		s.Stats.CHI -= int(speed)
		if s.Stats.CHI < 0 {
			s.Stats.CHI = 0
		}
		resp.Concat(c.GetHPandChi())
	}

	return resp, nil
}
