#!/usr/bin/env bash

  ./.tools/gophermarttest \
            -test.v -test.run=^TestGophermart$ \
            -gophermart-binary-path=cmd/gophermart/gophermart \
            -gophermart-host=localhost \
            -gophermart-port=8080 \
            -gophermart-database-uri="postgres://gopher:supersecretpass@postgres:5432/gophermart?sslmode=disable" \
            -accrual-binary-path=cmd/accrual/accrual_darwin_arm64 \
            -accrual-host=localhost \
            -accrual-port=8081 \
            -accrual-database-uri="postgres://gopher:supersecretpass@postgres:5432/gophermart?sslmode=disable"