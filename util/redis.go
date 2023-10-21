package util

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/amirphl/lottery-game/dto"
	"github.com/redis/go-redis/v9"
)

const (
	LotteryPrizeSuffix        = "prize"
	NumPrizePerPage           = 3
	WindowLengthInMinutesName = "WINDOW_LENGTH_IN_MINUTES"
)

type RedisInstance struct {
	ctx                   context.Context
	c                     *redis.Client
	windowLengthInMinutes int
	_                     struct{}
}

func (rdb *RedisInstance) GetNumTries(user dto.User) (string, error) {
	val, err := rdb.c.Get(rdb.ctx, user.UUID).Result()

	if err == redis.Nil {
		val = "0"
		err = nil
	}

	return val, err
}

func (rdb *RedisInstance) SetNumTries(user dto.User, numTries int64, exp time.Duration) error {
	val := strconv.FormatInt(numTries, 10)

	return rdb.c.Set(rdb.ctx, user.UUID, val, exp).Err()
}

func (rdb *RedisInstance) ComputeNextExp() time.Duration {
	t1 := time.Now().UTC()
	t2 := time.Now().UTC()
	m := t1.Minute()
	i := int(math.Floor(float64(m)/float64(rdb.windowLengthInMinutes))) + 1
	m = i*rdb.windowLengthInMinutes - m
	t1 = t1.Add(time.Duration(m) * time.Minute).Truncate(time.Minute)
	t1 = t1.Add(1 * time.Millisecond) // to prevent zero value for exp
	exp := t1.Sub(t2)

	return exp
}

func (rdb *RedisInstance) PushPrize(user dto.User, prize []byte) error {
	key := buildPrizeKey(user)
	_, err := rdb.c.RPush(rdb.ctx, key, prize).Result()

	return err
}

func (rdb *RedisInstance) RangePrizes(user dto.User, page int64) ([]string, error) {
	key := buildPrizeKey(user)
	start := -NumPrizePerPage * (page + 1)
	end := (-NumPrizePerPage * page) - 1

	return rdb.c.LRange(rdb.ctx, key, start, end).Result()
}

func buildPrizeKey(user dto.User) string {
	return fmt.Sprintf("%s-%s", user.UUID, LotteryPrizeSuffix)
}

func NewRedisInstance() *RedisInstance {
	val := os.Getenv(WindowLengthInMinutesName)
	if val == "" {
		log.Printf("Failed to read env var %s", WindowLengthInMinutesName)
		os.Exit(1)
	}

	windowLengthInMinutes, err := strconv.ParseInt(val, 10, 32)
	if err != nil {
		log.Printf("Failed to read env var %s: %s", WindowLengthInMinutesName, val)
		os.Exit(1)
	}

	c := redis.NewClient(&redis.Options{
		Addr:     "redis:6379", // TODO read from env
		Password: "",           // no password set
		DB:       0,            // use default DB
	})

	ctx := context.Background()
	if _, err := c.Ping(ctx).Result(); err != nil {
		log.Printf("Failed to ping redis: %s", err.Error())
		os.Exit(1)
	}

	return &RedisInstance{
		ctx:                   ctx,
		c:                     c,
		windowLengthInMinutes: int(windowLengthInMinutes),
	}
}
