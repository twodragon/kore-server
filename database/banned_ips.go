package database

import (
	"database/sql"
	"fmt"
)

var (
	BannedIps = make(map[int]*BannedIp)
)

type BannedIp struct {
	Id       int    `db:"id"`
	BannedIp string `db:"ip"`
}

func (b *BannedIp) Update() error {
	_, err := pgsql_DbMap.Update(b)
	return err
}

func GetBannedIps() error {
	var ips []*BannedIp
	query := `select * from hops.banned_ips`

	if _, err := pgsql_DbMap.Select(&ips, query); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("GetBannedIps: %s", err.Error())
	}

	for _, cr := range ips {
		BannedIps[cr.Id] = cr
	}
	return nil
}
