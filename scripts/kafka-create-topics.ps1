param(
  [string]$KafkaContainer = "go_blog_kafka",
  [string]$Bootstrap = "localhost:9092"
)

$cmd = @"
 /opt/kafka/bin/kafka-topics.sh --bootstrap-server $Bootstrap --create --if-not-exists --topic blog.events --partitions 3 --replication-factor 1
 /opt/kafka/bin/kafka-topics.sh --bootstrap-server $Bootstrap --create --if-not-exists --topic blog.events.dlq --partitions 1 --replication-factor 1
 /opt/kafka/bin/kafka-topics.sh --bootstrap-server $Bootstrap --list
"@

docker exec -i $KafkaContainer bash -lc $cmd
