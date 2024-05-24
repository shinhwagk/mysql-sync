FROM python:3.12

RUN pip install mysql-connector-python==8.0.29
RUN pip install mysql-replication==1.0.8
RUN pip install cryptography

COPY entrypoint.sh .
RUN chmod +x entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]

CMD ["--help"]
