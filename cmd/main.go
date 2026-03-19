package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/danilovaalina/goshrink/internal/api"
	"github.com/danilovaalina/goshrink/internal/config"
	"github.com/danilovaalina/goshrink/internal/db/postgres"
	"github.com/danilovaalina/goshrink/internal/db/redis"
	"github.com/danilovaalina/goshrink/internal/repository"
	"github.com/danilovaalina/goshrink/internal/service"
	"github.com/rs/zerolog/log"
)

func main() {
	cf, err := config.Load()
	if err != nil {
		log.Fatal().Stack().Err(err).Send()
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.Pool(ctx, cf.DatabaseURL)
	if err != nil {
		log.Fatal().Stack().Err(err).Send()
	}
	defer pool.Close()

	r, err := redis.Client(cf.RedisURL)
	if err != nil {
		log.Fatal().Stack().Err(err).Send()
	}

	a := api.New(service.New(repository.New(pool, r)))
	err = http.ListenAndServe(cf.Addr, a)
	if err != nil {
		log.Fatal().Stack().Err(err).Send()
	}

	err = a.Start(cf.Addr)
	if err != nil {
		log.Fatal().Stack().Err(err).Send()
	}
}
