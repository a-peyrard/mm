default: build

configure-dev:
  @echo "⬇️  installing tools (gotestsum, lefthook, ...)"
  @go install gotest.tools/gotestsum@latest
  @go install github.com/evilmartians/lefthook@latest
  @echo "🔧 configuring pre-commit hooks"
  @lefthook install
  @echo "👌 done, happy hacking!"

build:
    go build -o mm cmd/mm.go

clean:
    rm -f mm

test-go *ARGS:
    @gotestsum -- -v -race "$@" ./...

test-python *ARGS:
    @cd internal/embedding/python && uv run python -m pytest . -v "$@"

test-python-fast *ARGS:
    @cd internal/embedding/python && uv run python -m pytest . -v -m "not slow" "$@"

test: test-go test-python
