FROM --platform=linux/amd64 golang:1.23-bookworm@sha256:31dc846dd1bcca84d2fa231bcd16c09ff271bcc1a5ae2c48ff10f13b039688f3

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get -y update && \
    apt-get -y install apt-utils gcc-aarch64-linux-gnu libbtrfs-dev file &&  \
    curl -sSLO https://github.com/go-task/task/releases/download/v3.33.1/task_linux_amd64.deb && \
    apt-get -y install ./task_linux_amd64.deb && \
    rm -rf ./task_linux_amd64.deb /var/cache/apt/* /var/lib/apt/lists/* /var/log/*

ADD cmd /.werf-deps/cmd
ADD pkg /.werf-deps/pkg
ADD go.mod go.sum Taskfile.dist.yaml /.werf-deps/
ADD scripts /.werf-deps/scripts

RUN cd /.werf-deps && \
    task build:dist:all version=base && \
    task verify:binaries:dist:all version=base && \
    rm -rf /.werf-deps

RUN git config --global --add safe.directory /git
