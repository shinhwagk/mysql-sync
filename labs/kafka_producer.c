#include <librdkafka/rdkafka.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

/* 错误处理回调 */
static void error_cb(rd_kafka_t *rk, int err, const char *reason, void *opaque)
{
    fprintf(stderr, "ERROR: %s: %s\n", rd_kafka_err2str(err), reason);
}

/* 消息传递报告回调 */
static void dr_msg_cb(rd_kafka_t *rk, const rd_kafka_message_t *rkmessage, void *opaque)
{
    if (rkmessage->err)
    {
        fprintf(stderr, "Message delivery failed: %s\n", rd_kafka_err2str(rkmessage->err));
    }
    else
    {
        fprintf(stdout, "Message delivered (offset: %" PRId64 ")\n", rkmessage->offset);
    }
}
int main()
{
    rd_kafka_t *rk;
    rd_kafka_conf_t *conf;
    char errstr[512];
    const char *brokers = "172.19.0.6:19092"; // "getenv("KAFKA_BROKERS")";
    const char *topic = "test";               // getenv("KAFKA_TOPIC");

    conf = rd_kafka_conf_new();

    if (rd_kafka_conf_set(conf, "bootstrap.servers", brokers, errstr, sizeof(errstr)) != RD_KAFKA_CONF_OK)
    {
        fprintf(stderr, "%% rd_kafka_conf_set error: %s\n", errstr);
        return 1;
    }

    /* 启用幂等生产者 */
    if (rd_kafka_conf_set(conf, "enable.idempotence", "true", errstr, sizeof(errstr)) != RD_KAFKA_CONF_OK)
    {
        fprintf(stderr, "%s\n", errstr);
        return 1;
    }

    /* 设置错误回调 */
    rd_kafka_conf_set_error_cb(conf, error_cb);

    /* 设置消息传递报告回调 */
    rd_kafka_conf_set_dr_msg_cb(conf, dr_msg_cb);

    rk = rd_kafka_new(RD_KAFKA_PRODUCER, conf, errstr, sizeof(errstr));
    if (!rk)
    {
        fprintf(stderr, "Failed to create producer: %s\n", errstr);
        return 1;
    }

    printf("Enter your messages (Ctrl+D to exit):\n");
    char buf[1024];
    while (fgets(buf, sizeof(buf), stdin))
    {
        size_t len = strlen(buf);
        rd_kafka_resp_err_t err;
        if (buf[len - 1] == '\n')
        {
            buf[len - 1] = '\0';
        }

        err = rd_kafka_producev(
            rk,
            RD_KAFKA_V_TOPIC(topic),
            RD_KAFKA_V_MSGFLAGS(RD_KAFKA_MSG_F_COPY),
            RD_KAFKA_V_VALUE(buf, strlen(buf)),
            RD_KAFKA_V_OPAQUE(NULL),
            RD_KAFKA_V_END);

        if (err)
        {
            fprintf(stderr,
                    "%% Failed to produce to topic %s: %s\n", topic,
                    rd_kafka_err2str(err));

            if (err == RD_KAFKA_RESP_ERR__QUEUE_FULL)
            {
                rd_kafka_poll(rk, 1000);
            }
        }

        // fprintf(stdout, "Message sent: %s\n", buf);
        rd_kafka_poll(rk, 1);
    }

    rd_kafka_flush(rk, 10000);

    rd_kafka_destroy(rk);

    return 0;
}
