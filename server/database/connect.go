package database

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB
var initializers = map[any]func(){}

func Connect(dsn string) (e error) {
	if db, e = gorm.Open(postgres.Open(dsn), &gorm.Config{}); e == nil {
		for model, initialize := range initializers {
			db.AutoMigrate(model)
			initialize()
		}
	}
	return
}
