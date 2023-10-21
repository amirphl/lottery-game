package service

import (
	"github.com/amirphl/lottery-game/dto"
	"github.com/amirphl/lottery-game/util"
)

const UsersSetRedis = "lottery-users"

func Authenticate(rdb *util.RedisInstance, user dto.User) bool {
	exists, err := rdb.Exists(UsersSetRedis, user.UUID)
	if err != nil {

		return false
	}

	return exists
}
