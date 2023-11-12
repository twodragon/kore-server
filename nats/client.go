package nats

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

const (
	HOUSTON_CH = "Houston11"
)

var (
	conn *nats.Conn
)

type CastPacket struct {
	CastNear    bool `json:"cast_near"`
	CharacterID int  `json:"character_id"`
	MobID       int  `json:"mob_id"`
	PetID       int  `json:"pet_id"`
	BabyPetID   int  `json:"baby_pet_id"`
	DropID      int  `json:"loot_id"`
	Location    *struct {
		X float64
		Y float64
	} `json:"location"`
	MaxDistance float64 `json:"max_distance"`
	Data        []byte  `json:"data"`
	Type        int8    `json:"type"`
}

func ConnectSelf(opts *server.Options) (*nats.Conn, error) {
	var err error
	if opts == nil {
		opts = &DefaultOptions
	}

	url := fmt.Sprintf("nats://%s:%d", opts.Host, opts.Port)
	//use best options for nats connectivity
	conn, err = nats.Connect(url, nats.MaxReconnects(-1), nats.ReconnectWait(time.Second*5), nats.Timeout(time.Second*5), nats.DisconnectHandler(func(nc *nats.Conn) {
		fmt.Println("Disconnected from nats server")
		nc.Close()
	}))

	if err != nil {
		return nil, err
	}
	return conn, nil
}

func Connection() *nats.Conn {
	return conn
}

func (p *CastPacket) Cast() error {

	data, err := json.Marshal(p)
	if err != nil {
		return err
	}

	return Connection().Publish(HOUSTON_CH, data)
}
