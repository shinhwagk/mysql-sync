#!/bin/bash

set -e

dnf install -y https://dev.mysql.com/get/mysql80-community-release-el9-5.noarch.rpm
dnf install -y mysql-community-client

mysql -h -uroot -pexample 