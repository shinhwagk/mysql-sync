#!/bin/bash
dnf install -y diffutils iproute2 procps unzip

curl -OL https://go.dev/dl/go1.22.3.linux-amd64.tar.gz
rm -rf /usr/local/go && tar -C /usr/local -xzf go1.22.3.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# # install python stacks
# dnf install -y python3.12 python3.12-pip
# python3.12 -m pip install -r requirements.txt

# install mysql client
dnf install -y https://dev.mysql.com/get/mysql80-community-release-el9-5.noarch.rpm
dnf install -y mysql-community-client-8.0.36

# install sysbench
dnf install -y epel-release
dnf -y install sysbench

# install deno
curl -fsSL https://deno.land/install.sh | sh