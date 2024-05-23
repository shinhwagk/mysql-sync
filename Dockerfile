FROM python:3.12-slim

RUN pip install mysql-connector-python==8.0.29
RUN pip install mysql-replication==1.0.8
RUN pip install cryptography

WORKDIR /app
COPY main1.py main.py

ENTRYPOINT ["python", "/app/main.py"]

CMD ["--help"]
