create user repl@'%' identified by 'repl';
grant replication client, replication slave on *.* to repl@'%';