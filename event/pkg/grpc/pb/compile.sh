#!/usr/bin/env sh

protoc event.proto --go_out=plugins=grpc:.
