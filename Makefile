setup-dev:
	GO111MODULE=off go get github.com/google/wire/cmd/wire
	GO111MODULE=off go get -u github.com/cosmtrek/air
	GO111MODULE=off go get -u github.com/swaggo/swag/cmd/swa