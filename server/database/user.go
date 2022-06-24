package database

import (
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Name   string
	Mail   string
	Auth   []byte
	Group  uint
	Status string
}

var pass, _ = bcrypt.GenerateFromPassword([]byte("admin"), 10)
var root = User{Name: "admin", Auth: pass, Group: 2}

func initRootUser() {
	if db.Limit(1).Find(&root).RowsAffected == 0 {
		db.Create(&root)
	}
}

func GetUser(u string) *User {
	user := User{}
	if db.Find(&user, "name = ? OR mail = ?", u, u).RowsAffected == 0 {
		return nil
	}
	return &user
}
