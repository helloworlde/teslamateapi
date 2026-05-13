package server

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/tobiasehlert/teslamateapi/internal/apicommon"
	"github.com/tobiasehlert/teslamateapi/internal/config"
)

// initDB opens the Postgres connection and stores it on apicommon.DB.
func initDB() {
	dbhost := config.Env("DATABASE_HOST", "database")
	dbport := config.EnvAsInt("DATABASE_PORT", 5432)
	dbuser := config.Env("DATABASE_USER", "teslamate")
	dbpass := config.Env("DATABASE_PASS", "secret")
	dbname := config.Env("DATABASE_NAME", "teslamate")
	dbtimeout := config.EnvAsInt("DATABASE_TIMEOUT", 60000) / 1000
	dbsslmode := config.Env("DATABASE_SSL", "disable")
	dbsslrootcert := config.Env("DATABASE_SSL_CA_CERT_FILE", "")

	switch dbsslmode {
	case "true", "noverify":
		dbsslmode = "require"
	case "false":
		dbsslmode = "disable"
	}

	psqlInfo := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s connect_timeout=%d",
		dbhost, dbport, dbuser, dbpass, dbname, dbsslmode, dbtimeout,
	)
	if dbsslrootcert != "" {
		psqlInfo += " sslrootcert=" + dbsslrootcert
	}

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatalf("[error] initDB - database connection error: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("[error] initDB - database ping error: %v", err)
	}
	if gin.IsDebugging() {
		log.Println("[debug] initDB - database connection established successfully.")
	}
	apicommon.DB = db
}
