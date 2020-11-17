package db

import (
	"SMSRouter/internal"
	"fmt"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // postgres driver
)

const (
	SetMaxOpenConns = 30
	SetMaxIdleConns = 30
)

var schema = `
CREATE TABLE IF NOT EXISTS sms_sms (
    id               serial                   not null
        constraint sms_sms_pkey
            primary key,
    message_id       varchar(150),
    message_sequence integer,
    phone            varchar(150)             not null,
    message          text                     not null,
    is_sent          boolean                  not null,
    is_delivered     boolean                  not null,
    date_created     timestamp with time zone not null
);`

func InitDB() (*sqlx.DB, error) {
	DBDsn := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=disable",
		internal.Configuration.DB_HOST,
		internal.Configuration.DB_PORT,
		internal.Configuration.DB_USERNAME,
		internal.Configuration.DB_NAME,
		internal.Configuration.DB_PASSWORD,
	)
	db, err := sqlx.Connect("postgres", DBDsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(SetMaxOpenConns)
	db.SetMaxIdleConns(SetMaxIdleConns)

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	db.MustExec(schema)

	return db, nil
}
