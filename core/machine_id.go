package core

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"

	"github.com/melon0826/shaniu/core/storage"
	"github.com/melon0826/shaniu/utils"
)

func protect(appID, id string) string {
	mac := hmac.New(sha256.New, []byte(id))
	mac.Write([]byte(appID))
	return hex.EncodeToString(mac.Sum(nil))
}

func init() {
	GetMachineID()
	storage.Watch(shaniu, "machine_id", func(old, new, key string) *storage.Final {
		if old == "" {
			return nil
		}
		return &storage.Final{
			Now: old,
		}
	})
}

var machine_id = ""

var GetMachineID = func() string {
	var id = ""
	if id == "" {
		id = shaniu.GetString("machine_id")
	}
	if id == "" {
		id = protect(utils.GenUUID(), "shaniu")
		shaniu.Set("machine_id", id)
	}
	machine_id = id
	return id
}
