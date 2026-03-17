package repository

import (
	"context"
	"database/sql"

	"flight-booking/booking-service/internal/model"
)

type BookingRepo struct {
 	db *sql.DB
}

func NewBookingRepo(db *sql.DB) *BookingRepo {
 	return &BookingRepo{db: db}
}

func (r *BookingRepo) Create(ctx context.Context, tx *sql.Tx, booking *model.Booking) error {
	err := tx.QueryRowContext(
		ctx,
		`INSERT INTO bookings (user_id, flight_id, flight_number, origin_airport,
		destination_airport, departure_time, passenger_name, passenger_email, passenger_phone, 
		seat_count, total_price, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) 
		RETURNING id`,
		booking.UserID, booking.FlightID, booking.FlightNumber,
		booking.OriginAirport, booking.DestinationAirport, booking.DepartureTime,
		booking.PassengerName, booking.PassengerEmail, booking.PassengerPhone,
		booking.SeatCount, booking.TotalPrice, booking.Status,
	).Scan(&booking.ID)
	return err
}

func (r *BookingRepo) GetByID(ctx context.Context, id string) (*model.Booking, error) {
	var b model.Booking
	err := r.db.QueryRowContext(
		ctx,
		`SELECT id, user_id, flight_id, flight_number, origin_airport,
		destination_airport, departure_time, passenger_name, passenger_email,
		passenger_phone, seat_count, total_price, status, created_at, updated_at
		FROM bookings WHERE id = $1`,
		id,
	).Scan(
		&b.ID, &b.UserID, &b.FlightID, &b.FlightNumber,
		&b.OriginAirport, &b.DestinationAirport, &b.DepartureTime,
		&b.PassengerName, &b.PassengerEmail, &b.PassengerPhone, &b.SeatCount,
		&b.TotalPrice, &b.Status, &b.CreatedAt, &b.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, err
	}
	return &b, err
}

func (r *BookingRepo) GetByUserID(ctx context.Context, userID string) ([]model.Booking, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, user_id, flight_id, flight_number, origin_airport,
		destination_airport, departure_time, passenger_name, passenger_email,
		passenger_phone, seat_count, total_price, status, created_at, updated_at
		FROM bookings WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookings []model.Booking
	for rows.Next() {
		var b model.Booking
		err := rows.Scan(
			&b.ID,
			&b.UserID,
			&b.FlightID,
			&b.FlightNumber,
			&b.OriginAirport,
			&b.DestinationAirport,
			&b.DepartureTime,
			&b.PassengerName,
			&b.PassengerEmail,
			&b.PassengerPhone,
			&b.SeatCount,
			&b.TotalPrice,
			&b.Status,
			&b.CreatedAt,
			&b.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		bookings = append(bookings, b)
	}
	return bookings, rows.Err()
}

func (r *BookingRepo) UpdateStatus(ctx context.Context, tx *sql.Tx, id, status string) error {
	_, err := tx.ExecContext(
		ctx,
		"UPDATE bookings SET status = $1, updated_at = NOW() WHERE id = $2",
		status, id,
	)
	return err
}
