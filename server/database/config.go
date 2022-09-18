package database

import "github.com/google/uuid"

type Config struct {
	Key    string `gorm:"primaryKey"`
	Value  string
	Public bool
}

var ConfMap = map[string]string{}

var ConfList = []*Config{
	{"registration", "1", true},
	{"secret", uuid.New().String(), false},
}

func init() {
	initializers[&Config{}] = func() {
		for _, c := range ConfList {
			c.Sync()
			ConfMap[c.Key] = c.Value
		}
	}
}

func (c *Config) Sync() string {
	db.FirstOrCreate(c)
	return c.Value
}
