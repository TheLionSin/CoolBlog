#!/usr/bin/env bash
set -e

BROKER="${BROKER:-localhost:9092}"

docker exec -i go_blog_kafka bash -lc "\
/opt/kafka/bin/kafka-topics.sh --bootstrap-server $BROKER --create --if-not-exists --topic blog.events --partitions 3 --replication-factor 1 && \
/opt/kafka/bin/kafka-topics.sh --bootstrap-server $BROKER --list \
"
