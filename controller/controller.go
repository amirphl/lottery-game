package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/amirphl/lottery-game/dto"
	"github.com/amirphl/lottery-game/item"
	"github.com/amirphl/lottery-game/util"
	"github.com/confluentinc/confluent-kafka-go/kafka"
)

type LotteryHandler struct {
	ctx              context.Context
	rdb              *util.RedisInstance
	kaf              *util.KafkaProducerInstance
	maxReqsPerWindow int64
}

type PrizeHandler struct {
	ctx context.Context
	rdb *util.RedisInstance
}

func (h *LotteryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var user dto.User

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)

		return
	}

	if err := user.Validate(); err != nil {
		http.Error(w, "Failed to parse request body", http.StatusBadRequest)

		return
	}

	// TODO auth
	// TODO transaction
	val, err := h.rdb.GetNumTries(user)
	if err != nil {
		log.Printf("Error while reading redis: %s\n", err.Error())
		http.Error(w, "", http.StatusInternalServerError)

		return
	}

	numTries, _ := strconv.ParseInt(val, 10, 64)
	if h.maxReqsPerWindow <= numTries {
		http.Error(w, "Max tries exceeded", http.StatusTooManyRequests)

		return
	}

	exp := h.rdb.ComputeNextExp()
	if err := h.rdb.SetNumTries(user, numTries+1, exp); err != nil {
		log.Printf("Error while writing redis: %s\n", err.Error())
		http.Error(w, "", http.StatusInternalServerError)

		return
	}

	resChan := make(chan kafka.Event)
	defer close(resChan)
	h.kaf.Produce(user, resChan)
	e := <-resChan // TODO exit after some seconds
	switch ev := e.(type) {
	case *kafka.Message:
		if ev.TopicPartition.Error != nil {
			log.Printf("Failed to deliver message: %v\n", ev.TopicPartition)
			http.Error(w, "", http.StatusInternalServerError)

			return
		} else {
			log.Printf("Produced event to topic %s: key = %-10s value = %s\n",
				*ev.TopicPartition.Topic, string(ev.Key), string(ev.Value))
		}
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *PrizeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	user, err := h.extractUser(params)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)

		return
	}

	page := h.extractPage(params)

	// TODO auth
	strs, err := h.rdb.RangePrizes(user, page)
	if err != nil {
		log.Printf("Error while reading redis: %s\n", err.Error())
		http.Error(w, "", http.StatusInternalServerError)

		return
	}

	prizes := []item.Prize{}
	for _, v := range strs {
		var p item.Prize
		json.Unmarshal([]byte(v), &p)
		prizes = append(prizes, p)
	}

	if err := json.NewEncoder(w).Encode(prizes); err != nil {
		log.Printf("Error while serializing: %s\n", err.Error())
		http.Error(w, "", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")
}

func (h *PrizeHandler) extractUser(params url.Values) (dto.User, error) {
	if len(params["user"]) == 0 {

		return dto.User{}, fmt.Errorf("User not found")
	}

	return dto.User{
		UUID: params["user"][0],
	}, nil
}

func (h *PrizeHandler) extractPage(params url.Values) int64 {
	if len(params["page"]) > 0 {
		if page, err := strconv.ParseInt(params["page"][0], 10, 64); err == nil {

			return page
		}
	}

	return 0
}

func NewLotteryHandler(
	rdb *util.RedisInstance,
	kaf *util.KafkaProducerInstance,
	maxReqPerWindow int64,
) *LotteryHandler {

	return &LotteryHandler{
		ctx:              context.Background(),
		rdb:              rdb,
		kaf:              kaf,
		maxReqsPerWindow: maxReqPerWindow,
	}
}

func NewPrizeHandler(
	rdb *util.RedisInstance,
) *PrizeHandler {

	return &PrizeHandler{
		ctx: context.Background(),
		rdb: rdb,
	}
}
