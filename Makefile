.PHONY: proto-install proto generate up down test insert-test-data cache-logs

proto-install:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	export PATH="$PATH:$(go env GOPATH)/bin"

proto:
	protoc --go_out=.gen --go-grpc_out=.gen \
		--go_opt=paths=source_relative --go-grpc_opt=paths=source_relative \
		.proto/flight/flight.proto

generate:
	proto
	go generate ./...

up:
	docker-compose up --build -d

down:
	docker-compose down -v

test:
	go test ./... -race -cover

insert-test-data:
	docker-compose exec postgres-flight psql -U flight -d flight -c " \
	INSERT INTO flights ( \
		flight_number, departure_date, airline, \
		origin_airport, destination_airport, \
		departure_time, arrival_time, \
		total_seats, available_seats, price, status \
	) VALUES ( \
		'TEST001', '2026-04-01', 'TestAir',\
		'SVO', 'LED', \
		'2026-04-01 10:00:00', '2026-04-01 12:00:00', \
		100, 50, 5000.00, 'SCHEDULED' \
	) RETURNING id;"

cache-logs:
	docker-compose logs flight-service | grep -E "CACHE HIT|CACHE MISS"
