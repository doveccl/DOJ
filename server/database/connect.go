package database

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB
var models = []any{
	&Config{},
	&User{},
}

func Connect(dsn string) (err error) {
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err == nil && db.AutoMigrate(models...) == nil {
		initRootUser()
		syncConfigs()
	}
	return
}
