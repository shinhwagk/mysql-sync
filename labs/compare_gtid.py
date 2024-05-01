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
