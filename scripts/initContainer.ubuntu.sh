#!/bin/bash

set -ex

export DEBIAN_FRONTEND=noninteractive

apt update

# install packages
apt install -y curl wget lsb-release gnupg gcc # cmake make 

# install mysql clients
if [[ -z $(command -v mysqlbinlog) ]]; then
    wget https://dev.mysql.com/get/mysql-apt-config_0.8.30-1_all.deb && dpkg -i mysql-apt-config_0.8.30-1_all.deb && rm -f mysql-apt-config_0.8.30-1_all.deb
    apt install -y mysql-server-core-8.0 mysql-community-client-core
fi

# install rustup
[[ -z $(command -v cargo) ]] && curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
# curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y

# # install gh-ost
# curl -Lso /tmp/gh-ost.tar.gz https://github.com/github/gh-ost/releases/download/v1.1.6/gh-ost-binary-linux-amd64-20231207144046.tar.gz
# tar zxvf /tmp/gh-ost.tar.gz -C /usr/local/bin/

# install kafka sdk
rpm --import https://packages.confluent.io/rpm/7.6/archive.key
update-crypto-policies --set DEFAULT:SHA1
echo '[Confluent]
name=Confluent repository
baseurl=https://packages.confluent.io/rpm/7.6
gpgcheck=1
gpgkey=https://packages.confluent.io/rpm/7.6/archive.key
enabled=1

[Confluent-Clients]
name=Confluent Clients repository
baseurl=https://packages.confluent.io/clients/rpm/centos/$releasever/$basearch
gpgcheck=1
gpgkey=https://packages.confluent.io/clients/rpm/archive.key
enabled=1' >/etc/yum.repos.d/confluent.repo
dnf install -y librdkafka-devel