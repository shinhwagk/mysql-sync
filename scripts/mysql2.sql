DROP DATABASE IF EXISTS test;
CREATE DATABASE IF NOT EXISTS test;
USE test;

CREATE TABLE unified_table (
    id INT AUTO_INCREMENT PRIMARY KEY,
    tinyint_col TINYINT,
    smallint_col SMALLINT,
    mediumint_col MEDIUMINT,
    int_col INT,
    bigint_col BIGINT,
    float_col FLOAT,
    double_col DOUBLE,
    decimal_col DECIMAL(10,2),
    bit_col BIT(8),
    date_col DATE,
    datetime_col DATETIME,
    timestamp_col TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    time_col TIME,
    year_col YEAR,
    char_col CHAR(255),
    varchar_col VARCHAR(200),
    tinyblob_col TINYBLOB,
    tinytext_col TINYTEXT,
    blob_col BLOB,
    text_col TEXT,
    mediumblob_col MEDIUMBLOB,
    mediumtext_col MEDIUMTEXT,
    longblob_col LONGBLOB,
    longtext_col LONGTEXT,
    enum_col ENUM('value1', 'value2', 'value3'),
    set_col SET('option1', 'option2', 'option3')
);


INSERT INTO unified_table (
    tinyint_col, smallint_col, mediumint_col, int_col, bigint_col,
    float_col, double_col, decimal_col, bit_col, date_col,
    datetime_col, timestamp_col, time_col, year_col, char_col,
    varchar_col, tinyblob_col, tinytext_col, blob_col, text_col,
    mediumblob_col, mediumtext_col, longblob_col, longtext_col,
    enum_col, set_col
) VALUES (
    127, 32767, 8388607, 2147483647, 9223372036854775807,
    123.456, 12345678.901234, 99999.99, b'11111111', '2024-05-20',
    '2024-05-20 15:00:00', CURRENT_TIMESTAMP, '12:34:56', 2024, 'A character string',
    'A variable character string', 'blobdata', 'tinytext data', 'blob data', 'Text field with lots of characters',
    'mediumblob data', 'mediumtext data', 'largeblob data', 'Very long text data',
    'value1', 'option1,option2'
);
