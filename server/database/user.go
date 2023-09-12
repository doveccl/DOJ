package database

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
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
	UID   uint
	Group uint
	jwt.RegisteredClaims
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
	o := UserJWT{UID: u.ID, Group: u.Group}
	o.ExpiresAt = jwt.NewNumericDate(time.Now().Add(720 * time.Hour))
	return jwt.NewWithClaims(m, o).SignedString(k)
}

func (u *User) Create() bool {
	return db.Create(u).RowsAffected == 1
}

func GetUser(u any) *User {
	user := &User{}
	if s, _ := u.(string); s != "" {
		db.Find(user, nameOrMail, s, s)
	} else { // whatever int or float
		db.Find(user, u)
	}
	return user
}
