# to kafka
gcc tokafka.c -o kafka_producer -lrdkafka



export KAFKA_BROKERS=redpanda-0:19092
export KAFKA_TOPIC=mysqlbinlog
./kafka_producer
