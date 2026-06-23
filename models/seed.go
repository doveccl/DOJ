package models

import (
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func EnsureAdmin(db *gorm.DB, name string, mail string, password string) error {
	var count int64
	if err := db.Model(&User{}).Where("admin = ?", true).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	name = strings.TrimSpace(name)
	mail = strings.TrimSpace(mail)
	if name == "" || mail == "" || password == "" {
		return fmt.Errorf("bootstrap admin is required when no admin exists; set DOJ_BOOTSTRAP_ADMIN, DOJ_BOOTSTRAP_MAIL, and DOJ_BOOTSTRAP_PASSWORD")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user := User{
		Name:  name,
		Mail:  mail,
		Auth:  string(hash),
		Admin: true,
	}
	return db.Create(&user).Error
}

func DefaultLanguage() Language {
	return Language{
		ID:     "cpp",
		Name:   "C/C++",
		Source: "main.cc",
		Dockerfile: strings.TrimSpace(`
FROM gcc:14
WORKDIR /src
COPY main.cc main.cc
RUN g++ -std=c++20 -O2 -pipe -static -s main.cc -o /main
CMD ["/main"]
`),
	}
}

func EnsureDefaultLanguage(db *gorm.DB) error {
	var count int64
	if err := db.Model(&Language{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	lang := DefaultLanguage()
	return db.Create(&lang).Error
}

const ProblemIDBase uint = 1000

func EnsureProblemIDBase(db *gorm.DB) error {
	if db.Dialector.Name() != "postgres" {
		return nil
	}
	base := ProblemIDBase - 1
	return db.Exec(
		"SELECT setval(pg_get_serial_sequence('problems','id'), GREATEST((SELECT COALESCE(MAX(id), ?) FROM problems), ?), true)",
		base,
		base,
	).Error
}
