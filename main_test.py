# statements = [
#     "CREATE TABLE xxx.`myschema`.`mytable` ",
#     "create table myschema.mytable ",
#     "CREATE TABLE `schema`.table ",
#     "create table `table` ",
#     "create table table ",
#     "CREATE TABLE myschema.`table` ",
#     "create \n table `schema`.`table` ",
#     "create \n table `table` ",
# ]


# import main

# for st in statements:
#     s, t = main.tiquddl_schema_tab(st)
#     print(s, t)

import mysql.connector

import main

with mysql.connector.connect(host="db1", user="root", password="root_password") as con:
    with con.cursor(buffered=True) as cur:
        cur.execute("show master status")
        stat = cur.fetchone()
        log_file, log_pos, _, _, gtidsets = stat
        for gtidset in gtidsets.split(","):
            gtidset_split = gtidset.split(":")
            server_uuid = gtidset_split[0]
            last_xid = gtidset_split[-1]


ckt = main.Checkpoint("asdfsdfsdfsdf:1-100,adfsfsdf:9-10")

print(ckt.checkpoint_gtidset)
