package player

import (
	"fmt"

	"github.com/twodragon/kore-server/database"
	"github.com/twodragon/kore-server/nats"
	"github.com/twodragon/kore-server/utils"
)

type RespawnHandler struct {
}

var ()

func (h *RespawnHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	resp := utils.Packet{}
	respawnType := data[5]

	stat := s.Stats
	if stat == nil {
		return nil, nil
	}
	c := s.Character

	switch respawnType {
	case 1: // Respawn at Safe Zone
		c.Respawning = false
		save := database.SavePoints[int(c.Map)]
		point := &utils.Location{X: 100, Y: 100}
		if c.Map == 255 { //faction war map
			if c.Faction == 1 {
				point = &utils.Location{X: 325, Y: 465}
			}
			if c.Faction == 2 {
				point = &utils.Location{X: 179, Y: 45}
			}
		} else if c.Map == 249 {
			if c.Faction == 1 {
				point = &utils.Location{X: 439, Y: 51}
			}
			if c.Faction == 2 {
				point = &utils.Location{X: 67, Y: 481}
			}

		} else if c.Map == 230 {
			if c.Faction == 1 {
				x := 75.0
				y := 45.0
				point = database.ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", x, y))
			} else {
				x := 81.0
				y := 475.0
				point = database.ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", x, y))
			}
		} else {
			point = &utils.Location{X: save.X, Y: save.Y}
		}

		teleportData := c.Teleport(point)
		resp.Concat(teleportData)

		c.IsActive = false
		stat.HP = int(float64(stat.MaxHP) * 0.1)
		stat.CHI = int(float64(stat.MaxCHI) * 0.1)
		hpData := c.GetHPandChi()
		resp.Concat(hpData)

		c.Update()
		statData, err := c.GetStats()
		if err == nil {
			c.Socket.Write(statData)
		}

		p := nats.CastPacket{CastNear: true, CharacterID: c.ID, Type: nats.PLAYER_RESPAWN}
		p.Cast()
	case 2: // Resurrection scroll
		if c.Map == 255 || c.Map == 230 || c.Map == 243 {
			return nil, nil
		}
		slot, item, err := c.FindItemInInventory(nil, 203001186, 15600001, 15710601, 17200155, 17200205, 17200431, 17402882, 17500180, 17502671, 17502683)
		if err != nil {
			return nil, err
		}
		if item == nil || slot == -1 {
			return nil, nil
		}
		data := c.DecrementItem(slot, 1)
		c.Socket.Write(*data)

		c.Respawning = false
		c.IsActive = false
		stat.HP = stat.MaxHP
		stat.CHI = stat.MaxCHI

		hpData := c.GetHPandChi()
		resp.Concat(hpData)

		coordinate := database.ConvertPointToLocation(c.Coordinate)
		teleportData := c.Teleport(coordinate)
		resp.Concat(teleportData)

		p := nats.CastPacket{CastNear: true, CharacterID: c.ID, Type: nats.PLAYER_RESPAWN}
		p.Cast()
		statData, err := c.GetStats()
		if err == nil {
			c.Socket.Write(statData)
		}

		c.Update()

	case 4: // Respawn at Location
		c.Respawning = false
		save := database.SavePoints[int(c.Map)]
		point := &utils.Location{X: 100, Y: 100}
		if c.Map == 255 { //faction war map
			if c.Faction == 1 {
				point = &utils.Location{X: 325, Y: 465}
			}
			if c.Faction == 2 {
				point = &utils.Location{X: 179, Y: 45}
			}
		} else if c.Map == 249 {
			if c.Faction == 1 {
				point = &utils.Location{X: 439, Y: 51}
			}
			if c.Faction == 2 {
				point = &utils.Location{X: 67, Y: 481}
			}
		} else if c.Map == 230 {
			if c.Faction == 1 {
				x := 75.0
				y := 45.0
				point = database.ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", x, y))
			} else {
				x := 81.0
				y := 475.0
				point = database.ConvertPointToLocation(fmt.Sprintf("%.1f,%.1f", x, y))
			}
		} else {
			point = &utils.Location{X: save.X, Y: save.Y}
		}

		teleportData := c.Teleport(point)
		resp.Concat(teleportData)
		c.Injury = 0

		c.IsActive = false
		stat.HP = stat.MaxHP
		stat.CHI = stat.MaxCHI
		hpData := c.GetHPandChi()
		resp.Concat(hpData)

		p := nats.CastPacket{CastNear: true, CharacterID: c.ID, Type: nats.PLAYER_RESPAWN}
		p.Cast()
		price := 0
		if c.Level < 100 {
			price = c.Level * 10000
		} else if c.Level > 100 && c.Level < 201 {
			price = c.Level * 100000
		} else if c.Level > 200 {
			price = c.Level * 200000
		}
		if !c.SubtractGold(uint64(price)) {
			return nil, nil
		}
		c.Update()
		statData, err := c.GetStats()
		if err == nil {
			c.Socket.Write(statData)
		}

	}

	go c.ActivityStatus(30)
	return resp, nil
}
