package database

import (
	"fmt"
	"sync"
	"time"

	"github.com/twodragon/kore-server/utils"

	"github.com/thoas/go-funk"
	gorp "gopkg.in/gorp.v1"
	null "gopkg.in/guregu/null.v3"
)

var (
	Users     = make(map[string]*User)
	userMutex sync.RWMutex

	CLOCK = utils.Packet{0xAA, 0x55, 0x1E, 0x00, 0x72, 0x01, 0x00, 0x00, 0x03, 0x08, 0x00, 0x16, 0x00, 0x24, 0x00, 0x00, 0x00, 0x55, 0xAA}
)

type User struct {
	ID              string `db:"id" json:"ID"`
	Username        string `db:"user_name" json:"Username"`
	Password        string `db:"password" json:"Password"`
	UserType        int8   `db:"user_type" json:"UserType"`
	ConnectedIP     string `db:"ip" json:"ConnectedIP"`
	ConnectedServer int    `db:"server" json:"ConnectedServer"`
	NCash           uint64 `db:"ncash" json:"NCash"`
	BankGold        uint64 `db:"bank_gold" json:"BankGold"`
	Mail            string `db:"mail" json:"Mail"`
	CreatedAt       string `db:"created_at" json:"createdAt"`
	DisabledUntil   string `db:"disabled_until" json:"disabledUntil"`
	LastLogin       string `db:"last_login" json:"last_login"`
	CheckinCounter  int    `db:"checkin_counter" json:"checkin_counter"`

	ConnectingIP string `db:"-"`
	ConnectingTo int    `db:"-"`

	SelectedServerID int `db:"-"`

	RelicCooldown int `db:"-"`

	MapBookCooldown uint16 `db:"-"`
}

func (u *User) Create() error {
	return pgsql_DbMap.Insert(u)
}

func (u *User) CreateWithTransaction(tr *gorp.Transaction) error {
	return tr.Insert(u)
}

func (u *User) Update() error {
	_, err := pgsql_DbMap.Update(u)
	return err
}

func (u *User) PreInsert(s gorp.SqlExecutor) error {
	now := time.Now().UTC()
	u.CreatedAt = null.TimeFrom(now).Time.String()
	return nil
}
func (u *User) Delete() error {
	_, err := pgsql_DbMap.Delete(u)
	return err
}

func FindUserByName(name string) (*User, error) {

	usersCache := AllUsers()

	for _, u := range usersCache {
		if u.Username == name {
			return u, nil
		}
	}

	var user User
	err := pgsql_DbMap.SelectOne(&user, "select * from hops.users where user_name = $1", name)
	if err != nil {
		return nil, err
	}

	userMutex.Lock()
	defer userMutex.Unlock()
	Users[user.ID] = &user

	return &user, nil
}

func FindUserByID(id string) (*User, error) {
	if id == "" {
		return nil, fmt.Errorf("id is empty")
	}

	usersCache := AllUsers()

	for _, u := range usersCache {
		if u.ID == id {
			return u, nil
		}
	}

	var user User
	err := pgsql_DbMap.SelectOne(&user, "select * from hops.users where id = $1", id)
	if err != nil {
		return nil, err
	}

	userMutex.Lock()
	defer userMutex.Unlock()
	Users[user.ID] = &user

	return &user, nil
}

func FindUserByIP(ip string) (*User, error) {

	usersCache := AllUsers()

	for _, u := range usersCache {
		if u.ConnectedIP == ip {
			return u, nil
		}
	}

	var user User
	err := pgsql_DbMap.SelectOne(&user, "select * from hops.users where ip = $1", ip)
	if err != nil {
		return nil, err
	}

	userMutex.Lock()
	defer userMutex.Unlock()
	Users[user.ID] = &user

	return &user, nil
}

func FindUserByMail(mail string) (*User, error) {

	usersCache := AllUsers()

	for _, u := range usersCache {
		if u.Mail == mail {
			return u, nil
		}
	}

	var user User
	err := pgsql_DbMap.SelectOne(&user, "select * from hops.users where email = $1", mail)
	if err != nil {
		return nil, err
	}

	userMutex.Lock()
	defer userMutex.Unlock()
	Users[user.ID] = &user

	return &user, nil
}

func AllUsers() []*User {
	userMutex.RLock()
	defer userMutex.RUnlock()
	return funk.Values(Users).([]*User)
}

func FindUsersInServer(server int) ([]*User, error) {

	userMutex.RLock()
	defer userMutex.RUnlock()

	arr := []*User{}
	for _, u := range Users {
		if u.ConnectedServer == server && u.ConnectedIP != "" {
			arr = append(arr, u)
		}
	}

	return arr, nil
}

func (u *User) Logout() {
	u.ConnectedIP = ""
	u.ConnectedServer = 0
	go u.Update()
}

func DeleteUserFromCache(id string) {
	userMutex.Lock()
	defer userMutex.Unlock()
	delete(Users, id)
}
func UnbanUsers() {

	userMutex.RLock()
	all := funk.Values(Users).([]*User)
	userMutex.RUnlock()

	all = funk.Filter(all, func(u *User) bool {
		date, _ := time.Parse("2006-01-02 15:04:05", u.DisabledUntil)

		return u.UserType == 0 && time.Since(date) >= 0
	}).([]*User)

	for _, u := range all {
		u.UserType = 1
		u.Update()
	}

	time.AfterFunc(time.Minute, func() {
		UnbanUsers()
	})
}

func (u *User) GetTime() []byte {

	resp := CLOCK

	serverName := fmt.Sprintf("Dragon %d", u.ConnectedServer)

	resp[7] = byte(len(serverName))
	resp.Insert([]byte(serverName), 8)

	length := int16(25 + len(serverName))
	resp.SetLength(length)

	now := time.Now()
	year := uint64(now.Year())
	month := uint64(now.Month())
	day := uint64(now.Day())
	h := uint64(now.Hour())
	m := uint64(now.Minute())
	s := uint64(now.Second())

	index := 9 + len(serverName)
	resp.Insert(utils.IntToBytes(year-2003, 2, true), index) // year
	index += 2

	resp.Insert(utils.IntToBytes(month-1, 2, true), index) // month
	index += 2

	resp.Insert(utils.IntToBytes(day, 2, true), index) // day
	index += 2

	index += 8

	resp.Insert(utils.IntToBytes(h, 2, true), index) // hour
	index += 2

	resp.Insert(utils.IntToBytes(m, 2, true), index) // minute
	index += 2

	resp.Insert(utils.IntToBytes(s, 2, true), index) // second
	index += 2

	return resp
}
