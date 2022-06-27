package database

import (
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Name   string `gorm:"uniqueIndex"`
	Mail   string `gorm:"uniqueIndex"`
	Auth   []byte `json:"-"`
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

func GetUser(u any) *User {
	user := &User{}
	if i, _ := u.(uint); i != 0 {
		db.Find(user, i)
	} else if s, _ := u.(string); s != "" {
		db.Find(user, "name = ? OR mail = ?", s, s)
	}
	if user.ID == 0 {
		return nil
	}
	return user
}
