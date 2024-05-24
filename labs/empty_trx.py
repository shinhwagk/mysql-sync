import mysql.connector

with mysql.connector.connect(host="db1", user="root", password="root_password") as con:
    con.start_transaction()
    with con.cursor(buffered=True) as cur:
        cur.execute("SET SESSION TRANSACTION READ ONLY")
        cur.execute("select * from test.abc")
        cur.execute("commit")

    con.commit()
