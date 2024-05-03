#!/bin/bash

set -e

# install packages
dnf install -y git jq diffutils gcc procps-ng iproute iputils telnet dnf-utils epel-release

# install rustup
if ! command -v cargo &> /dev/null; then 
  curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
fi

# install mysql clients
dnf install -y https://dev.mysql.com/get/mysql80-community-release-el9-5.noarch.rpm
dnf install -y mysql-community-client-8.0.36

# install sysbench
dnf install -y epel-release
dnf -y install sysbench

# install python & packages
dnf install -y python3.11 python3.11-pip
python3.11 -m pip install -r requirements.txt

# install gh-ost
curl -Lso /tmp/gh-ost.tar.gz https://github.com/github/gh-ost/releases/download/v1.1.6/gh-ost-binary-linux-amd64-20231207144046.tar.gz
tar zxvf /tmp/gh-ost.tar.gz -C /usr/local/bin/