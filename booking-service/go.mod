module flight-booking/booking-service

go 1.24.0

require flight-booking/.gen v0.0.0

require (
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0 // indirect
	github.com/lib/pq v1.11.2 // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
	google.golang.org/grpc v1.79.2 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
)

replace flight-booking/.gen => ../.gen
