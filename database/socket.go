package database

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/twodragon/kore-server/utils"

	"github.com/nats-io/nats.go"
	"github.com/paulbellamy/ratecounter"
)

var (
	Handler        func(*Socket, []byte, uint16) ([]byte, error)
	Sockets        = make(map[string]*Socket)
	socketMutex    sync.RWMutex
	HeartBeats     = make(map[string]*HeartBeat)
	HeartBeatMutex sync.RWMutex
)

type Socket struct {
	Conn       net.Conn
	ClientAddr string
	User       *User
	Character  *Character
	Stats      *Stat
	Skills     *Skills
	Teleports  *Teleports
	HoustonSub *nats.Subscription
}

func init() {
}

func (s *Socket) Add(id string) {
	socketMutex.Lock()
	defer socketMutex.Unlock()
	Sockets[id] = s
}

func (s *Socket) Remove(id string) {
	socketMutex.Lock()
	defer socketMutex.Unlock()
	delete(Sockets, id)
}

func GetSocket(id string) *Socket {
	socketMutex.RLock()
	defer socketMutex.RUnlock()
	return Sockets[id]
}

func (s *Socket) Read() {

	counter := ratecounter.NewRateCounter(1 * time.Second)

	for {
		buf := make([]byte, 4096)
		n, err := s.Conn.Read(buf)
		if err != nil { // do not remove connecting ip here
			s.OnClose()
			break
		}

		go func() {
			counter.Incr(1)
			resp, err := s.recognizePacket(buf[:n])
			if err != nil {
				log.Println("recognize packet error:", err)
			}

			if len(resp) > 0 {
				packets := bytes.SplitAfter(resp, []byte{0x55, 0xAA})
				for _, packet := range packets {
					if len(packet) == 0 {
						continue
					}
					err := s.Write(packet)
					if err != nil {
						s.OnClose()
						break
					}
					time.Sleep(time.Duration(len(packet)/25) * time.Millisecond)
				}
			}
		}()

		if counter.Rate() > 35 {
			if s.User != nil {
				text := "Account " + s.User.Username + " disconnected from server for ratecounter."
				log.Print(text)
				utils.NewLog("logs/rate_kicks.txt", text)
			} else {
				text := "IP " + s.ClientAddr + " disconnected from server for ratecounter."
				log.Print(text)
				utils.NewLog("logs/rate_kicks.txt", text)
			}

			s.OnClose()
			break
		}

	}
}

func (s *Socket) OnClose() {
	if s == nil {
		return
	}
	s.Conn.Close()
	if u := s.User; u != nil {
		s.Remove(u.ID)
		if s.User.ConnectingIP == "" {
			u.Logout()
		}
	}
	if c := s.Character; c != nil {
		c.Logout()
	}
	if s.HoustonSub != nil {
		s.HoustonSub.Unsubscribe()
	}
	s = nil
}

func (s *Socket) recognizePacket(data []byte) ([]byte, error) {
	packets := bytes.SplitAfter(data, []byte{0x55, 0xAA})

	resp := utils.Packet{}
	for _, packet := range packets {

		if len(packet) < 6 {
			continue
		}

		if os.Getenv("PROXY_ENABLED") == "5" {
			header, body := []byte{}, []byte{}
			if bytes.Contains(packet, []byte{0xAA, 0x55}) {
				pParts := bytes.Split(packet, []byte{0xAA, 0x55})
				if len(pParts) == 1 {
					body = append([]byte{0xAA, 0x55}, pParts[0]...)

				} else {
					header = pParts[0]
					body = append([]byte{0xAA, 0x55}, pParts[1]...)
				}
			} else {
				header = packet
			}

			s.ParseHeader(header)

			if len(body) > 0 {
				sign := uint16(utils.BytesToInt(body[4:6], false))
				d, err := Handler(s, body, sign)
				if err != nil {
					return nil, err
				}

				resp.Concat(d)
			}

		} else {
			s.ClientAddr = s.Conn.RemoteAddr().String()

			sign := uint16(utils.BytesToInt(packet[4:6], false))
			d, err := Handler(s, packet, sign)
			if err != nil {
				return nil, err
			}

			resp.Concat(d)
		}
	}

	return resp, nil
}

func (s *Socket) Write(data []byte) error {

	if s != nil && s.Conn != nil {
		_, err := s.Conn.Write(data)
		if err != nil {
			s.OnClose()
			return err
		}
	}

	return nil
}

func (s *Socket) ParseHeader(header []byte) {

	if len(header) == 0 {
		return
	}

	sHeader := string(header)
	if !strings.HasPrefix(sHeader, "PROXY TCP4") {
		return
	}

	parts := strings.Split(sHeader, " ")
	clientIP := parts[2]
	clientPort := parts[4]

	s.ClientAddr = fmt.Sprintf("%s:%s", clientIP, clientPort)
}

type HeartBeat struct {
	Ip    string
	Count int64
	Last  time.Time
}

func GetHeartBeats() []*HeartBeat {
	HeartBeatMutex.RLock()
	defer HeartBeatMutex.RUnlock()
	var heartbeats []*HeartBeat
	for _, v := range HeartBeats {
		heartbeats = append(heartbeats, v)
	}
	return heartbeats
}
func GetHeartBeatsByIp(ip string) *HeartBeat {
	HeartBeatMutex.RLock()
	defer HeartBeatMutex.RUnlock()
	ratecounter, ok := HeartBeats[ip]
	if !ok {
		return nil
	}
	return ratecounter
}
func SetHeartBeats(heartbeat *HeartBeat) {
	HeartBeatMutex.Lock()
	defer HeartBeatMutex.Unlock()
	HeartBeats[heartbeat.Ip] = heartbeat
}
