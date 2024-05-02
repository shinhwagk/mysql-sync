import random


def generate_creates(num_creates):
    creates = []
    databases = []
    tables = []
    for i in range(num_creates):
        db_name = f"database_{i+1}"
        table_name = f"table_{i+1}"
        # Remove AUTO_INCREMENT and set up a PRIMARY KEY only
        creates.append(f"CREATE DATABASE IF NOT EXISTS {db_name};")
        creates.append(f"USE {db_name};")
        creates.append(
            f"CREATE TABLE IF NOT EXISTS {table_name} (id INT PRIMARY KEY, data VARCHAR(100));"
        )
        databases.append(db_name)
        tables.append((db_name, table_name))
    return creates, databases, tables


def generate_inserts(tables, num_inserts_per_table):
    inserts = []
    id_tracker = {}
    for db_name, table in tables:
        ids = []
        for i in range(1, num_inserts_per_table + 1):
            value = f"value_{random.randint(1, 100)}"
            # Explicitly specifying the ID
            inserts.append(
                f"INSERT INTO {db_name}.{table} (id, data) VALUES ({i}, '{value}');"
            )
            ids.append(i)  # Track the IDs
        id_tracker[table] = ids
    return inserts, id_tracker


def generate_deletes(tables, id_tracker, num_deletes_per_table):
    deletes = []
    for db_name, table in tables:
        if table in id_tracker:
            ids = id_tracker[table]
            random.shuffle(ids)
            for i in range(min(num_deletes_per_table, len(ids))):
                deletes.append(f"DELETE FROM {db_name}.{table} WHERE id = {ids[i]};")
                ids.pop(i)  # Remove the ID from the list after deleting
    return deletes


def generate_updates(tables, id_tracker, num_updates_per_table):
    updates = []
    for db_name, table in tables:
        if table in id_tracker:
            ids = id_tracker[table]
            random.shuffle(ids)
            for i in range(min(num_updates_per_table, len(ids))):
                new_value = f"updated_{random.randint(1, 100)}"
                updates.append(
                    f"UPDATE {db_name}.{table} SET data = '{new_value}' WHERE id = {ids[i]};"
                )
    return updates


def generate_replaces(tables, num_replaces_per_table):
    replaces = []
    for db_name, table in tables:
        # REPLACE still uses random IDs since REPLACE can also act like INSERT
        for _ in range(num_replaces_per_table):
            id_to_replace = random.randint(1, 100)
            new_value = f"replaced_value_{random.randint(1, 100)}"
            replaces.append(
                f"REPLACE INTO {db_name}.{table} (id, data) VALUES ({id_to_replace}, '{new_value}');"
            )
    return replaces


def generate_alters(tables):
    alters = []
    for db_name, table in tables:
        new_column = f"new_column_{random.randint(1, 100)}"
        alters.append(
            f"ALTER TABLE {db_name}.{table} ADD COLUMN {new_column} VARCHAR(255) DEFAULT 'add column';"
        )
    return alters


def generate_truncates(tables):
    truncates = []
    for db_name, table in tables:
        truncates.append(f"TRUNCATE TABLE {db_name}.{table};")
        # Clear the IDs in tracker after truncating
    return truncates


def generate_rename_tables(tables):
    renames = []
    for db_name, table in tables:
        new_table_name = f"renamed_{table}"
        if random.choice([True, False]):
            renames.append(
                f"RENAME TABLE {db_name}.{table} TO {db_name}.{new_table_name};"
            )
        else:
            renames.append(
                f"ALTER TABLE {db_name}.{table} RENAME TO {db_name}.{new_table_name};"
            )
        # Update the table list with new name
        tables[tables.index((db_name, table))] = (db_name, new_table_name)
    return renames


def generate_drop_tables(tables):
    drop_tables = []
    for db_name, table in tables:
        drop_tables.append(f"DROP TABLE IF EXISTS {db_name}.{table};")
    return drop_tables


def generate_drop_databases(databases):
    drop_dbs = []
    for db in databases:
        drop_dbs.append(f"DROP DATABASE IF EXISTS {db};")
    return drop_dbs


def main():
    num_tables = 4
    num_inserts_per_table = 10
    num_deletes_per_table = 3
    num_updates_per_table = 5
    num_replaces_per_table = 5

    creates, databases, tables = generate_creates(num_tables)
    inserts, id_tracker = generate_inserts(tables, num_inserts_per_table)
    deletes = generate_deletes(tables, id_tracker, num_deletes_per_table)
    updates = generate_updates(tables, id_tracker, num_updates_per_table)
    replaces = generate_replaces(tables, num_replaces_per_table)
    alters = generate_alters(tables)
    truncates = generate_truncates(tables)
    renames = generate_rename_tables(tables)
    drop_tables = generate_drop_tables(tables)
    drop_dbs = generate_drop_databases(databases)

    statements = (
        drop_dbs
        + creates
        + inserts
        + updates
        + deletes
        + replaces
        + alters
        + truncates
        + renames
        + drop_tables
    )
    for statement in statements:
        print(statement)


if __name__ == "__main__":
    main()
