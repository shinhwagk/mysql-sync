reset master;
drop database if exists test;
create database test;
CREATE TABLE test.data_types_demo (
  `id` INT AUTO_INCREMENT PRIMARY KEY,
  `varchar_col` VARCHAR(255),     -- 可变长度字符串
  `char_col` CHAR(10),            -- 固定长度字符串
  `int_col` INT,                  -- 整数
  `smallint_col` SMALLINT,        -- 小整数
  `tinyint_col` TINYINT,          -- 很小的整数
  `bigint_col` BIGINT,            -- 大整数
  `float_col` FLOAT,              -- 浮点数
  `double_col` DOUBLE,            -- 双精度浮点数
  `decimal_col` DECIMAL(10, 2),   -- 十进制数，适用于财务计算
  `date_col` DATE,                -- 日期
  `time_col` TIME,                -- 时间
  `datetime_col` DATETIME,        -- 日期和时间
  `timestamp_col` TIMESTAMP,      -- 时间戳
  `year_col` YEAR,                -- 年份
  `blob_col` BLOB,                -- 二进制大对象
  `text_col` TEXT,                -- 文本
  `enum_col` ENUM('val1', 'val2', 'val3'),  -- 枚举类型
  `set_col` SET('set1', 'set2', 'set3')     -- 集合类型
);

INSERT INTO test.data_types_demo (
  varchar_col, 
  char_col, 
  int_col, 
  smallint_col, 
  tinyint_col, 
  bigint_col, 
  float_col, 
  double_col, 
  decimal_col, 
  date_col, 
  time_col, 
  datetime_col, 
  timestamp_col, 
  year_col, 
  blob_col, 
  text_col, 
  enum_col, 
  set_col
) VALUES (
  'Example text',          -- VARCHAR(255)
  'ABCDE',                 -- CHAR(10)
  12345,                   -- INT
  32767,                   -- SMALLINT
  127,                     -- TINYINT
  9223372036854775807,     -- BIGINT
  12345.678,               -- FLOAT
  12345678.91011,          -- DOUBLE
  12345.67,                -- DECIMAL(10, 2)
  '2024-05-10',            -- DATE
  '15:30:00',              -- TIME
  '2024-05-10 15:30:00',   -- DATETIME
  CURRENT_TIMESTAMP,       -- TIMESTAMP
  2024,                    -- YEAR
  'binary data here',      -- BLOB (简单文本表示二进制数据，实际用法可能需要函数处理)
  'Longer piece of text',  -- TEXT
  'val1',                  -- ENUM
  'set1,set2'              -- SET
);

update test.data_types_demo set int_col=2222 where id=1;
delete from test.data_types_demo  where id=1;


alter table test.data_types_demo add index idx_int_col (int_col);

INSERT INTO test.data_types_demo (
  varchar_col, 
  char_col, 
  int_col, 
  smallint_col, 
  tinyint_col, 
  bigint_col, 
  float_col, 
  double_col, 
  decimal_col, 
  date_col, 
  time_col, 
  datetime_col, 
  timestamp_col, 
  year_col, 
  blob_col, 
  text_col, 
  enum_col, 
  set_col
) VALUES (
  'Example text',          -- VARCHAR(255)
  'ABCDE',                 -- CHAR(10)
  12345,                   -- INT
  32767,                   -- SMALLINT
  127,                     -- TINYINT
  9223372036854775807,     -- BIGINT
  12345.678,               -- FLOAT
  12345678.91011,          -- DOUBLE
  12345.67,                -- DECIMAL(10, 2)
  '2024-05-10',            -- DATE
  '15:30:00',              -- TIME
  '2024-05-10 15:30:00',   -- DATETIME
  CURRENT_TIMESTAMP,       -- TIMESTAMP
  2024,                    -- YEAR
  'binary data here',      -- BLOB (简单文本表示二进制数据，实际用法可能需要函数处理)
  'Longer piece of text',  -- TEXT
  'val1',                  -- ENUM
  'set1,set2'              -- SET
);

update test.data_types_demo set int_col=2222 where id=1;
delete from test.data_types_demo  where id=1;