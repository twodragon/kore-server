package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/thoas/go-funk"
	gorp "gopkg.in/gorp.v1"
)

const (
	SERVER_COUNT = 100
)

var (
	servers []*Server
)

type Server struct {
	ID       int    `db:"id" json:"id"`
	Name     string `db:"name" json:"name"`
	MaxUsers int    `db:"max_users" json:"max_users"`
	Epoch    int64  `db:"epoch" json:"epoch"`
}

type ServerItem struct {
	Server
	ConnectedUsers int `json:"conn_users"`
}

func (t *Server) Create() error {
	return pgsql_DbMap.Insert(t)
}

func (t *Server) CreateWithTransaction(tr *gorp.Transaction) error {
	return tr.Insert(t)
}

func (t *Server) Update() error {
	_, err := pgsql_DbMap.Update(t)
	return err
}

func (t *Server) Delete() error {
	_, err := pgsql_DbMap.Delete(t)
	return err
}

func GetServers() ([]*ServerItem, error) {

	var (
		items []*ServerItem
	)

	if len(servers) == 0 {
		query := `select * from hops.servers`

		if _, err := pgsql_DbMap.Select(&servers, query); err != nil {
			if err == sql.ErrNoRows {
				return nil, nil
			}
			return nil, fmt.Errorf("GetServers: %s", err.Error())
		}
	}

	socketMutex.RLock()
	sArr := funk.Values(Sockets)
	socketMutex.RUnlock()

	for _, s := range servers {

		i := &ServerItem{*s, 0}
		count := len(funk.Filter(sArr, func(socket *Socket) bool {
			if socket.User == nil {
				return false
			}

			return socket.User.ConnectedServer == s.ID
		}).([]*Socket))

		i.ConnectedUsers = int(count)
		items = append(items, i)
	}

	return items, nil
}

func GetServerByID(id string) (*ServerItem, error) {
	var (
		server = &Server{}
	)

	query := `select * from hops.servers where id = $1`

	if err := pgsql_DbMap.SelectOne(&server, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("GetServerByID: %s", err.Error())
	}

	i := &ServerItem{*server, 0}
	query = `select count(*) from hops.users where server = $1`
	count, err := pgsql_DbMap.SelectInt(query, server.ID)
	if err != nil {
		return nil, fmt.Errorf("GetConnectedUserCount: %s", err.Error())
	}

	i.ConnectedUsers = int(count)
	return i, nil
}
func EpochHandler() {
	server, err := GetServerByID("1")
	if err != nil {
		log.Print(err)
	}
	if server != nil {
		server.Epoch++
		server.Update()
	}
	time.AfterFunc(time.Second, EpochHandler)
}
func GetServerEpoch() int64 {
	server, err := GetServerByID("1")
	if err != nil {
		log.Print(err)
	}
	return server.Epoch
}
