package util

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"github.com/amirphl/lottery-game/dto"
	"github.com/redis/go-redis/v9"
)

const (
	LotteryPrizeSuffix = "prize"
	NumPrizePerPage    = 3
	MaxTriesExceeded   = "Max tries exceeded"
)

type RedisInstance struct {
	ctx context.Context
	c   *redis.Client
	_   struct{}
}

// TODO What happens in the case of running the function twice with the same key and the same redis client?
func (rdb *RedisInstance) AtomicInc(
	key string,
	exp time.Duration,
	maxReqPerWindow int64,
) error {
	txf := func(tx *redis.Tx) error {
		val, err := tx.Get(rdb.ctx, key).Int64()
		if err != nil && err != redis.Nil {

			return err
		}

		if maxReqPerWindow <= val {

			return errors.New(MaxTriesExceeded)
		}

		val++

		// runs only if the watched keys remain unchanged
		_, err = tx.TxPipelined(rdb.ctx, func(pipe redis.Pipeliner) error {
			// pipe handles the error case
			pipe.Set(rdb.ctx, key, val, exp)

			return nil
		})

		return err
	}

	return rdb.c.Watch(rdb.ctx, txf, key)
}

func (rdb *RedisInstance) ComputeExp(windowLengthInMinutes int64) time.Duration {
	t1 := time.Now().UTC()
	t2 := time.Now().UTC()
	m := t1.Minute()
	i := int(math.Floor(float64(m)/float64(windowLengthInMinutes))) + 1
	m = i*int(windowLengthInMinutes) - m
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
		ctx: ctx,
		c:   c,
	}
}
