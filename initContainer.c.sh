#!/usr/bin/env bash

set -x

dnf install -y gcc make gdb

dnf install -y https://dev.mysql.com/get/mysql80-community-release-el9-5.noarch.rpm
dnf install -y mysql-community-client
