package database

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB = nil
var models = []any{
	&Config{},
	&User{},
}

func Connect(dsn string) (err error) {
	p, c := postgres.Open(dsn), &gorm.Config{}
	if db, err = gorm.Open(p, c); err != nil {
		return
	}
	if err = db.AutoMigrate(models...); err != nil {
		return
	}
	updateConfigs()
	createRootUser()
	return
}
