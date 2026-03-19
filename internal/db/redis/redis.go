package redis

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/redis/go-redis/v9"
)

func Client(url string) (*redis.Client, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, errors.Wrap(err, "invalid redis url")
	}

	rdb := redis.NewClient(opts)

	if err = rdb.Ping(context.Background()).Err(); err != nil {
		return nil, errors.Wrap(err, "redis ping failed")
	}

	return rdb, nil
}
