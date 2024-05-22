statements = [
    "CREATE TABLE xxx.`myschema`.`mytable` ",
    "create table myschema.mytable ",
    "CREATE TABLE `schema`.table ",
    "create table `table` ",
    "create table table ",
    "CREATE TABLE myschema.`table` ",
    "create \n table `schema`.`table` ",
    "create \n table `table` ",
]


import main1

for st in statements:
    s, t = main1.tiquddl_schema_tab(st)
    print(s, t)
