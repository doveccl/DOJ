package database

import (
	"time"

	"github.com/golang-jwt/jwt"
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

type UserJWT struct {
	ID    uint
	Group uint
	jwt.StandardClaims
}

var root = User{Name: "admin", Group: 3}

func init() {
	if e := root.SetPass("admin"); e != nil {
		panic(e)
	}
}

func initRootUser() {
	if db.Limit(1).Find(&root).RowsAffected == 0 {
		db.Create(&root)
	}
}

func (u *User) SetPass(p string) (e error) {
	u.Auth, e = bcrypt.GenerateFromPassword([]byte(p), 10)
	return
}

func (u *User) Compare(p string) bool {
	return bcrypt.CompareHashAndPassword(u.Auth, []byte(p)) == nil
}

func (u *User) GetToken() (string, error) {
	m := jwt.SigningMethodHS256
	k := []byte(PrivateConfigs["secret"])
	o := UserJWT{ID: u.ID, Group: u.Group}
	o.ExpiresAt = time.Now().Add(720 * time.Hour).Unix()
	return jwt.NewWithClaims(m, o).SignedString(k)
}

func (u *User) Create() bool {
	return db.Create(u).RowsAffected == 1
}

func GetUser(u any) *User {
	user := &User{}
	if i, _ := u.(uint); i != 0 {
		db.Find(user, i)
	} else if s, _ := u.(string); s != "" {
		db.Find(user, "name = ? OR mail = ?", s, s)
	}
	return user
}
