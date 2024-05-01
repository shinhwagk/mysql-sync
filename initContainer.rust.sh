#!/bin/bash

set -e

dnf install -y git jq diffutils gcc

curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y

dnf install -y https://dev.mysql.com/get/mysql80-community-release-el9-5.noarch.rpm
dnf install -y mysql-community-client


dnf install -y python3 python3-pip