## docker-compose.yml
```yml
services:
  repl:
    image: shinhwagk/mysql-sync:consul-0.2.60
    command: -repl
    volumes:
      - ./config.yml:/etc/mysqlsync/config.yml
    depends_on:
      - consul
  dest:
    image: shinhwagk/mysql-sync:consul-0.2.60
    command: -dest -name sc_db3_3317
    volumes:
      - ./config.yml:/etc/mysqlsync/config.yml
    depends_on:
      - repl
      - consul
  consul:
    image: hashicorp/consul:1.19.1
```

## config.yml
```yml
replication:
  name: "xxxx"
  tcpaddr: "0.0.0.0:9998"
  serverid: 9999
  host: "db1"
  port: 3306
  user: "root"
  password: "root_password"
  loglevel: 1
  settings:
    cache: 1000
  prom:
    export: 9091
destination:
  tcpaddr: "127.0.0.1:9998"
  cache: 1000
  destinations:
    db2: # custom dest name
      mysql:
        dsn: "root:root_password@tcp(db2:3306)/"
        skip_errors: 1007,1008,1050,1051,1054,1060,1061,1068,1091,1146
        session_params: # map[string]string
          foreign_key_checks: off
      sync:
        replicate:
          do_db: test
          ignore_tab: test.year_table
        gtidsets: ""
      prom:
        export: 9092
      loglevel: 1
    db3:
      mysql:
        dsn: "root:root_password@tcp(db3:3306)/"
        skip_errors: 1007,1008,1050,1051,1054,1060,1061,1068,1091,1146
      sync:
        gtidsets: ""
      loglevel: 1
      prom:
        export: 9093
consul:
  addr: consul:8500
```