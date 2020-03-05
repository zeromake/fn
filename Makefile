test:
	go test -coverprofile=coverage .

cover: test
	go tool cover -html=coverage
