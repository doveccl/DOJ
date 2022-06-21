package db

import (
	"log"

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

func initUser() {
	if e := db.AutoMigrate(&User{}); e != nil {
		log.Fatal(e)
	}
	if db.First(&root).RowsAffected == 0 {
		root.Name = "admin"
		pass, _ := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.MinCost)
		root.Auth = string(pass)
		db.Save(&root)
	}
}
