package config

type config struct {
	Database Database
	Server   Server
}

type Database struct {
	Driver          string
	IP              string
	Port            int
	User            string
	Password        string `json:"-"`
	Name            string
	ConnMaxIdle     int
	ConnMaxOpen     int
	ConnMaxLifetime int
	Debug           bool
	SSLMode         string
}

type Server struct {
	IP   string
	Port int
}

var Default = &config{
	Database: Database{
		Driver:          "postgres",
		IP:              "localhost",
		Port:            5432,
		User:            "postgres",
		Password:        "123456",
		Name:            "kore",
		ConnMaxIdle:     996,
		ConnMaxOpen:     944,
		ConnMaxLifetime: 10,
		Debug:           false,
		SSLMode:         "disable",
	},
	Server: Server{

		IP:   "127.0.0.1",
		Port: 4515,
	},
}
