package database

import (
	"database/sql"

	"fmt"
	"log"
	"os"
	"time"

	"github.com/twodragon/kore-server/config"
	"github.com/twodragon/kore-server/logging"
	"github.com/twodragon/kore-server/utils"

	_ "github.com/lib/pq"
	gorp "gopkg.in/gorp.v1"
)

var (
	DROP_LIFETIME                 = time.Duration(60) * time.Second
	FREEDROP_LIFETIME             = time.Duration(15) * time.Second
	DROP_RATE                     = utils.ParseFloat("1.0")
	DEFAULT_DROP_RATE             = utils.ParseFloat("1.0")
	DROP_RATE_TIME                = int64(0)
	EXP_RATE                      = utils.ParseFloat("1.0")
	DEFAULT_EXP_RATE              = utils.ParseFloat("1.0")
	EXP_RATE_TIME                 = int64(0)
	GOLD_RATE                     = utils.ParseFloat("1.0")
	DEFAULT_GOLD_RATE             = utils.ParseFloat("1.0")
	GOLD_RATE_TIME                = int64(0)
	MAX_INJURY                    = utils.ParseFloat("100.0")
	RELIC_DROP_ENABLED            = true
	DEFAULT_RUNNING_SPEED float64 = 0.9
	DEBUG_FACTORY                 = 0
	MAX_GUILD_MEMBERS             = int8(22)
	DEVLOG                        = 0
)

var (
	cfg         = config.Default
	pgsql_DbMap *gorp.DbMap

	Init                    = make(chan bool, 1)
	GetFromRegister         func(int, int16, uint16) interface{}
	RemoveFromRegister      func(*Character)
	RemovePetFromRegister   func(c *Character)
	FindCharacterByPseudoID func(server int, ID uint16) *Character

	AccUpgrades    []byte
	ArmorUpgrades  []byte
	WeaponUpgrades []byte
	HTarmorSockets []byte
	plusRates      = []int{800, 900, 950, 980, 990, 996, 999}
	logger         = logging.Logger
)

