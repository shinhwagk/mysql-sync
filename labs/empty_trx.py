import mysql.connector

with mysql.connector.connect(host="db1", user="root", password="root_password") as con:
    con.start_transaction()
    with con.cursor() as cur:
        cur.execute(f"begin")
        cur.execute("")
        cur.execute("commit;")
    con.commit()
