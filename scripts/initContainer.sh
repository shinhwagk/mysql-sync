#!/bin/bash

set -e

dnf install -y git jq diffutils gcc procps-ng zsh iproute iputils telnet

sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)"

dnf install -y python3.11 python3.11-pip

curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y

dnf install -y https://dev.mysql.com/get/mysql80-community-release-el9-5.noarch.rpm
dnf install -y mysql-community-client-8.0.36
