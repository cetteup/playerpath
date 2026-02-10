package sqlutil

import (
	"database/sql"
	"time"

	"github.com/go-sql-driver/mysql"
)

func Connect(host, dbname, user, passwd string) *sql.DB {
	cfg := mysql.Config{
		User:                 user,
		Passwd:               passwd,
		Net:                  "tcp",
		Addr:                 host,
		DBName:               dbname,
		Loc:                  time.UTC,
		MaxAllowedPacket:     64 << 20, // same as mysql.defaultMaxAllowedPacket
		ParseTime:            true,
		AllowNativePasswords: true,
		CheckConnLiveness:    true,
	}

	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		panic(err)
	}

	db.SetConnMaxIdleTime(15 * time.Minute)
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(2)

	return db
}
