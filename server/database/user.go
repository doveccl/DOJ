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
	Uid   uint
	Group uint
	jwt.StandardClaims
}

var nameOrMail = "LOWER(name) = LOWER(?) OR LOWER(mail) = LOWER(?)"

func init() {
	initializers[&User{}] = func() {
		root := &User{Name: "admin", Group: 3}
		if e := root.SetPass("admin"); e == nil {
			db.FirstOrCreate(root, &User{Group: 3})
		}
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
	k := []byte(ConfMap["secret"])
	o := UserJWT{Uid: u.ID, Group: u.Group}
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
		db.Find(user, nameOrMail, s, s)
	}
	return user
}
