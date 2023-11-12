package server

import (
	cmap "github.com/orcaman/concurrent-map"
	"github.com/thoas/go-funk"
	"github.com/twodragon/kore-server/database"
)

const (
	BANNED_USER = iota
	COMMON_USER
	GA_USER
	GAL_USER
	GM_USER
	HGM_USER
)

var (
	MutedPlayers = cmap.New()
)

func init() {
	accUpgrades := []byte{}
	armorUpgrades := []byte{}
	weaponUpgrades := []byte{}
	htSockets := []byte{}

	for i := 1; i <= 40; i++ {
		for j := 0; j <= 5-(i%5); j++ {
			accUpgrades = append(accUpgrades, byte(i))
		}
	}

	for i := 26; i <= 65; i++ {
		for j := 0; j <= 5-(i%5); j++ {
			armorUpgrades = append(armorUpgrades, byte(i))
		}
	}
	for i := 26; i <= 65; i++ {
		for j := 0; j <= 5-(i%5); j++ {
			htSockets = append(htSockets, byte(i))
		}
	}
	htSockets = append(htSockets, byte(166)) //PVP1 Socket
	htSockets = append(htSockets, byte(167)) //PVP2 Socket
	htSockets = append(htSockets, byte(168)) //PVP3 Socket
	htSockets = append(htSockets, byte(169)) //PVP4 Socket
	htSockets = append(htSockets, byte(170)) //PVP5 Socket

	for i := 66; i <= 105; i++ {
		for j := 0; j <= 5-(i%5); j++ {
			weaponUpgrades = append(weaponUpgrades, byte(i))
		}
	}

	database.AccUpgrades = funk.Shuffle(accUpgrades).([]byte)
	database.ArmorUpgrades = funk.Shuffle(armorUpgrades).([]byte)
	database.WeaponUpgrades = funk.Shuffle(weaponUpgrades).([]byte)
	database.HTarmorSockets = funk.Shuffle(htSockets).([]byte)
}
