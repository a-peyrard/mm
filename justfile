default: build

build:
    go build -o mm cmd/mm.go

clean:
    rm -f mm