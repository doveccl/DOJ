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

var root = User{Group: 2}
var pass, _ = bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.MinCost)

func createRootUser() {
	if db.Limit(1).Find(&root).RowsAffected == 0 {
		root.Name = "admin"
		root.Auth = string(pass)
		db.Create(&root)
	}
}