func InitPostgreSQL() error {

	var (
		//drv         = cfg.Database.Driver
		ip          = cfg.Database.IP
		port        = cfg.Database.Port
		user        = cfg.Database.User
		pass        = cfg.Database.Password
		name        = cfg.Database.Name
		maxIdle     = cfg.Database.ConnMaxIdle
		maxOpen     = cfg.Database.ConnMaxOpen
		maxLifetime = cfg.Database.ConnMaxLifetime
		debug       = cfg.Database.Debug
		sslMode     = cfg.Database.SSLMode
		err         error
		conn        *sql.DB
	)

	conn, err = sql.Open("postgres", fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s", ip, port, user, pass, name, sslMode))
	if err != nil {
		return fmt.Errorf("database connection error: %s", err.Error())
	}

	conn.SetMaxIdleConns(maxIdle)
	conn.SetMaxOpenConns(maxOpen)
	conn.SetConnMaxLifetime(time.Duration(maxLifetime) * time.Second)

	if err = conn.Ping(); err != nil {
		return fmt.Errorf("database connection error: %s", err.Error())
	}

	pgsql_DbMap = &gorp.DbMap{Db: conn, Dialect: gorp.PostgresDialect{}}

	pgsql_DbMap.AddTableWithNameAndSchema(Adventurer{}, "hops", "adventurers").SetKeys(true, "id")
	pgsql_DbMap.AddTableWithNameAndSchema(HousingItem{}, "hops", "houses").SetKeys(true, "id")
	pgsql_DbMap.AddTableWithNameAndSchema(Character{}, "hops", "characters").SetKeys(true, "id")
	pgsql_DbMap.AddTableWithNameAndSchema(Buff{}, "hops", "characters_buffs").SetKeys(false, "id", "character_id")
	pgsql_DbMap.AddTableWithNameAndSchema(Friend{}, "hops", "characters_friends").SetKeys(true, "id")
	pgsql_DbMap.AddTableWithNameAndSchema(MailMessage{}, "hops", "characters_mails").SetKeys(true, "id")

	pgsql_DbMap.AddTableWithNameAndSchema(Guild{}, "hops", "guilds").SetKeys(true, "id")
	pgsql_DbMap.AddTableWithNameAndSchema(InventorySlot{}, "hops", "items_characters").SetKeys(true, "id")
	pgsql_DbMap.AddTableWithNameAndSchema(DisabledInventorySlot{}, "hops", "items_disabled_characters").SetKeys(true, "id")
	pgsql_DbMap.AddTableWithNameAndSchema(Relic{}, "hops", "relics")
	pgsql_DbMap.AddTableWithNameAndSchema(Server{}, "hops", "servers").SetKeys(true, "id")
	pgsql_DbMap.AddTableWithNameAndSchema(FiveClan{}, "hops", "fiveclan_war").SetKeys(true, "id")
	pgsql_DbMap.AddTableWithNameAndSchema(BannedIp{}, "hops", "banned_ips").SetKeys(true, "id")

	pgsql_DbMap.AddTableWithNameAndSchema(Teleports{}, "hops", "characters_teleports").SetKeys(false, "id")
	pgsql_DbMap.AddTableWithNameAndSchema(ConsignmentItem{}, "hops", "consign").SetKeys(false, "id")
	pgsql_DbMap.AddTableWithNameAndSchema(CheckIn{}, "hops", "checkin").SetKeys(false, "charid")
	pgsql_DbMap.AddTableWithNameAndSchema(BabyPet{}, "hops", "baby_pets").SetKeys(true, "id")

	pgsql_DbMap.AddTableWithNameAndSchema(ExpInfo{}, "data", "exp_table").SetKeys(false, "level")
	pgsql_DbMap.AddTableWithNameAndSchema(Production{}, "data", "productions").SetKeys(false, "id")
	pgsql_DbMap.AddTableWithNameAndSchema(CookingItem{}, "data", "cooking").SetKeys(false, "id")
	pgsql_DbMap.AddTableWithNameAndSchema(Gambling{}, "data", "gambling").SetKeys(false, "id")
	pgsql_DbMap.AddTableWithNameAndSchema(HaxCode{}, "data", "hax_codes").SetKeys(false, "id")
	pgsql_DbMap.AddTableWithNameAndSchema(NPCScript{}, "data", "npc_scripts").SetKeys(false, "id")
	pgsql_DbMap.AddTableWithNameAndSchema(QuestList{}, "data", "quests").SetKeys(false, "id")
	pgsql_DbMap.AddTableWithNameAndSchema(Enhancement{}, "data", "enchant").SetKeys(false, "bookid")
	pgsql_DbMap.AddTableWithNameAndSchema(TempleData{}, "data", "fiveclan_war").SetKeys(false, "id")
	pgsql_DbMap.AddTableWithNameAndSchema(Relic{}, "data", "relics").SetKeys(false, "id")
	pgsql_DbMap.AddTableWithNameAndSchema(CheckinReward{}, "data", "checkin_rewards").SetKeys(false, "day")
	pgsql_DbMap.AddTableWithNameAndSchema(AI{}, "data", "ai").SetKeys(false, "id")

	pgsql_DbMap.AddTableWithNameAndSchema(User{}, "hops", "users").SetKeys(true, "id")
	pgsql_DbMap.AddTableWithNameAndSchema(Skills{}, "hops", "skills").SetKeys(false, "id")
	pgsql_DbMap.AddTableWithNameAndSchema(Stat{}, "hops", "stats").SetKeys(false, "id")
	pgsql_DbMap.AddTableWithNameAndSchema(Guild{}, "hops", "guilds").SetKeys(true, "id")

	if debug {
		pgsql_DbMap.TraceOn("[gorp]", log.New(os.Stdout, "myapp:", log.Lmicroseconds))
	}

	if err = resetDB(); err != nil {
		return err
	}

	if err = getAll(); err != nil {
		return err
	}

	Init <- err == nil
	return nil
}

func resetDB() error {

	query := `update hops.characters set is_active = false, is_online = false`
	if _, err := pgsql_DbMap.Exec(query); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("Reset DB error: %s", err.Error())
	}

	query = `update hops.users set ip = $1, server = 0`
	if _, err := pgsql_DbMap.Exec(query, ""); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("Reset DB error: %s", err.Error())
	}

	return nil
}

func getAll() error {

	fmt.Print("------------------------Reading Tables...-------------------------\n")

	callBacks := []func() error{getScripts, getHaxCodes, getEnhancements, GetBuffInfections, GetExps, GetAllCheckinRewards, GetBabyPets,
		GetRelics, getFiveAreas, getCookingItems,
		getTempleDatas, GetBannedIps, ReadConsignmentData, GetCheckIns, GetAdventurers, GetAllGuilds}

	for _, cb := range callBacks {
		if err := cb(); err != nil {
			return err
		}
	}

	log.Print("Initiating spreadsheets reading...")
	callBacks = []func() error{ReadAllDropsInfo, GetHTItems, GetAdvancedFusions, GetAllItems, GetAllSavePoints, GetCraftItem, getItemJudgements, GetProductions, GetMaps, GetEmotions,
		GetPets, GetPetsExps, GetGates, GetBossStages, ReadAllNPCsInfo, GetSkills, GetJobPassives, GetShopItems, GetShopsTable, ReadMeltings, ReadItemSets, GetHouseItems, FindBuiltHouses,
		GetHeadItem, GetFaceItem, GetGamblings, GetStarerItems}
	for _, cb := range callBacks {
		if err := cb(); err != nil {
			return err
		}
	}
	log.Print("----------------------------------------------------------------")
	return nil
}
