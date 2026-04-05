# makefile

.PHONY: build
build:
	go build .

.PHONY: run
run:
	go run . -c 3 -p 8080 -t "$(POISON)" data/*.txt

.PHONY: test
test:
	go test

.PHONY: bench
bench:
	go test -bench=. -benchmem

.PHONY: benchprof
benchprof:
	go test -bench=. -benchmem -cpuprofile main_cpu.prof -memprofile main_mem.prof

.PHONY: profweb
profweb:
	go tool pprof -http :9000 $(PROF)
