package util

import (
	"bufio"
	"context"
	"github.com/amirphl/lottery-game/dto"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var Topic string = "lottery"

type KafkaProducerInstance struct {
	ctx context.Context
	p   *kafka.Producer
	_   struct{}
}

type KafkaConsumerInstance struct {
	ctx     context.Context
	C       *kafka.Consumer
	SigChan chan<- os.Signal
	ResChan <-chan *kafka.Message
	_       struct{}
}

func (kaf *KafkaProducerInstance) Produce(user dto.User, resChan chan kafka.Event) {
	key := []byte(user.UUID)
	val := key

	kaf.p.Produce(
		&kafka.Message{
			TopicPartition: kafka.TopicPartition{
				Topic:     &Topic,
				Partition: kafka.PartitionAny,
			},
			Key:   key,
			Value: val,
		},
		resChan,
	)
}

func ReadConfig(configFile string) kafka.ConfigMap {
	conf := make(map[string]kafka.ConfigValue)

	file, err := os.Open(configFile)
	if err != nil {
		log.Printf("Failed to open file: %s", err.Error())
		os.Exit(1)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "#") && len(line) != 0 {
			before, after, found := strings.Cut(line, "=")
			if found {
				parameter := strings.TrimSpace(before)
				value := strings.TrimSpace(after)
				conf[parameter] = value
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Failed to read file: %s", err.Error())
		os.Exit(1)
	}

	return conf
}

func NewKafkaProducerInstance(conf kafka.ConfigMap) *KafkaProducerInstance {
	p, err := kafka.NewProducer(&conf)

	if err != nil {
		log.Printf("Failed to create producer: %s", err.Error())
		os.Exit(1)
	}

	go func() {
		for e := range p.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					log.Printf("Failed to deliver message: %v\n", ev.TopicPartition)
				} else {
					log.Printf("Produced event to topic %s: key = %-10s value = %s\n",
						*ev.TopicPartition.Topic, string(ev.Key), string(ev.Value))
				}
			}
		}
	}()

	return &KafkaProducerInstance{
		ctx: context.Background(),
		p:   p,
	}
}

func NewkafkaConsumerInstance(conf kafka.ConfigMap) *KafkaConsumerInstance {
	c, err := kafka.NewConsumer(&conf)

	if err != nil {
		log.Printf("Failed to create consumer: %s", err.Error())
		os.Exit(1)
	}

	err = c.SubscribeTopics([]string{Topic}, nil)
	// Set up a channel for handling Ctrl-C, etc
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	resChan := make(chan *kafka.Message)

	kaf := &KafkaConsumerInstance{
		ctx:     context.Background(),
		C:       c,
		SigChan: sigChan,
		ResChan: resChan,
	}

	go func() {
		run := true
		for run {
			select {
			case sig := <-sigChan:
				log.Printf("Caught signal %v: terminating\n", sig)
				run = false
			default:
				ev, err := c.ReadMessage(100 * time.Millisecond)
				if err != nil {
					// Errors are informational and automatically handled by the consumer
					continue
				}
				log.Printf("Consumed event from topic %s: key = %-10s value = %s time = %v\n",
					*ev.TopicPartition.Topic, string(ev.Key), string(ev.Value), ev.Timestamp)
				resChan <- ev
			}
		}
		close(resChan)
	}()

	return kaf
}
