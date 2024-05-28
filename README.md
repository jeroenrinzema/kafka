# Apache Kafka

## Running Apache Kafka locally

```bash
$ docker-compose -f 01-single-broker.yaml up
$ docker-compose -f 02-kafka-ui.yaml up
$ docker-compose -f 03-schema-registry.yaml up
```

## Apache Kafka bin

https://kafka.apache.org/downloads

```bash
$ ./kafka-topics.sh --bootstrap-server 127.0.0.1:9092 --list
$ ./kafka-e2e-latency.sh localhost:9092 purchases 10000 1 20
$ ./kafka-producer-perf-test.sh --producer-props bootstrap.servers=localhost:9092 --topic purchases --num-records 50 --throughput 10 --record-size 100
```