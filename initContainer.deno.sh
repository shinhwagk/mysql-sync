#!/usr/bin/env bash

set -x

dnf install -y unzip diffutils

test -z "$(command -v deno)" && curl -fsSL https://deno.land/install.sh | sh -s v1.42.4

code --install-extension denoland.vscode-deno-3.37.0.vsix 

dnf install -y https://dev.mysql.com/get/mysql80-community-release-el9-5.noarch.rpm
dnf install -y mysql-community-client

