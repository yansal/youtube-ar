package broker

import (
	"context"
	"time"

	"github.com/go-redis/redis"
	"github.com/yansal/youtube-ar/log"
)

// New returns a new Broker.
func New(r *redis.Client, log log.Logger) *Broker {
	return &Broker{redis: r, log: log}
}

// Broker is a broker.
type Broker struct {
	log   log.Logger
	redis *redis.Client
}

// Send sends payload to queue.
func (b *Broker) Send(ctx context.Context, queue string, payload string) error {
	return b.redis.LPush(queue, payload).Err()
}

// Receive pops next item from queue and calls handler.
func (b *Broker) Receive(ctx context.Context, queue string, handler Handler) error {
	tmp := queue + ":tmp"
	payload, err := b.redis.BRPopLPush(queue, tmp, 0).Result()
	if err == redis.Nil {
		return nil
	} else if err != nil {
		return err
	}

	fields := []log.Field{
		log.String("queue", queue),
		log.String("payload", payload),
	}

	defer func() {
		if err := b.redis.LRem(tmp, 1, payload).Err(); err != nil {
			b.log.Log(ctx, err.Error())
		}
	}()

	start := time.Now()
	err = handler(ctx, payload)
	fields = append(fields, log.Stringer("duration", time.Since(start)))
	if err == nil {
		b.log.Log(ctx, queue+": "+payload, fields...)
		return nil
	}

	b.log.Log(ctx, err.Error(), fields...)

	failed := queue + ":failed"
	return b.Send(ctx, failed, payload)
}

// Handler is a broker handler.
type Handler func(ctx context.Context, payload string) error

// PopNextFailed pops next element from failed queue.
func (b *Broker) PopNextFailed(ctx context.Context, queue string) (string, error) {
	return b.redis.RPop(queue + ":failed").Result()
}
