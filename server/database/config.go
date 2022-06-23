package database

import (
	"github.com/google/uuid"
)

type Config struct {
	Key    string `gorm:"primaryKey"`
	Value  string
	Public bool
}

var PublicConfigs = map[string]string{
	"registration": "1",
}

var PrivateConfigs = map[string]string{
	"secret": uuid.New().String(),
}

func (c *Config) Sync() string {
	if db.Find(c).RowsAffected == 0 {
		db.Create(c)
	}
	return c.Value
}

func syncConfigs() {
	for k, v := range PublicConfigs {
		PublicConfigs[k] = (&Config{k, v, true}).Sync()
	}
	for k, v := range PrivateConfigs {
		PrivateConfigs[k] = (&Config{k, v, false}).Sync()
	}
}
