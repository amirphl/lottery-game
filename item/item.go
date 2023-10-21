package item

import (
	"encoding/json"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/amirphl/lottery-game/dto"
	"github.com/amirphl/lottery-game/util"
)

var Items = []Item{
	{Name: "A", Prob: 0.1},
	{Name: "B", Prob: 0.3},
	{Name: "C", Prob: 0.2},
	{Name: "D", Prob: 0.15},
	{Name: "E", Prob: 0.25},
}

type Item struct {
	Name string
	Prob float32
	_    struct{}
}

type Prize struct {
	Item Item
	Time time.Time
	_    struct{}
}

func PickOne() Item {
	p := rand.Float32()
	cdf := float32(0)

	for _, item := range Items {
		cdf += item.Prob
		if p <= cdf {

			return item
		}
	}

	return Items[0]
}

func HeavyProcess(
	kaf *util.KafkaConsumerInstance,
	rdb *util.RedisInstance,
	wg *sync.WaitGroup,
) {
	for ev := range kaf.ResChan {
		user := dto.User{
			UUID: string(ev.Key),
		}
		item := PickOne()
		prize := Prize{
			Item: item,
			Time: time.Now(),
		}
		prizeB, _ := json.Marshal(prize)
		if err := rdb.PushPrize(user, prizeB); err != nil {
			log.Printf("Error while writing redis: %s\n", err.Error())
		}
	}

	wg.Done()
}
