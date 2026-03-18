package main

import (
	"log"
	"net"
	"os"

	"google.golang.org/grpc"

	"flight-booking/flight-service/internal/handler"
	"flight-booking/flight-service/internal/repository"
	"flight-booking/flight-service/internal/middleware"
	"flight-booking/flight-service/pkg/database"
	"flight-booking/flight-service/pkg/cache"
	flightpb "flight-booking/.gen/.proto/flight"
)

func main() {
	db := database.Connect()
	cacheClient := cache.NewCache()
	defer cacheClient.Close()

	repo := repository.NewFlightRepo(db, cacheClient)

	h := handler.NewFlightHandler(repo, db, cacheClient)

	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "9090"
	}

	lis, err := net.Listen("tcp", ":" + grpcPort)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.AuthInterceptor()),
	)
	flightpb.RegisterFlightServiceServer(grpcServer, h)
	

	log.Printf("Flight Service starting on port %s", grpcPort)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
