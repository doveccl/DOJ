package database

import (
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Name   string
	Mail   string
	Auth   string
	Group  uint
	Status string
}

var pass, _ = bcrypt.GenerateFromPassword([]byte("admin"), 10)
var root = User{Name: "admin", Auth: string(pass), Group: 2}

func initRootUser() {
	if db.Limit(1).Find(&root).RowsAffected == 0 {
		db.Create(&root)
	}
}
