#!/bin/bash

set -e

dnf install -y git jq diffutils gcc procps-ng

dnf install -y python3.11 python3.11-pip

curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -v -y

dnf install -y https://dev.mysql.com/get/mysql80-community-release-el9-5.noarch.rpm
dnf install -y mysql-community-client

