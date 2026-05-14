a:
	go build -o bin/golinq ./cmd/golinq
	./bin/golinq -dir ./tests/integration -res ./tests/integration -pkg integration
