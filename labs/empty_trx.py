import mysql.connector

with mysql.connector.connect(host="db1", user="root", password="root_password") as con:
    # con.start_transaction()
    with con.cursor() as cur:
        cur.execute(f"SET autocommit=0")
        cur.execute("insert into test.abc2 values(3)")
        cur.execute("commit")
        cur.execute("SET autocommit=1")
        cur.execute("SELECT @@session.transaction_read_only")
    # con.commit()
