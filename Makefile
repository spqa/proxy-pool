setup-dev:
	GO111MODULE=off go get github.com/google/wire/cmd/wire
	GO111MODULE=off go get -u github.com/cosmtrek/air
	GO111MODULE=off go get -u github.com/swaggo/swag/cmd/swag

start-dev:
	docker-compose --project-name proxy-pool -f ./deployments/docker-compose-dev.yaml up -d
	air -c ./scripts/air.toml

build-air:
	go build -o ./tmp/main .