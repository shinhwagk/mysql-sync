DROP DATABASE IF EXISTS test;
CREATE DATABASE IF NOT EXISTS test;
USE test;

CREATE TABLE tinyint_table (id INT AUTO_INCREMENT PRIMARY KEY, tinyint_col TINYINT);
CREATE TABLE smallint_table (id INT AUTO_INCREMENT PRIMARY KEY, smallint_col SMALLINT);
CREATE TABLE mediumint_table (id INT AUTO_INCREMENT PRIMARY KEY, mediumint_col MEDIUMINT);
CREATE TABLE int_table (id INT AUTO_INCREMENT PRIMARY KEY, int_col INT);
CREATE TABLE bigint_table (id INT AUTO_INCREMENT PRIMARY KEY, bigint_col BIGINT);
CREATE TABLE float_table (id INT AUTO_INCREMENT PRIMARY KEY, float_col FLOAT);
CREATE TABLE double_table (id INT AUTO_INCREMENT PRIMARY KEY, double_col DOUBLE);
CREATE TABLE decimal_table (id INT AUTO_INCREMENT PRIMARY KEY, decimal_col DECIMAL(10,2));
CREATE TABLE bit_table (id INT AUTO_INCREMENT PRIMARY KEY, bit_col BIT(8));
CREATE TABLE date_table (id INT AUTO_INCREMENT PRIMARY KEY, date_col DATE);
CREATE TABLE datetime_table (id INT AUTO_INCREMENT PRIMARY KEY, datetime_col DATETIME);
CREATE TABLE timestamp_table (id INT AUTO_INCREMENT PRIMARY KEY, timestamp_col TIMESTAMP);
CREATE TABLE time_table (id INT AUTO_INCREMENT PRIMARY KEY, time_col TIME);
CREATE TABLE year_table (id INT AUTO_INCREMENT PRIMARY KEY, year_col YEAR);
CREATE TABLE char_table (id INT AUTO_INCREMENT PRIMARY KEY, char_col CHAR(255));
CREATE TABLE varchar_table (id INT AUTO_INCREMENT PRIMARY KEY, varchar_col VARCHAR(200));
CREATE TABLE tinyblob_table (id INT AUTO_INCREMENT PRIMARY KEY, tinyblob_col TINYBLOB);
CREATE TABLE tinytext_table (id INT AUTO_INCREMENT PRIMARY KEY, tinytext_col TINYTEXT);
CREATE TABLE blob_table (id INT AUTO_INCREMENT PRIMARY KEY, blob_col BLOB);
CREATE TABLE text_table (id INT AUTO_INCREMENT PRIMARY KEY, text_col TEXT);
CREATE TABLE mediumblob_table (id INT AUTO_INCREMENT PRIMARY KEY, mediumblob_col MEDIUMBLOB);
CREATE TABLE mediumtext_table (id INT AUTO_INCREMENT PRIMARY KEY, mediumtext_col MEDIUMTEXT);
CREATE TABLE longblob_table (id INT AUTO_INCREMENT PRIMARY KEY, longblob_col LONGBLOB);
CREATE TABLE longtext_table (id INT AUTO_INCREMENT PRIMARY KEY, longtext_col LONGTEXT);
CREATE TABLE enum_table (id INT AUTO_INCREMENT PRIMARY KEY, enum_col ENUM('value1', 'value2', 'value3'));
CREATE TABLE set_table (id INT AUTO_INCREMENT PRIMARY KEY, set_col SET('option1', 'option2', 'option3'));

-- tinyint_table
INSERT INTO tinyint_table (tinyint_col) VALUES (1);
UPDATE tinyint_table SET tinyint_col = 2 WHERE id = 1;
DELETE FROM tinyint_table WHERE id = 1;

-- smallint_table
INSERT INTO smallint_table (smallint_col) VALUES (100);
UPDATE smallint_table SET smallint_col = 200 WHERE id = 1;
DELETE FROM smallint_table WHERE id = 1;

-- mediumint_table
INSERT INTO mediumint_table (mediumint_col) VALUES (10000);
UPDATE mediumint_table SET mediumint_col = 20000 WHERE id = 1;
DELETE FROM mediumint_table WHERE id = 1;

-- int_table
INSERT INTO int_table (int_col) VALUES (123456);
UPDATE int_table SET int_col = 654321 WHERE id = 1;
DELETE FROM int_table WHERE id = 1;

-- bigint_table
INSERT INTO bigint_table (bigint_col) VALUES (1000000000);
UPDATE bigint_table SET bigint_col = 2000000000 WHERE id = 1;
DELETE FROM bigint_table WHERE id = 1;

-- float_table
INSERT INTO float_table (float_col) VALUES (123.456);
UPDATE float_table SET float_col = 654.321 WHERE id = 1;
DELETE FROM float_table WHERE id = 1;

