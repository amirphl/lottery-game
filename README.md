## Interview Question
Implement a Lottery Game  
A system that our users can use their chances to win prizes. We have 5 prizes A, B, C, D, and E with different weights 0.1,
0.3, 0.2, 0.15, and 0.25 respectively. Also, we have registered users in our Redis database who are allowed to participate in
the lottery only 3 times a day. Suppose the lottery process is a heavy process and takes about 5 seconds.
We want to implement an asynchronous service that accepts the incoming HTTP requests from the clients and performs
the lottery, at last, it gives the users the prize as a response.  
Request details:  
POST api/v1/lottery  
BODY:  
{  
UUID: “f1bc8f04-0500-11ee-be56-0242ac120002” // UUID of the user  
}  
Implementation criteria:  
1.Layer and directory structure according to the standard of the language you decided to use  
2.Use docker compose to run all of the services  
3.Writing unit tests is a plus  

## To run
- `docker build . -t lotterygame`
- `docker compose up -d`
- To submit a request for user `53800cbd-a509-4091-9122-32f1e5c0ac88`:
    - `curl localhost:8080/api/v1/lottery -X POST -d '{"user_id":"53800cbd-a509-4091-9122-32f1e5c0ac88"}' -v`
- To receive the list of prizes:
    - `curl 'localhost:8080/api/v1/prize?user=53800cbd-a509-4091-9122-32f1e5c0ac88&page=1' -X GET -v`
    - page=0 means the last three prizes, page=1 means the next three prizes, and so on.

## If you run outside of docker
- `sudo apt-get install build-essential pkg-config`
- `sudo apt-get install librdkafka-dev`
- TODO (change producer.properties and consumers.properties)

## Considerations
- Kafka
    - Use a kafka cluster for high availability and increasing performace.
    - Employ sharding technique.
    - ack = 0 | 1 | all? This affects both the performance and the integrity of the code.
    - When should the consumers commit?
    - Use consumer groups for high availability of the consumers.
- Redis
    - A cluster of redis instances
    - Persist data to prevent data loss.


## TODO
- setup Nginx + load balancing
- use transactions for limiting request rate using redis
- perform authentication on endpoints
- write unit tests
- TODOs in the codebase
- improve env var handling
- remove .properties files and read from env
- graceful stopping consumers
