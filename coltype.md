```sh

id INT AUTO_INCREMENT PRIMARY KEY
tinyint_col TINYINT
smallint_col SMALLINT
mediumint_col MEDIUMINT
int_col INT
bigint_col BIGINT
float_col FLOAT
double_col DOUBLE
decimal_col DECIMAL(10,2)
bit_col BIT(8)
date_col DATE
datetime_col DATETIME
timestamp_col TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
time_col TIME
year_col YEAR
char_col CHAR(255)
varchar_col VARCHAR(200)
tinyblob_col TINYBLOB
tinytext_col TINYTEXT
blob_col BLOB
text_col TEXT
mediumblob_col MEDIUMBLOB
mediumtext_col MEDIUMTEXT
longblob_col LONGBLOB
longtext_col LONGTEXT
enum_col ENUM('value1', 'value2', 'value3')
set_col SET('option1', 'option2', 'option3'
```


2 test unified_table 1
```txt
tinyint 1 int8
smallint 2 int16
mediumint 9 int32
int 3 int32
bigint 8 int64
float 4 float32
double 5 float64
decimal 246 string
bit 16 int64
date 10 string
datetime 18 string
timestamp 17 string
time 19 string
year 13 int
char 254 string
varchar 15 string
tinyblob 252 []uint8
tinytext 252 []uint8
blob 252 []uint8
text 252 []uint8
mediumblob 252 []uint8
mediumtext 252 []uint8
longblob 252 []uint8
longtext 252 []uint8
enum 254 int64

```


tinyblob tinytext blob text mediumblob mediumtext longblob longtext