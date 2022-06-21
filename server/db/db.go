package db

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB = nil

func Connect(dsn string) (err error) {
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err == nil {
		initConfig()
		initUser()
	}
	return
}
