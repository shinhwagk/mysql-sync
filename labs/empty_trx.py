import mysql.connector

with mysql.connector.connect(host="db1", user="root", password="root_password") as con:
    con.start_transaction()
    cur1 = con.cursor(buffered=True)
    cur2 = con.cursor(buffered=True)
    cur1.execute("set autocommit=0")
    cur2.execute("commit;")
    con.commit()
