import mysql.connector
from mysql.connector import Error

def connect_to_mysql(host, port, user, password):
    """ Connect to MySQL database """
    try:
        conn = mysql.connector.connect(host=host, port=port, user=user, password=password)
        if conn.is_connected():
            print(f"Connected to MySQL Server at {host}:{port}")
            return conn
    except Error as e:
        print(f"Error: {e}")
        return None

def get_gtid_set(conn):
    """ Retrieve the GTID set from a MySQL connection """
    try:
        cursor = conn.cursor()
        cursor.execute("SHOW GLOBAL VARIABLES LIKE 'gtid_executed';")
        result = cursor.fetchone()
        if result:
            return result[1]
    except Error as e:
        print(f"Error: {e}")
        return None
    finally:
        cursor.close()

def compare_gtids(gtid_set1, gtid_set2):
    """ Compare two GTID sets and print their status """
    if gtid_set1 == gtid_set2:
        print("GTID sets are identical.")
    else:
        print("GTID sets are different.")
        print(f"GTID Set 1: {gtid_set1}")
        print(f"GTID Set 2: {gtid_set2}")

def main():
    # Connection details for MySQL instances
    config1 = {'host': 'host1', 'port': 3306, 'user': 'user1', 'password': 'pass1'}
    config2 = {'host': 'host2', 'port': 3306, 'user': 'user2', 'password': 'pass2'}

    # Connect to MySQL databases
    conn1 = connect_to_mysql(**config1)
    conn2 = connect_to_mysql(**config2)

    if conn1 and conn2:
        # Get GTID sets from both connections
        gtid_set1 = get_gtid_set(conn1)
        gtid_set2 = get_gtid_set(conn2)

        # Compare GTID sets
        compare_gtids(gtid_set1, gtid_set2)

        # Close connections
        conn1.close()
        conn2.close()

if __name__ == "__main__":
    main()



import subprocess
import threading

def consume_input(input_stream):
    try:
        for line in iter(input_stream.readline, b''):
            print(line.decode().strip())
    finally:
        input_stream.close()

def run_pipeline():
    mysqlbinlog_cmd = ['mysqlbinlog', 'mysql-bin.000001']
    mysqlbinlog_statistics_cmd = ['mysqlbinlog_statistics']
    mysql_cmd = ['mysql', '-u', 'username', '-p', 'password', '-h', 'hostname', 'database_name']

    p1 = subprocess.Popen(mysqlbinlog_cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    p2 = subprocess.Popen(mysqlbinlog_statistics_cmd, stdin=p1.stdout, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    p3 = subprocess.Popen(mysql_cmd, stdin=p2.stdout, stdout=subprocess.PIPE, stderr=subprocess.PIPE)

    # Close the p1 stdout to allow p1 to receive a SIGPIPE if p2 exits
    p1.stdout.close()
    # Close the p2 stdout to allow p2 to receive a SIGPIPE if p3 exits
    p2.stdout.close()

    # Create thread to continuously read and print output from p3
    thread = threading.Thread(target=consume_input, args=(p3.stdout,))
    thread.start()

    # Wait for the thread to finish
    thread.join()

    # Check for errors
    errors = p3.stderr.read().decode()
    if p3.returncode == 0 and not errors:
        print("Pipeline executed successfully!")
    else:
        print("Error in executing pipeline:")
        print(errors)

if __name__ == '__main__':
    run_pipeline()
