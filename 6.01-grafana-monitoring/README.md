# Exercise 6.01: Grafana Monitoring

Learn how to monitor your Kafka cluster using Grafana, Prometheus, and JMX Exporter.

## What You'll Learn

- Setting up JMX Exporter for Kafka metrics
- Configuring Prometheus to scrape Kafka metrics
- Creating Grafana dashboards for Kafka monitoring
- Understanding key Kafka metrics (throughput, latency, consumer lag, etc.)

## Architecture

This setup includes:
- **Kafka Broker** (KRaft mode): Single broker cluster with JMX enabled
- **JMX Exporter**: Standalone container that connects to Kafka's JMX port and exposes metrics in Prometheus format
- **Prometheus**: Time-series database that scrapes and stores metrics
- **Grafana**: Visualization platform for creating dashboards

## Prerequisites

- Docker and Docker Compose installed

## Starting the Stack

Start all services:

```bash
docker compose up -d
```

Verify all services are running:

```bash
docker compose ps
```

## Accessing the Services

- **Kafka**: `localhost:9092`
- **Kafka JMX**: `localhost:9101`
- **JMX Exporter Metrics**: http://localhost:9404/metrics
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (username: `admin`, password: `admin`)

## Verify Metrics Collection

### Check JMX Exporter

View raw Kafka metrics in Prometheus format:

```bash
curl http://localhost:9404/metrics | grep kafka
```

### Check Prometheus

1. Open http://localhost:9090
2. Go to Status > Targets
3. Verify that the `kafka-jmx` target is UP
4. Try a query: `kafka_server_replicamanager_leadercount`

## Creating Your Dashboard

A starter dashboard file has been created at `grafana/dashboards/kafka-dashboard.json`. You can:

1. Open Grafana at http://localhost:3000
2. Create your dashboard in the UI
3. Export it as JSON (Share > Export > Save to file)
4. Replace the content of `kafka-dashboard.json` with your exported dashboard

### Important Kafka Metrics to Monitor

Here are some key metrics you might want to include in your dashboard:

#### Broker Metrics
- `kafka_server_replicamanager_leadercount` - Number of leader partitions
- `kafka_server_replicamanager_partitioncount` - Total partitions on broker
- `kafka_server_brokertopicmetrics_messagesinpersec` - Messages in per second
- `kafka_server_brokertopicmetrics_bytesinpersec` - Bytes in per second
- `kafka_server_brokertopicmetrics_bytesoutpersec` - Bytes out per second

#### Request Metrics
- `kafka_network_requestmetrics_totaltimems` - Request processing time
- `kafka_network_requestmetrics_requestqueuetimems` - Time in request queue
- `kafka_network_requestmetrics_localtimems` - Local processing time
- `kafka_network_requestmetrics_remotetimems` - Time waiting for follower
- `kafka_network_requestmetrics_responsequeuetimems` - Time in response queue

#### Log Metrics
- `kafka_log_logflushstats_logflushrateandtimems` - Log flush performance
- `kafka_log_log_size` - Log size by topic and partition

#### JVM Metrics
- `java_lang_memory_heapmemoryusage_used` - JVM heap memory usage
- `java_lang_garbagecollector_collectioncount` - GC collection count
- `java_lang_garbagecollector_collectiontime` - GC collection time

## Generating Load

To see metrics in action, generate some Kafka traffic:

```bash
# Create a topic
docker exec -it kafka /opt/kafka/bin/kafka-topics.sh --create \
  --topic test-metrics \
  --bootstrap-server localhost:9092 \
  --partitions 3 \
  --replication-factor 1

# Start a console producer
docker exec -it kafka /opt/kafka/bin/kafka-console-producer.sh \
  --topic test-metrics \
  --bootstrap-server localhost:9092

# In another terminal, start a console consumer
docker exec -it kafka /opt/kafka/bin/kafka-console-consumer.sh \
  --topic test-metrics \
  --bootstrap-server localhost:9092 \
  --from-beginning
```

Type messages in the producer and watch the metrics change in Grafana!

## Dashboard Tips

### Panel Types
- **Time Series**: Best for trends over time (throughput, latency)
- **Stat**: Current values (partition count, leader count)
- **Gauge**: Percentage metrics (CPU, memory usage)
- **Table**: Detailed breakdowns by topic/partition

### PromQL Examples

Average message rate over 5 minutes:
```promql
rate(kafka_server_brokertopicmetrics_messagesinpersec{topic!=""}[5m])
```

Total bytes in/out:
```promql
sum(rate(kafka_server_brokertopicmetrics_bytesinpersec[5m]))
sum(rate(kafka_server_brokertopicmetrics_bytesoutpersec[5m]))
```

Request latency (p99):
```promql
histogram_quantile(0.99, 
  rate(kafka_network_requestmetrics_totaltimems_bucket[5m])
)
```

## Cleanup

Stop and remove all containers:

```bash
docker compose down -v
```

## Next Steps

Continue to [Exercise 6.02: Benchmarking & Performance](../6.02-benchmarking-performance/) to learn how to use Kafka's built-in performance testing tools and visualize results in Grafana.

### Additional Ideas
- Add alerting rules in Prometheus for critical metrics
- Create separate dashboards for different use cases (producer, consumer, operations)
- Monitor consumer lag (requires additional configuration)
- Set up multi-broker cluster monitoring
- Explore Kafka Exporter for additional consumer group metrics

## Resources

- [Kafka Monitoring Documentation](https://kafka.apache.org/documentation/#monitoring)
- [JMX Exporter](https://github.com/prometheus/jmx_exporter)
- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Dashboards](https://grafana.com/grafana/dashboards/)
