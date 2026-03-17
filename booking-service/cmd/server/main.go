package main

import (
	"log"
	"net/http"
	"os"

	"flight-booking/booking-service/internal/handler"
	"flight-booking/booking-service/internal/repository"
	"flight-booking/booking-service/internal/service"
	"flight-booking/booking-service/pkg/database"
	"flight-booking/booking-service/pkg/grpcclient"
)

func main() {
	db := database.Connect()
	flightClient := grpcclient.NewFlightClient()

	repo := repository.NewBookingRepo(db)
	bookingService := service.NewBookingService(repo, flightClient, db)
	h := handler.NewBookingHandler(bookingService)

	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /flights", h.SearchFlights)
	mux.HandleFunc("GET /flights/", h.GetFlight)
	mux.HandleFunc("POST /bookings", h.CreateBooking)
	mux.HandleFunc("GET /bookings/", h.GetBooking)
	mux.HandleFunc("POST /bookings/", h.CancelBooking)
	mux.HandleFunc("GET /bookings", h.ListBookings)

	log.Printf("Booking Service starting on port %s", httpPort)

	if err := http.ListenAndServe(":" + httpPort, mux); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}