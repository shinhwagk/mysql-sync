#!/bin/bash

set -ex

# install packages
dnf install -y gcc diffutils iproute iputils telnet dnf-utils jq procps-ng git epel-release sysstat curl which

# install rustup
[[ -z $(command -v cargo) ]] && curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
# curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y

# install mysql clients
dnf install -y https://dev.mysql.com/get/mysql80-community-release-el9-5.noarch.rpm
dnf install -y mysql-community-client-8.0.36

# install sysbench
dnf install -y epel-release
dnf -y install sysbench

# install python & packages
dnf install -y python3.11 python3.11-pip
python3.11 -m pip install -r requirements.txt

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