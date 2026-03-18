package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewCache() *Cache {
	host := os.Getenv("REDIS_HOST")
	if host == "" {
		host = "redis"
	}
	
	port := os.Getenv("REDIS_PORT")
	if port == "" {
		port = "6379"
	}
	
	ttlStr := os.Getenv("CACHE_TTL")
	ttl := 600 * time.Second
	if ttlStr != "" {
		if seconds, err := strconv.Atoi(ttlStr); err == nil {
			ttl = time.Duration(seconds) * time.Second
		}
	}

	client := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", host, port),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
	defer cancel()
	
	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("Redis connection failed: %v (cache will be disabled)", err)
		return &Cache{client: nil, ttl: ttl}
	}

	log.Println("Connected to Redis")
	return &Cache{client: client, ttl: ttl}
}


func (c *Cache) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

func (c *Cache) GetFlight(ctx context.Context, id string) (*FlightCache, bool, error) {
	if c.client == nil { // no cache
		return nil, false, nil
	}

	key := fmt.Sprintf("flight:%s", id)
	data, err := c.client.Get(ctx, key).Bytes()
	
	if err == redis.Nil {
		log.Printf("CACHE MISS: flight:%s", id)
		return nil, false, nil
	}
	if err != nil {
		log.Printf("Redis error: %v", err)
		return nil, false, err
	}

	log.Printf("CACHE HIT: flight:%s", id)
	
	var flight FlightCache
	if err := json.Unmarshal(data, &flight); err != nil {
		return nil, false, err
	}

	return &flight, true, nil
}

func (c *Cache) SetFlight(ctx context.Context, id string, flight *FlightCache) error {
	if c.client == nil {
		return nil
	}

	key := fmt.Sprintf("flight:%s", id)
	data, err := json.Marshal(flight)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, c.ttl).Err()
}

func (c *Cache) GetSearch(ctx context.Context, origin, destination string, date string) ([]FlightCache, bool, error) {
	if c.client == nil {
		return nil, false, nil
	}

	key := fmt.Sprintf("search:%s:%s:%s", origin, destination, date)
	data, err := c.client.Get(ctx, key).Bytes()
	
	if err == redis.Nil {
		log.Printf("CACHE MISS: search:%s", key)
		return nil, false, nil
	}
	if err != nil {
		log.Printf("Redis error: %v", err)
		return nil, false, err
	}

	log.Printf("CACHE HIT: search:%s", key)
	
	var flights []FlightCache
	if err := json.Unmarshal(data, &flights); err != nil {
		return nil, false, err
	}

	return flights, true, nil
}

func (c *Cache) SetSearch(ctx context.Context, origin, destination, date string, flights []FlightCache) error {
	if c.client == nil {
		return nil
	}

	key := fmt.Sprintf("search:%s:%s:%s", origin, destination, date)
	data, err := json.Marshal(flights)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, c.ttl).Err()
}

func (c *Cache) InvalidateFlight(ctx context.Context, id string) error {
	if c.client == nil {
		return nil
	}

	key := fmt.Sprintf("flight:%s", id)
	return c.client.Del(ctx, key).Err()
}

func (c *Cache) InvalidateSearch(ctx context.Context, origin, destination string) error {
	if c.client == nil {
		return nil
	}

	pattern := fmt.Sprintf("search:%s:%s:*", origin, destination)
	
	iter := c.client.Scan(ctx, 0, pattern, 0).Iterator()
	for ; iter.Next(ctx); {
		if err := c.client.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	
	return iter.Err()
}

type FlightCache struct {
	ID string `json:"id"`
	FlightNumber string `json:"flight_number"`
	DepartureDate time.Time `json:"departure_date"`
	Airline string `json:"airline"`
	OriginAirport string `json:"origin_airport"`
	DestinationAirport string `json:"destination_airport"`
	DepartureTime time.Time `json:"departure_time"`
	ArrivalTime time.Time `json:"arrival_time"`
	TotalSeats int32 `json:"total_seats"`
	AvailableSeats int32 `json:"available_seats"`
	Price float64 `json:"price"`
	Status string `json:"status"`
}
