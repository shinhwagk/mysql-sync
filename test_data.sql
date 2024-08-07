-- create user repl@'%' identified by 'repl';
-- grant replication client, replication slave on *.* to repl@'%';

drop database if exists test;
flush logs;
create database test;
flush logs;
create table test.tab1(a int primary key, b varchar(10));
flush logs;
insert into test.tab1 values(1,'a');
flush logs;
update test.tab1 set b='b' where a=1;
flush logs;
delete from test.tab1 where a=1;
flush logs;
insert into test.tab1 values(1,'a');
insert into test.tab1 values(2,'a');
insert into test.tab1 values(3,'a');
flush logs;
begin;
delete from test.tab1 where a=1;
update test.tab1 set b='b' where a=2;
replace into test.tab1 values(3,'c');
commit;
-- drop table test.tab1;
CREATE TABLE test.tab2 (col1 INT primary key, col2 INT, col3 INT, col4 INT, col5 INT, col6 INT, col7 INT, col8 INT, col9 INT, col10 INT, col11 INT, col12 INT, col13 INT, col14 INT, col15 INT,
col16 INT, col17 INT, col18 INT, col19 INT, col20 INT, col21 INT, col22 INT, col23 INT, col24 INT, col25 INT, col26 INT, col27 INT, col28 INT, col29 INT, col30 INT);
