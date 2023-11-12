package auth

import (
	"crypto/sha256"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/twodragon/kore-server/database"
	"github.com/twodragon/kore-server/logging"
	"github.com/twodragon/kore-server/utils"
)

type LoginHandler struct {
	password string
	username string
}

var (
	USER_NOT_FOUND = utils.Packet{0xAA, 0x55, 0x23, 0x00, 0x00, 0x01, 0x00, 0x1F, 0x4D, 0x69, 0x73, 0x6D, 0x61, 0x74, 0x63, 0x68, 0x20, 0x41, 0x63, 0x63, 0x6F, 0x75, 0x6E, 0x74, 0x20, 0x49, 0x44, 0x20, 0x6F, 0x72, 0x20, 0x50, 0x61, 0x73, 0x73, 0x77, 0x6F, 0x72, 0x64, 0x55, 0xAA}
	LOGGED_IN      = utils.Packet{0xaa, 0x55, 0x57, 0x00, 0x00, 0x01, 0x01, 0x40, 0x30, 0x42, 0x41, 0x45, 0x35, 0x32, 0x46, 0x45, 0x34, 0x32, 0x30, 0x44, 0x41, 0x35, 0x39, 0x32, 0x30, 0x41, 0x30, 0x33, 0x39, 0x46, 0x31, 0x41, 0x39, 0x30, 0x38, 0x34, 0x46, 0x31, 0x38, 0x38, 0x34, 0x41, 0x39, 0x36, 0x33, 0x44, 0x34, 0x30, 0x45, 0x38, 0x41, 0x39, 0x45, 0x44, 0x37, 0x35, 0x44, 0x35, 0x43, 0x41, 0x45, 0x43, 0x31, 0x46, 0x43, 0x44, 0x39, 0x45, 0x44, 0x33, 0x31, 0x38, 0x00, 0x00, 0xdb, 0x89, 0x2d, 0x06, 0x55, 0xaa}
	USER_BANNED    = utils.Packet{0xAA, 0x55, 0x36, 0x00, 0x00, 0x01, 0x00, 0x32, 0x59, 0x6F, 0x75, 0x72, 0x20, 0x61, 0x63, 0x63, 0x6F, 0x75, 0x6E, 0x74, 0x20, 0x68, 0x61, 0x73, 0x20, 0x62, 0x65, 0x65, 0x6E, 0x20, 0x64, 0x69, 0x73, 0x61, 0x62, 0x6C, 0x65, 0x64, 0x20, 0x75, 0x6E, 0x74, 0x69, 0x6C, 0x20, 0x5B, 0x5D, 0x2E, 0x55, 0xAA}

	logger = logging.Logger

	ip2locationFile = "./IP2LOCATION-LITE-DB1.BIN"
)

func (lh *LoginHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	index := 6 //9
	uNameLen := int(utils.BytesToInt(data[index:index+1], false))
	lh.username = string(data[index+1 : index+uNameLen+1])
	passworde := string(data[index+uNameLen+2 : index+uNameLen+37])
	salt := "188.132.128.35"
	test := fmt.Sprintf("%s%s", passworde, salt) //salt test --
	data2 := []byte(test)
	hashx2 := sha256.Sum256(data2)
	hasp2 := string(hashx2[:])
	lh.password = fmt.Sprintf("%X", hasp2) //--

	log.Printf("lh.username= %s lh.password=     %s", lh.username, lh.password)
	return lh.login(s)
}

func (lh *LoginHandler) login(s *database.Socket) ([]byte, error) {
	var user *database.User
	var err error

	user, err = database.FindUserByName(lh.username)
	if err != nil {
		log.Print(err)
	}

	if user == nil || err != nil {
		time.Sleep(time.Second / 2)
		return USER_NOT_FOUND, nil
	}

	parts := strings.Split(s.Conn.RemoteAddr().String(), ":")
	ip := parts[0]

	if !checkip(ip) {
		s.Conn.Close()
		user.Logout()
		if sock := database.GetSocket(user.ID); sock != nil {
			if c := sock.Character; c != nil {
				c.Logout()
			}
			sock.Conn.Close()
		}
	}

	var resp utils.Packet
	if strings.Compare(lh.password, user.Password) == 0 { // login succeeded

		if user.UserType == 0 { // Banned
			resp = USER_BANNED
			resp.Insert([]byte(user.DisabledUntil), 0x2E) // ban duration
			return resp, nil
		}

		if user.ConnectedIP != "" { // user already online
			logger.Log(logging.ACTION_LOGIN, 0, "Multiple login", user.ID)

			if sock := database.GetSocket(user.ID); sock != nil {
				if c := sock.Character; c != nil {
					c.Logout()
				}
				sock.Conn.Close()
			}
		}
		logger.Log(logging.ACTION_LOGIN, 0, "Login successful", user.ID)
		resp = LOGGED_IN
		s.User = user
		s.User.ConnectedIP = s.ClientAddr
		log.Print(s.ClientAddr)
		length := int16(len(lh.username) + 75)
		namelength := len(lh.username)

		s.User.LastLogin = time.Now().Format("2006-01-02 15:04:05")

		resp.SetLength(length)
		resp.Insert([]byte(utils.IntToBytes(uint64(namelength), 1, false)), 7)
		resp.Insert([]byte(lh.username), 8)
		err := s.User.Update()
		if err != nil {
			log.Print(err)
		}
		text := "ID: " + s.User.Username + "(" + s.User.ID + ") logged in with ip: " + s.User.ConnectedIP
		utils.NewLog("logs/ip_logs.txt", text)
	} else { // login failed
		logger.Log(logging.ACTION_LOGIN, 0, "Login failed.", user.ID)
		time.Sleep(time.Second / 2)
		resp = USER_NOT_FOUND
	}

	return resp, nil
}

func checkip(ip string) bool {
	// return true
	// database.GetBannedBannedRegions()
	// all, err := db.GetAll(ip)

	// if err != nil {
	// 	fmt.Print(err)
	// 	return false
	// }
	for _, bi := range database.BannedIps {
		if bi.BannedIp == ip {
			return false
		}
	}
	//fmt.Printf("isProxy: %s\n", all["isProxy"])
	//fmt.Printf("CountryShort: %s\n", all["CountryShort"])
	//fmt.Printf("CountryLong: %s\n", all["CountryLong"])

	return true
}
