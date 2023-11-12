package database

import "github.com/twodragon/kore-server/utils"

type Duel struct {
	EnemyID    int
	Coordinate utils.Location
	Started    bool
}
