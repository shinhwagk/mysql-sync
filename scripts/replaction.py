from pymysqlreplication import BinLogStreamReader

mysql_settings = {'host': 'db1', 'port': 3306, 'user': 'root', 'passwd': 'root_password'}

stream = BinLogStreamReader(connection_settings = mysql_settings, server_id=100)

for binlogevent in stream:
    binlogevent.dump()

stream.close()