package database

import (
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

const schema = `
CREATE TABLE users (
	id INT NOT NULL PRIMARY KEY, -- use a provided id
	name VARCHAR(26),
	display_name VARCHAR(26),

	first_seen TIMESTAMP,
	last_seen TIMESTAMP,

	timeout TIMESTAMP NULL,
	banned BOOLEAN DEFAULT false,

	last_patsched TIMESTAMP NULL,
	patsch_streak INT DEFAULT 0,
	patsch_count INT DEFAULT 0,

	birthday DATE NULL
);
CREATE TABLE channels (
	id INT NOT NULL PRIMARY KEY, -- use a provided id
	name VARCHAR(26),
	enabled BOOLEAN NOT NULL,
	paused BOOLEAN DEFAULT false
	readonly BOOLEAN DEFAULT false
);
CREATE TABLE voicemails (
	id SERIAL PRIMARY KEY,
	creator INT REFERENCES users(id) ON DELETE CASCADE,
	created TIMESTAMP NOT NULL,
	recipent VARCHAR(26) NOT NULL,
	message VARCHAR(500) NOT NULL,
	replayed BOOLEAN DEFAULT false
);
CREATE TABLE copypastas (
	id SERIAL PRIMARY KEY,
	message VARCHAR(500)
)
`

type Client struct {
	DB    *sqlx.DB
	viper *viper.Viper
}

func NewClient(viper *viper.Viper) *Client {
	return &Client{
		viper: viper,
	}
}

func (c *Client) Connect() {
	user := c.viper.GetString("user")
	password := c.viper.GetString("password")
	dbname := c.viper.GetString("dbname")

	password = strings.ReplaceAll(password, "'", "\\'")

	db, err := sqlx.Connect("postgres",
		fmt.Sprintf("user=%s password='%s' dbname=%s sslmode=require", user, dbname, password),
	)
	if err != nil {
		log.Fatal().
			Err(err).
			Str("dbname", dbname).
			Str("user", user).
			Msg("Could not connect to database")
	}

	_, err = db.Exec(schema)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Could not verify schema")
	}

	c.DB = db
}
