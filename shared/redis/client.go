package redisx

import (
	"fmt"
	"os"

	redis "github.com/redis/go-redis/v9"
)

// NewClient создает redis-клиент из ENV REDIS_URL или параметров REDIS_HOST/REDIS_PORT/REDIS_DB/REDIS_PASSWORD
func NewClient() (*redis.Client, error) {
    if url := os.Getenv("REDIS_URL"); url != "" {
        opt, err := redis.ParseURL(url)
        if err != nil {
            return nil, fmt.Errorf("invalid REDIS_URL: %w", err)
        }
        return redis.NewClient(opt), nil
    }
    host := os.Getenv("REDIS_HOST")
    if host == "" { host = "127.0.0.1" }
    port := os.Getenv("REDIS_PORT")
    if port == "" { port = "6379" }
    db := 0
    if v := os.Getenv("REDIS_DB"); v != "" {
        // ignore error, default 0
        fmt.Sscanf(v, "%d", &db)
    }
    pwd := os.Getenv("REDIS_PASSWORD")
    return redis.NewClient(&redis.Options{Addr: host+":"+port, DB: db, Password: pwd}), nil
}


