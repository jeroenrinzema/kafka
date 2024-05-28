# Kubernetes

## Strimzi

```bash
$ kubectl create namespace kafka
$ kubectl create -f 'https://strimzi.io/install/latest?namespace=kafka' -n kafka

$ kubectl apply -f ./simple -n kafka 
$ kubectl apply -f ./tls-auth -n kafka 
```

## Kafka UI

```bash
$ helm repo add kafka-ui https://provectus.github.io/kafka-ui-charts
$ helm install meterstanden kafka-ui/kafka-ui -f values.yaml
```