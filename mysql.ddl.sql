-- DDLCreateSchema
CREATE SCHEMA my_database;

-- Switch to the new schema
USE my_database;

-- DDLCreateTable
CREATE TABLE my_table (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    age INT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- DDLDropSchema
-- Uncomment the following line to drop the schema and all its tables (not recommended to run immediately after creation)
-- DROP SCHEMA my_database;

-- DDLDropTable
-- Uncomment the following line to drop the table (not recommended to run immediately after creation)
-- DROP TABLE my_table;

-- DDLDropIndex
-- First, create an index to drop later
CREATE INDEX idx_name ON my_table(name);

-- Drop the index
DROP INDEX idx_name ON my_table;

-- DDLTruncate  
-- Truncate the table to remove all rows
TRUNCATE TABLE my_table;

-- DDLAlterTable
-- Add a column to the table
ALTER TABLE my_table
ADD COLUMN email VARCHAR(100) AFTER name;

-- DDLAlterTableAddColumn
-- Add another column to the table
ALTER TABLE my_table
ADD COLUMN address VARCHAR(255);

-- DDLAlterTableDropColumn
-- Drop the address column from the table
ALTER TABLE my_table
DROP COLUMN address;

-- DDLAlterTableModifyColumn
-- Modify the data type of the age column
ALTER TABLE my_table
MODIFY COLUMN age SMALLINT;

-- DDLAlterTableChangeColumn
-- Change the name of the column 'name' to 'full_name' and modify its data type
ALTER TABLE my_table
CHANGE COLUMN name full_name VARCHAR(150);

-- DDLAlterTableAlterColumn
-- Modify the created_at column to set a default value
ALTER TABLE my_table
ALTER COLUMN created_at SET DEFAULT NOW();
