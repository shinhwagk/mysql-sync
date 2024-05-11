#include <librdkafka/rdkafka.h>
#include <stdio.h>
#include <string.h>

int main() {
    rd_kafka_t *rk; // Producer instance
    rd_kafka_conf_t *conf; // Configuration object
    char errstr[512]; // Buffer for error messages
    const char *brokers = "localhost:9092"; // Kafka broker list
    const char *topic = "test"; // Kafka topic to produce to

    // Create a new configuration object
    conf = rd_kafka_conf_new();

    // Set the bootstrap broker(s) and other config options
    if (rd_kafka_conf_set(conf, "bootstrap.servers", brokers, errstr, sizeof(errstr)) != RD_KAFKA_CONF_OK) {
        fprintf(stderr, "%% rd_kafka_conf_set error: %s\n", errstr);
        return 1;
    }

    // Create a new producer instance
    rk = rd_kafka_new(RD_KAFKA_PRODUCER, conf, errstr, sizeof(errstr));
    if (!rk) {
        fprintf(stderr, "Failed to create producer: %s\n", errstr);
        return 1;
    }

    // Continuously read lines from stdin
    printf("Enter your messages (Ctrl+D to exit):\n");
    char buf[1024];
    while (fgets(buf, sizeof(buf), stdin)) {
        size_t len = strlen(buf);
        if (buf[len - 1] == '\n') {
            buf[len - 1] = '\0';
        }

        // Produce a message
        if (rd_kafka_producev(
            rk,
            RD_KAFKA_V_TOPIC(topic),
            RD_KAFKA_V_MSGFLAGS(RD_KAFKA_MSG_F_COPY),
            RD_KAFKA_V_VALUE(buf, strlen(buf)),
            RD_KAFKA_V_END
        ) == -1) {
            fprintf(stderr, "Failed to produce to topic %s: %s\n", topic, rd_kafka_err2str(rd_kafka_last_error()));
            rd_kafka_poll(rk, 0); // Handle delivery reports
            continue;
        }

        fprintf(stdout, "Message sent: %s\n", buf);
        rd_kafka_poll(rk, 0); // Handle delivery reports
    }

    // Wait up to 1 second for any outstanding messages to be delivered
    rd_kafka_flush(rk, 1000);

    // Clean up
    rd_kafka_destroy(rk);

    return 0;
}
