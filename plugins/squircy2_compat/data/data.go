package data // import "code.dopame.me/veonik/squircy3/plugins/squircy2_compat/data"

import (
	log "github.com/sirupsen/logrus"
)

func NewDatabaseConnection(rootPath string, l *log.Logger) (database *DB) {
	dir := rootPath
	database, err := OpenDB(dir, l)
	if err != nil {
		panic(err)
	}

	initDatabase(database)

	return
}

func initDatabase(database *DB) {
	col := database.Use("Settings")
	if col == nil {
		err := database.Create("Settings")
		if err != nil {
			panic(err)
		}
	}

	col = database.Use("Scripts")
	if col == nil {
		err := database.Create("Scripts")
		if err != nil {
			panic(err)
		}
	}

	col = database.Use("Webhooks")
	if col == nil {
		err := database.Create("Webhooks")
		if err != nil {
			panic(err)
		}
	}
}
