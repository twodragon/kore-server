package database

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"log"

	"github.com/twodragon/kore-server/utils"

	"github.com/osamingo/boolconv"
	"gopkg.in/gorp.v1"
)

type Friend struct {
	ID          int  `db:"id" json:"id"`
	CharacterID int  `db:"character_id" json:"character_id"`
	FriendID    int  `db:"friend_id" json:"friend_id"`
	IsBlocked   bool `db:"is_blocked" json:"is_blocked"`
}

var (
	INIT_FRIEND   = utils.Packet{0xAA, 0x55, 0x1d, 0x00, 0xcb, 0x01, 0x55, 0xAA}
	ADD_FRIEND    = utils.Packet{0xaa, 0x55, 0x18, 0x00, 0xcb, 0x03, 0x0a, 0x00, 0x00, 0x00, 0xff, 0x00, 0x55, 0xaa}
	BLOCK_FRIEND  = utils.Packet{0xaa, 0x55, 0x18, 0x00, 0xcb, 0x03, 0x0a, 0x00, 0x01, 0x00, 0xff, 0x00, 0x55, 0xaa}
	MODIFY_FRIEND = utils.Packet{0xaa, 0x55, 0x09, 0x00, 0xcb, 0x02, 0xff, 0x00, 0x55, 0xaa}
)

func (b *Friend) Create() error {
	return pgsql_DbMap.Insert(b)
}

func (b *Friend) CreateWithTransaction(tr *gorp.Transaction) error {
	return tr.Insert(b)
}

func (b *Friend) Delete() error {
	_, err := pgsql_DbMap.Delete(b)
	return err
}
func (b *Friend) Update() error {
	_, err := pgsql_DbMap.Update(b)
	if err != nil {
		fmt.Printf("Error: %s", err.Error())
	}
	return err
}

func FindAllFriendsIDsByCharacterID(characterID int) ([]*Friend, error) {

	var friends []*Friend
	query := `select * from hops.characters_friends where friend_id = $1`

	if _, err := pgsql_DbMap.Select(&friends, query, characterID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindFriendsByCharacterID: %s", err.Error())
	}

	return friends, nil
}

func FindFriendsByCharacterID(characterID int) ([]*Friend, error) {

	var friends []*Friend
	query := `select * from hops.characters_friends where character_id = $1`

	if _, err := pgsql_DbMap.Select(&friends, query, characterID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindFriendsByCharacterID: %s", err.Error())
	}

	return friends, nil
}

func FindFriendByCharacterAndFriendID(characterID, friendID int) (*Friend, error) {

	var g Friend
	query := `select * from hops.characters_friends where character_id = $1 AND friend_id = $2`

	if err := pgsql_DbMap.SelectOne(&g, query, characterID, friendID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindGuildByName: %s", err.Error())
	}

	return &g, nil
}

func FindFriendsByID(id int) (*Friend, error) {

	friend := &Friend{}
	query := `select * from "hops".characters_friends where "id" = $1`

	if err := pgsql_DbMap.SelectOne(&friend, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindFriendByID: %s", err.Error())
	}

	return friend, nil
}
func DeleteAllFriendsByCharID(id int) bool {
	res, err := pgsql_DbMap.Exec(`DELETE FROM "hops".characters_friends WHERE character_id = $1 or friend_id = $1`, id)

	if err == nil {

		_, err := res.RowsAffected()
		if err == nil {
			return true
		}

	}

	return false

}
func InitFriends(char *Character) ([]byte, error) {
	resp := INIT_FRIEND
	friends, err := FindFriendsByCharacterID(char.ID)
	index := 6
	resp.Insert(utils.IntToBytes(uint64(len(friends)), 1, true), index) //Friends length
	index++
	if err != nil {
		return nil, err
	}
	for _, friend := range friends {
		resp.Insert(utils.IntToBytes(uint64(friend.ID), 4, true), index)
		index += 4
		friendchar, _ := FindCharacterByID(friend.FriendID)
		resp.Insert(utils.IntToBytes(uint64(len(friendchar.Name)), 1, true), index)
		index++
		resp.Insert([]byte(friendchar.Name), index)
		index += len(friendchar.Name)
		if friend.IsBlocked {
			resp.Insert([]byte{0x01}, index)
		} else {
			resp.Insert([]byte{0x00}, index)
		}
		index++
		online, err := boolconv.NewBoolByInterface(friendchar.IsOnline)
		if err != nil {
			log.Println("error should not be nil")
		}
		resp.Insert(online.Bytes(), index)
		index++
	}
	resp.Insert([]byte{0x00}, index)
	resp.SetLength(int16(binary.Size(resp) - 6))
	return resp, nil
}
