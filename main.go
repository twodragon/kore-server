package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strconv"
	"time"

	"github.com/robfig/cron"
	"github.com/twodragon/kore-server/ai"
	"github.com/twodragon/kore-server/config"
	"github.com/twodragon/kore-server/database"
	_ "github.com/twodragon/kore-server/factory"
	"github.com/twodragon/kore-server/logging"
	"github.com/twodragon/kore-server/nats"
	//	"github.com/twodragon/kore-server/redis"
)

var (
	logger    = logging.Logger
	CacheFile = "cache.json"
)

func initDB_PostgreSQL() {
	for {
		err := database.InitPostgreSQL()
		if err == nil {
			log.Print("Connected to PostgreSQL database...")
			return
		}
		log.Print(fmt.Sprintf("PostgreSQL Database connection error: %+v, waiting 30 sec...", err))
		time.Sleep(time.Duration(30) * time.Second)
	}
}

/*
	func initRedis() {
		for {
			err := redis.InitRedis()
			if err != nil {
				log.Printf("Redis connection error: %+v, waiting 30 sec...", err)
				time.Sleep(time.Duration(30) * time.Second)
				continue
			}

			if redisHost := os.Getenv("REDIS_HOST"); redisHost != "" {
				log.Printf("Connected to redis...")
				go logger.StartLogging()
			}

			return
		}
	}
*/
func StartLogging() {
	fi, err := os.OpenFile("Log.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666) //log file
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	log.SetOutput(fi)
}

func startServer() {
	cfg := config.Default
	port := cfg.Server.Port
	listen, err := net.Listen("tcp4", ":"+strconv.Itoa(port))
	if err != nil {
		fmt.Printf("Socket listen port %d failed,%s", port, err)
	}
	defer listen.Close()
	log.Printf("Begin listen port: %d", port)
	//StartLogging()
	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Fatalln(err)
			continue
		}

		ws := database.Socket{Conn: conn}
		go ws.Read()

	}

}

//

func cronHandler() {
	var err error
	aidr := cron.New()
	aidr.AddFunc("0 0 0 * *", func() {

		err = database.RefreshAIDs()
		err = database.RefreshYingYangKeys()
	})
	aidr.Start()

	if err != nil {
		log.Print("cronHandler err")
		fmt.Print(err)
	}
}

func main() {

	var err error
	go func() {
		fmt.Println(http.ListenAndServe("localhost:8080", nil))
	}()
	log.Print("-----------------Initialize pgsql-------------------------------")
	initDB_PostgreSQL()
	log.Print("--------------------------------------------------------------")

	cronHandler()
	log.Print("--------------------------------------------------------------")
	err = ai.Init()
	if err != nil {
		log.Fatalln(err)
	}
	log.Print("--------------------------------------------------------------")
	//ai.InitBabyPets()
	//ai.InitHouseItems()
	//go database.HandleClanBuffs()
	//go database.StartLoto() buglu
	//go database.UnbanUsers()
	//go database.InitDiscordBot()
	//go database.DeleteInexistentItems()
	//go database.FactionWarSchedule()
	go database.DeleteUnusedStats()
	log.Print("--------------------------------------------------------------")
	s := nats.RunServer(nil)
	log.Print("-------------------------------------------------------------*")
	defer s.Shutdown()
	c, err := nats.ConnectSelf(nil)
	if err != nil {
		log.Fatalln(err)
	}
	defer c.Close()
	go database.EpochHandler()
	startServer()

}
