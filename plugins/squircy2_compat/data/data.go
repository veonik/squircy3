package data // import "code.dopame.me/veonik/squircy3/plugins/squircy2_compat/data"

import (
	log "github.com/sirupsen/logrus"
)

func NewDatabaseConnection(rootPath string, l *log.Logger) *DB {
	dir := rootPath
	database, err := OpenDB(dir, l)
	if err != nil {
		panic(err)
	}

	return database
}
