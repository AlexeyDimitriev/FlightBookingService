package handler

import (
	"context"
	"database/sql"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	flightpb "flight-booking/.gen/.proto/flight"
	"flight-booking/flight-service/internal/model"
	"flight-booking/flight-service/internal/repository"
)

type FlightHandler struct {
	flightpb.UnimplementedFlightServiceServer
	repo *repository.FlightRepo
	db   *sql.DB
}

func NewFlightHandler(repo *repository.FlightRepo, db *sql.DB) *FlightHandler {
	return &FlightHandler{
		repo: repo,
		db: db,
	}
}

func (h *FlightHandler) SearchFlights(ctx context.Context, req *flightpb.SearchFlightsRequest) (*flightpb.SearchFlightsResponse, error) {
	var date *time.Time = nil
	if req.Date != nil {
		t := req.Date.AsTime()
		date = &t
	}

	flights, err := h.repo.Search(ctx, req.Origin, req.Destination, date)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var pbFlights []*flightpb.Flight
	for _, f := range flights {
		pbFlights = append(pbFlights, toPBFlight(&f))
	}
	return &flightpb.SearchFlightsResponse{Flights: pbFlights}, nil
}

func (h *FlightHandler) GetFlight(ctx context.Context, req *flightpb.GetFlightRequest) (*flightpb.GetFlightResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "Flight ID is required")
	}

	f, err := h.repo.GetByID(ctx, req.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	} else if f == nil {
		return nil, status.Error(codes.NotFound, "Flight not found")
	}
	return &flightpb.GetFlightResponse{Flight: toPBFlight(f)}, nil
}

func (h *FlightHandler) ReserveSeats(ctx context.Context, req *flightpb.ReserveSeatsRequest) (*flightpb.ReserveSeatsResponse, error) {
	if req.SeatCount <= 0 {
		return nil, status.Error(codes.InvalidArgument, "seat_count must be positive")
	}
	if req.BookingId == "" {
		return nil, status.Error(codes.InvalidArgument, "booking_id is required")
	}

	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Error(codes.Internal, "Failed to start transaction")
	}
	defer tx.Rollback()

	reservationID, err := h.repo.ReserveSeats(ctx, tx, req.FlightId, req.SeatCount, req.BookingId)
	if err != nil {
		if err.Error() == "Flight not found" {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		if err.Error() == "Not enough seats" {
			return nil, status.Error(codes.ResourceExhausted, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	if err = tx.Commit(); err != nil {
		return nil, status.Error(codes.Internal, "Failed to commit")
	}

	return &flightpb.ReserveSeatsResponse{ReservationId: reservationID}, nil
}

func (h *FlightHandler) ReleaseReservation(ctx context.Context, req *flightpb.ReleaseReservationRequest) (*flightpb.ReleaseReservationResponse, error) {
	if req.BookingId == "" {
		return nil, status.Error(codes.InvalidArgument, "booking_id is required")
	}

	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Error(codes.Internal, "Failed to start transaction")
	}
	defer tx.Rollback()

	released, err := h.repo.ReleaseReservation(ctx, tx, req.BookingId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if err = tx.Commit(); err != nil {
		return nil, status.Error(codes.Internal, "Failed to commit")
	}

	return &flightpb.ReleaseReservationResponse{Released: released}, nil
}

func toPBFlight(f *model.Flight) *flightpb.Flight {
	return &flightpb.Flight{
		Id: f.ID,
		FlightNumber: f.FlightNumber,
		DepartureDate: timestamppb.New(f.DepartureDate),
		Airline: f.Airline,
		OriginAirport: f.OriginAirport,
		DestinationAirport: f.DestinationAirport,
		DepartureTime: timestamppb.New(f.DepartureTime),
		ArrivalTime: timestamppb.New(f.ArrivalTime),
		TotalSeats: f.TotalSeats,
		AvailableSeats: f.AvailableSeats,
		Price: f.Price,
		Status: flightpb.FlightStatus(flightpb.FlightStatus_value[f.Status]),
	}
}