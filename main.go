package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/amirphl/lottery-game/controller"
	"github.com/amirphl/lottery-game/item"
	"github.com/amirphl/lottery-game/util"
	"github.com/gorilla/mux"
)

const (
	ProducerConfPath    = "./producer.properties"
	ConsumerConfPath    = "./consumer.properties"
	MaxReqPerWindowName = "MAX_REQ_PER_WINDOW"
	NumConsumers        = 4
)

func runProducer() {
	val := os.Getenv(MaxReqPerWindowName)
	if val == "" {
		log.Printf("Failed to read env var %s", MaxReqPerWindowName)
		os.Exit(1)
	}

	maxReqPerWindow, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		log.Printf("Failed to read env var %s: %s", MaxReqPerWindowName, val)
		os.Exit(1)
	}

	conf := util.ReadConfig(ProducerConfPath)
	rdb := util.NewRedisInstance()
	kaf := util.NewKafkaProducerInstance(conf)
	r := mux.NewRouter()

	lotteryHandler := &controller.LotteryHandler{
		CTX:              context.Background(),
		RDB:              rdb,
		KAF:              kaf,
		MaxReqsPerWindow: maxReqPerWindow,
	}
	prizeHandler := &controller.PrizeHandler{
		CTX: context.Background(),
		RDB: rdb,
	}
	// TODO CORS Token
	// TODO group APIs
	r.Handle("/api/v1/lottery", lotteryHandler).Methods("POST")
	r.Handle("/api/v1/prize", prizeHandler).Methods("GET")

	log.Fatal(http.ListenAndServe(":8080", r))
}

func runConsumer() {
	conf := util.ReadConfig(ConsumerConfPath)
	rdb := util.NewRedisInstance()
	kaf := util.NewkafkaConsumerInstance(conf)

	var wg sync.WaitGroup
	wg.Add(NumConsumers)

	for i := 0; i < NumConsumers; i++ {
		go item.HeavyProcess(kaf, rdb, &wg)
	}

	wg.Wait()
	kaf.C.Close()
	close(kaf.SIGCH)
}

func main() {
	mod := os.Getenv("MODE")

	if mod == "producer" {
		runProducer()

	} else if mod == "consumer" {
		runConsumer()
	} else {
		log.Printf("env var MODE is not set")
		os.Exit(1)
	}
}
