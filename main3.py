from pymysqlreplication import BinLogStreamReader
from pymysqlreplication.event import GtidEvent

mysql_settings = {'host': '172.16.0.179', 'port': 3306, 'user': 'ghost', 'passwd': '54448hotINBOX'}

stream = BinLogStreamReader(connection_settings = mysql_settings, server_id=9990,log_file="mysql-bin.000220",log_pos=4 ,blocking=True)

for binlogevent in stream:
    # binlogevent.dump()
    if type(binlogevent) == GtidEvent:
        print(binlogevent.gtid)

stream.close()

# docker run --log-opt max-size=1m --log-opt max-file=1 python3.12 