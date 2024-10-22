RECORDER_BINARY = dispatchr-recorder
WORKER_BINARY = dispatchr-worker

build_recorder:
	go build -race -o bin/$(RECORDER_BINARY) ./cmd/recorder
.PHONY: build_recorder

build_worker:
	go build -race -o bin/$(WORKER_BINARY) ./cmd/worker
.PHONY: build_worker

build: build_recorder build_worker

clean:
	rm -rf build
.PHONY: clean