-- double_table
INSERT INTO double_table (double_col) VALUES (123456.789);
UPDATE double_table SET double_col = 987654.321 WHERE id = 1;
DELETE FROM double_table WHERE id = 1;

-- decimal_table
INSERT INTO decimal_table (decimal_col) VALUES (12345.67);
UPDATE decimal_table SET decimal_col = 76543.21 WHERE id = 1;
DELETE FROM decimal_table WHERE id = 1;

-- bit_table
INSERT INTO bit_table (bit_col) VALUES (b'10101010');
UPDATE bit_table SET bit_col = b'01010101' WHERE id = 1;
DELETE FROM bit_table WHERE id = 1;

-- date_table
INSERT INTO date_table (date_col) VALUES ('2023-01-01');
UPDATE date_table SET date_col = '2024-01-01' WHERE id = 1;
DELETE FROM date_table WHERE id = 1;

-- datetime_table
INSERT INTO datetime_table (datetime_col) VALUES ('2023-01-01 12:00:00');
UPDATE datetime_table SET datetime_col = '2024-01-01 12:00:00' WHERE id = 1;
DELETE FROM datetime_table WHERE id = 1;

-- timestamp_table
INSERT INTO timestamp_table (timestamp_col) VALUES (CURRENT_TIMESTAMP);
UPDATE timestamp_table SET timestamp_col = CURRENT_TIMESTAMP WHERE id = 1;
DELETE FROM timestamp_table WHERE id = 1;

-- time_table
INSERT INTO time_table (time_col) VALUES ('12:00:00');
UPDATE time_table SET time_col = '13:00:00' WHERE id = 1;
DELETE FROM time_table WHERE id = 1;

-- year_table
INSERT INTO year_table (year_col) VALUES (2023);
UPDATE year_table SET year_col = 2024 WHERE id = 1;
DELETE FROM year_table WHERE id = 1;

-- char_table
INSERT INTO char_table (char_col) VALUES ('example text');
UPDATE char_table SET char_col = 'new text' WHERE id = 1;
DELETE FROM char_table WHERE id = 1;

-- varchar_table
INSERT INTO varchar_table (varchar_col) VALUES ('example text');
UPDATE varchar_table SET varchar_col = 'new text' WHERE id = 1;
DELETE FROM varchar_table WHERE id = 1;

-- tinyblob_table
INSERT INTO tinyblob_table (tinyblob_col) VALUES ('example blob');
UPDATE tinyblob_table SET tinyblob_col = 'new blob' WHERE id = 1;
DELETE FROM tinyblob_table WHERE id = 1;

-- tinytext_table
INSERT INTO tinytext_table (tinytext_col) VALUES ('example text');
UPDATE tinytext_table SET tinytext_col = 'new text' WHERE id = 1;
DELETE FROM tinytext_table WHERE id = 1;

-- blob_table
INSERT INTO blob_table (blob_col) VALUES ('example blob');
UPDATE blob_table SET blob_col = 'new blob' WHERE id = 1;
DELETE FROM blob_table WHERE id = 1;

-- text_table
INSERT INTO text_table (text_col) VALUES ('example text');
UPDATE text_table SET text_col = 'new text' WHERE id = 1;
DELETE FROM text_table WHERE id = 1;

-- mediumblob_table
INSERT INTO mediumblob_table (mediumblob_col) VALUES ('example blob');
UPDATE mediumblob_table SET mediumblob_col = 'new blob' WHERE id = 1;
DELETE FROM mediumblob_table WHERE id = 1;

-- mediumtext_table
INSERT INTO mediumtext_table (mediumtext_col) VALUES ('example text');
UPDATE mediumtext_table SET mediumtext_col = 'new text' WHERE id = 1;
DELETE FROM mediumtext_table WHERE id = 1;

-- longblob_table
INSERT INTO longblob_table (longblob_col) VALUES ('example blob');
UPDATE longblob_table SET longblob_col = 'new blob' WHERE id = 1;
DELETE FROM longblob_table WHERE id = 1;

-- longtext_table
INSERT INTO longtext_table (longtext_col) VALUES ('example text');
UPDATE longtext_table SET longtext_col = 'new text' WHERE id = 1;
DELETE FROM longtext_table WHERE id = 1;

-- enum_table
INSERT INTO enum_table (enum_col) VALUES ('value1');
UPDATE enum_table SET enum_col = 'value2' WHERE id = 1;
DELETE FROM enum_table WHERE id = 1;

-- set_table
INSERT INTO set_table (set_col) VALUES ('option1');
UPDATE set_table SET set_col = 'option1,option2' WHERE id = 1;
DELETE FROM set_table WHERE id = 1;
