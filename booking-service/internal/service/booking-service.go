package service

import (
	"context"
	"database/sql"
	"fmt"
	"crypto/rand"

	"flight-booking/booking-service/internal/model"
	"flight-booking/booking-service/internal/repository"
	"flight-booking/booking-service/pkg/grpcclient"
)

func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

type BookingService struct {
	repo *repository.BookingRepo
	flightClient *grpcclient.FlightClient
	db *sql.DB
}

func NewBookingService(repo *repository.BookingRepo, fc *grpcclient.FlightClient, db *sql.DB) *BookingService {
 	return &BookingService{repo: repo, flightClient: fc, db: db}
}

func (s *BookingService) CreateBooking(ctx context.Context, req *model.CreateBookingRequest) (*model.Booking, error) {
	flight, err := s.flightClient.GetFlight(ctx, req.FlightID)
	if err != nil {
		return nil, fmt.Errorf("Failed to get flight: %w", err)
	}

	if flight.AvailableSeats < req.SeatCount {
		return nil, fmt.Errorf("Not enough seats left")
	}

	totalPrice := float64(req.SeatCount) * flight.Price

	booking := &model.Booking{
		UserID: req.UserID,
		FlightID: req.FlightID,
		FlightNumber: flight.FlightNumber,
		OriginAirport: flight.OriginAirport,
		DestinationAirport: flight.DestinationAirport,
		DepartureTime: flight.DepartureTime.AsTime(),
		PassengerName: req.PassengerName,
		PassengerEmail: req.PassengerEmail,
		PassengerPhone: req.PassengerPhone,
		SeatCount: req.SeatCount,
		TotalPrice: totalPrice,
		Status: "CONFIRMED",
	}

	// temporarily
	booking.UserID = generateUUID()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	if err = s.repo.Create(ctx, tx, booking); err != nil {
		return nil, fmt.Errorf("Failed to save booking: %w", err)
	}

	_, err = s.flightClient.ReserveSeats(ctx, req.FlightID, req.SeatCount, booking.ID)
	if err != nil {
		return nil, fmt.Errorf("Failed to reserve seats: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("Failed to commit: %w", err)
	}

	return booking, nil
}

func (s *BookingService) CancelBooking(ctx context.Context, id string) error {
	booking, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if booking == nil {
		return fmt.Errorf("Booking not found")
	}
	if booking.Status != "CONFIRMED" {
		return fmt.Errorf("Only CONFIRMED bookings can be cancelled")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("Failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	if err = s.flightClient.ReleaseReservation(ctx, id); err != nil {
		return fmt.Errorf("Failed to release seats: %w", err)
	}

	if err = s.repo.UpdateStatus(ctx, tx, id, "CANCELLED"); err != nil {
		return fmt.Errorf("Failed to update booking: %w", err)
	}

	return tx.Commit()
}

func (s *BookingService) GetByID(ctx context.Context, id string) (*model.Booking, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *BookingService) GetByUserID(ctx context.Context, userID string) ([]model.Booking, error) {
	return s.repo.GetByUserID(ctx, userID)
}
