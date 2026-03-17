package model

import (
	"time"
)

type Flight struct {
	ID string
	FlightNumber string
	DepartureDate time.Time
	Airline string
	OriginAirport string
	DestinationAirport string
	DepartureTime time.Time
	ArrivalTime time.Time
	TotalSeats int32
	AvailableSeats int32
	Price float64
	Status string
}
