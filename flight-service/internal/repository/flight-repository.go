package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"flight-booking/flight-service/internal/model"
	"flight-booking/flight-service/pkg/cache"
)

type FlightRepo struct {
 	db *sql.DB
	cache *cache.Cache
}

func NewFlightRepo(db *sql.DB, cache *cache.Cache) *FlightRepo {
	return &FlightRepo{
		db: db,
		cache: cache,
	}
}

func (r *FlightRepo) Search(ctx context.Context, origin string, destination string, date *time.Time) ([]model.Flight, error) {
	dateStr := ""
	if date != nil {
		dateStr = date.Format("2005-02-21")
	}
	
	if r.cache != nil {
		cached, hit, err := r.cache.GetSearch(ctx, origin, destination, dateStr)
		if err != nil {
			fmt.Printf("Cache error: %v, going to database\n", err)
		} else if hit && cached != nil {
			var flights []model.Flight
			for _, flight := range cached {
				flights = append(flights, model.Flight{
					ID: flight.ID,
					FlightNumber: flight.FlightNumber,
					DepartureDate: flight.DepartureDate,
					Airline: flight.Airline,
					OriginAirport: flight.OriginAirport,
					DestinationAirport: flight.DestinationAirport,
					DepartureTime: flight.DepartureTime,
					ArrivalTime: flight.ArrivalTime,
					TotalSeats:flight.TotalSeats,
					AvailableSeats: flight.AvailableSeats,
					Price: flight.Price,
					Status: flight.Status,
				})
			}
			return flights, nil
		}
	}
	
	query := `
		SELECT id, flight_number, departure_date, airline, origin_airport, 
		destination_airport, departure_time, arrival_time, total_seats, 
		available_seats, price, status 
		FROM flights 
		WHERE origin_airport = $1 AND destination_airport = $2 
		AND status = 'SCHEDULED'
	`

	args := []interface{}{origin, destination}
	if date != nil {
		query += " AND departure_date = $3"
		args = append(args, date)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var flights []model.Flight
	for rows.Next() {
		var f model.Flight
		err := rows.Scan(&f.ID, &f.FlightNumber, &f.DepartureDate, &f.Airline,
		&f.OriginAirport, &f.DestinationAirport, &f.DepartureTime,
		&f.ArrivalTime, &f.TotalSeats, &f.AvailableSeats, &f.Price, &f.Status)
		if err != nil {
			return nil, err
		}
		flights = append(flights, f)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	if r.cache != nil {
		var cacheFlights []cache.FlightCache
		for _, f := range flights {
			cacheFlights = append(cacheFlights, cache.FlightCache{
				ID:                 f.ID,
				FlightNumber:       f.FlightNumber,
				DepartureDate:      f.DepartureDate,
				Airline:            f.Airline,
			})
			if err := r.cache.SetSearch(ctx, origin, destination, dateStr, cacheFlights); err != nil {
				fmt.Printf("Cache error while setting search: %v\n", err)
			}
		}
	}

	return flights, nil
}

func (r *FlightRepo) GetByID(ctx context.Context, id string) (*model.Flight, error) {
	if r.cache != nil {
		cached, hit, err := r.cache.GetFlight(ctx, id)
		if err != nil {
			fmt.Printf("Cache error: %v, going to database\n", err)
		} else if hit && cached != nil {
			return &model.Flight{
				ID: cached.ID,
				FlightNumber: cached.FlightNumber,
				DepartureDate: cached.DepartureDate,
				Airline: cached.Airline,
				OriginAirport: cached.OriginAirport,
				DestinationAirport: cached.DestinationAirport,
				DepartureTime: cached.DepartureTime,
				ArrivalTime: cached.ArrivalTime,
				TotalSeats:cached.TotalSeats,
				AvailableSeats: cached.AvailableSeats,
				Price: cached.Price,
				Status: cached.Status,
			}, nil
		}
	}
	
	var f model.Flight
	err := r.db.QueryRowContext(
		ctx,
		`SELECT id, flight_number, departure_date, airline, origin_airport,
		destination_airport, departure_time, arrival_time, total_seats,
		available_seats, price, status 
		FROM flights WHERE id = $1`,
		id,
	).Scan(
		&f.ID, &f.FlightNumber, &f.DepartureDate, &f.Airline,
		&f.OriginAirport, &f.DestinationAirport, &f.DepartureTime,
		&f.ArrivalTime, &f.TotalSeats, &f.AvailableSeats, &f.Price, &f.Status,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if r.cache != nil {
		caching := &cache.FlightCache{
			ID: f.ID,
			FlightNumber: f.FlightNumber,
			DepartureDate: f.DepartureDate,
			Airline: f.Airline,
			OriginAirport: f.OriginAirport,
			DestinationAirport: f.DestinationAirport,
			DepartureTime: f.DepartureTime,
			ArrivalTime: f.ArrivalTime,
			TotalSeats:f.TotalSeats,
			AvailableSeats: f.AvailableSeats,
			Price: f.Price,
			Status: f.Status,
		}
		if err := r.cache.SetFlight(ctx, id, caching); err != nil {
			fmt.Printf("Cache error while writing: %v\n", err)
		}
	}

	return &f, err
}

func (r *FlightRepo) ReserveSeats(ctx context.Context, tx *sql.Tx, flightID string, seatCount int32, bookingID string) (string, error) {
	var available int32
	err := tx.QueryRowContext(
		ctx, 
		"SELECT available_seats FROM flights WHERE id = $1 FOR UPDATE", 
		flightID,
	).Scan(
		&available,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("Flight not found")
		}
		return "", err
	}

	if available < seatCount {
		return "", fmt.Errorf("Not enough seats")
	}

	_, err = tx.ExecContext(
		ctx, 
		"UPDATE flights SET available_seats = available_seats - $1, updated_at = NOW() WHERE id = $2", 
		seatCount, flightID,
	)
	if err != nil {
		return "", err
	}

	var reservationID string
	err = tx.QueryRowContext(
		ctx,
		`INSERT INTO seat_reservations (flight_id, booking_id, seat_count, status)
		VALUES ($1, $2, $3, 'ACTIVE')
		RETURNING id`, 
		flightID, bookingID, seatCount,
	).Scan(
		&reservationID,
	)

	if r.cache != nil {
		if err := r.cache.InvalidateFlight(ctx, flightID); err != nil {
			fmt.Printf("Cache error while invalidating flight: %v\n", err)
		}
	}

	return reservationID, err
}

func (r *FlightRepo) ReleaseReservation(ctx context.Context, tx *sql.Tx, bookingID string) (bool, error) {
	var reservationID, flightID string
	var seatCount int32

	err := tx.QueryRowContext(
		ctx,
		`SELECT id, flight_id, seat_count FROM seat_reservations 
		WHERE booking_id = $1 AND status = 'ACTIVE' FOR UPDATE`, 
		bookingID,
	).Scan(
		&reservationID,
		&flightID,
		&seatCount,
	)

	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	_, err = tx.ExecContext(
		ctx, 
		"UPDATE flights SET available_seats = available_seats + $1 WHERE id = $2", 
		seatCount, flightID,
	)
	if err != nil {
		return false, err
	}

	_, err = tx.ExecContext(
		ctx, 
		"UPDATE seat_reservations SET status = 'RELEASED', updated_at = NOW() WHERE id = $1", 
		reservationID,
	)

	if r.cache != nil {
		if err := r.cache.InvalidateFlight(ctx, flightID); err != nil {
			fmt.Printf("Cache error while invalidating flight: %v\n", err)
		}
	}

	return err == nil, err
}
