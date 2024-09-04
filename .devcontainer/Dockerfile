FROM quay.io/centos/centos:stream9

RUN dnf install -y git

ENV PATH /usr/local/go/bin:$PATH

RUN curl -OL https://go.dev/dl/go1.22.4.linux-amd64.tar.gz && rm -rf /usr/local/go && tar -C /usr/local -xzf go1.22.4.linux-amd64.tar.gz