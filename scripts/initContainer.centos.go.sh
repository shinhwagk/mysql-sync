#!/bin/bash
dnf install -y diffutils iproute procps unzip iputils yum-utils

# install mysql client
dnf install -y https://dev.mysql.com/get/mysql80-community-release-el9-5.noarch.rpm
dnf install -y mysql-community-client-8.0.36

# install sysbench
dnf install -y epel-release
dnf -y install sysbench

# install python

# install terraform
# yum-config-manager --add-repo https://rpm.releases.hashicorp.com/RHEL/hashicorp.repo
# yum -y install terraform-1.9.2