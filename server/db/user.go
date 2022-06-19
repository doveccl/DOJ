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

func initUsers() {
	if e := db.AutoMigrate(&User{}); e != nil {
		log.Fatal(e)
	}
	cnt, user := int64(0), User{}
	db.Model(&user).Count(&cnt)
	if cnt == 0 {
		user.Name = "admin"
		pass, _ := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.MinCost)
		user.Auth = string(pass)
		user.Group = 2
		db.Create(&user)
	}
}
