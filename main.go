package main

import (
	"fmt"
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
	ProducerConfPath          = "./producer.properties"
	ConsumerConfPath          = "./consumer.properties"
	MaxReqPerWindowName       = "MAX_REQ_PER_WINDOW"
	WindowLengthInMinutesName = "WINDOW_LENGTH_IN_MINUTES"
	NumConsumers              = 4
)

func readEnv(key string) (int64, error) {
	val := os.Getenv(key)
	if val == "" {

		return 0, fmt.Errorf("Failed to read env var %s", key)
	}

	res, err := strconv.ParseInt(val, 10, 64)
	if err != nil {

		return 0, fmt.Errorf("Failed to read env var %s: %s", key, val)
	}

	return res, nil
}

func runProducer() {
	maxReqPerWindow, err := readEnv(MaxReqPerWindowName)
	if err != nil {
		log.Println(err.Error())
		os.Exit(1)
	}

	windowLengthInMinutes, err := readEnv(WindowLengthInMinutesName)
	if err != nil {
		log.Println(err.Error())
		os.Exit(1)
	}

	conf := util.ReadConfig(ProducerConfPath)
	rdb := util.NewRedisInstance()
	kaf := util.NewKafkaProducerInstance(conf)
	r := mux.NewRouter()

	lotteryHandler := controller.NewLotteryHandler(rdb, kaf, maxReqPerWindow, windowLengthInMinutes)
	prizeHandler := controller.NewPrizeHandler(rdb)

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
	close(kaf.SigChan)
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
