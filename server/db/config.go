package db

import (
	"log"

	"github.com/google/uuid"
)

type Config struct {
	Key    string `gorm:"primaryKey"`
	Value  string
	Public bool
}

var Secret = Config{Key: "secret"}

func initConfig() {
	if e := db.AutoMigrate(&Config{}); e != nil {
		log.Fatal(e)
	}
	if db.First(&Secret).RowsAffected == 0 {
		Secret.Value = uuid.New().String()
		db.Save(&Secret)
	}
}
