package models

import (
	"fmt"
	"strings"

	"github.com/doveccl/doj/contract/limits"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func EnsureAdmin(db *gorm.DB, name string, mail string, password string) (bool, error) {
	var count int64
	if err := db.Model(&User{}).Where("admin = ?", true).Count(&count).Error; err != nil {
		return false, err
	}
	if count > 0 {
		return false, nil
	}

	name = strings.TrimSpace(name)
	mail = strings.TrimSpace(mail)
	if name == "" || mail == "" || password == "" {
		return false, fmt.Errorf("default admin seed is incomplete")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return false, err
	}
	user := User{
		Name:  name,
		Mail:  mail,
		Auth:  string(hash),
		Admin: true,
	}
	if err := db.Create(&user).Error; err != nil {
		return false, err
	}
	return true, nil
}

func DefaultLanguage() Language {
	return Language{
		ID:        "cpp",
		Name:      "C/C++",
		Source:    "main.cc",
		Image:     "gcc",
		Compile:   "g++ main.cc -o main",
		CompileMS: limits.DefaultLanguageCompileMS,
		Run:       "./main",
	}
}

func EnsureDefaultLanguage(db *gorm.DB) error {
	if err := EnsureDefaultLanguageRuntime(db); err != nil {
		return err
	}
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

func EnsureDefaultLanguageRuntime(db *gorm.DB) error {
	lang := DefaultLanguage()
	return db.Model(&Language{}).
		Where("id = ? AND (image = '' OR compile = '' OR compile_ms <= 0 OR run = '')", lang.ID).
		Updates(map[string]any{"image": lang.Image, "compile": lang.Compile, "compile_ms": lang.CompileMS, "run": lang.Run}).Error
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
