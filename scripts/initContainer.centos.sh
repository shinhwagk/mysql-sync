#!/bin/bash
dnf install -y zsh git diffutils

# install python stacks
dnf install -y python3.12 python3.12-pip
python3.12 -m pip install -r requirements.txt

# install mysql client
dnf install -y https://dev.mysql.com/get/mysql80-community-release-el9-5.noarch.rpm
dnf install -y mysql-community-client-8.0.36

# install sysbench
dnf install -y epel-release
dnf -y install sysbench