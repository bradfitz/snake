run:
	GOOS=js GOARCH=wasm go build -o snake.wasm .
	go run serve.go
