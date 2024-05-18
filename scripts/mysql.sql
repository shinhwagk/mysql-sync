drop database if exists test;
create database if not exists test;
use test;

CREATE TABLE all_types (
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
    timestamp_col TIMESTAMP,
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

INSERT INTO all_types (
    tinyint_col, smallint_col, mediumint_col, int_col, bigint_col,
    float_col, double_col, decimal_col, bit_col,
    date_col, datetime_col, timestamp_col, time_col, year_col,
    char_col, varchar_col, tinyblob_col, tinytext_col, blob_col, text_col,
    mediumblob_col, mediumtext_col, longblob_col, longtext_col,
    enum_col, set_col
) VALUES (
    127, 32767, 8388607, 2147483647, 9223372036854775807,
    12345.12, 1234567890.1234, 123456.78, b'10101010',
    '2024-05-16', '2024-05-16 12:34:56', CURRENT_TIMESTAMP, '12:34:56', 2024,
    'Fixed length char', 'Variable length varchar', _binary '\x01\x02\x03', 'tiny text', _binary '\x01\x02\x03\x04', 'Example text',
    _binary '\x01\x02\x03\x04\x05', 'Medium text example', _binary '\x01\x02\x03\x04\x05\x06', 'Long text example',
    'value2', 'option1,option3'
);

-- alter table test.data_types_demo add index idx_int_col (int_col);
