#!/bin/bash
set -e

go test -covermode=atomic -coverprofile=coverage.out .

go tool cover -html=coverage.out -o coverage.html

go tool cover -func=coverage.out | grep "total:"

echo "Coverage report saved to coverage.html"