from kafka import KafkaProducer

producer = KafkaProducer(bootstrap_servers="redpanda-0:19092")
for _ in range(100):
    future = producer.send("test", b"another_message")
    result = future.get(timeout=1)
producer.flush()
