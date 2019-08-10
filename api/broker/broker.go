package broker

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis"
	"github.com/yansal/youtube-ar/api/log"
)

// New returns a new Broker.
func New(r Redis, log log.Logger) *Broker {
	return &Broker{redis: r, log: log}
}

// Redis is the redis interface required by Broker.
type Redis interface {
	LPush(key string, values ...interface{}) *redis.IntCmd
	BRPopLPush(source, destination string, timeout time.Duration) *redis.StringCmd
	LRem(key string, count int64, value interface{}) *redis.IntCmd
	RPop(key string) *redis.StringCmd
}

// Broker is a broker.
type Broker struct {
	log   log.Logger
	redis Redis
}

// Send sends payload to queue.
func (b *Broker) Send(ctx context.Context, queue string, payload string) error {
	return b.redis.LPush(queue, payload).Err()
}

// Receive pops next item from queue and calls handler.
func (b *Broker) Receive(ctx context.Context, queue string, handler Handler) error {
	fields := []log.Field{log.String("queue", queue)}
	tmp := queue + ":tmp"
	payload, err := b.redis.BRPopLPush(queue, tmp, 0).Result()
	if err == redis.Nil {
		return nil
	} else if err != nil {
		return err
	}

	fields = append(fields, log.String("payload", payload))

	defer func() {
		if err := b.redis.LRem(tmp, 1, payload).Err(); err != nil {
			b.log.Log(ctx, err.Error())
		}
	}()

	var (
		start = time.Now()
		herr  error
	)
	defer func() {
		fields = append(fields, log.Stringer("duration", time.Since(start)))
		if r := recover(); r != nil {
			herr = fmt.Errorf("%s", r)
		}
		if herr == nil {
			b.log.Log(ctx, queue+": "+payload, fields...)
			return
		}

		b.log.Log(ctx, herr.Error(), fields...)

		failed := queue + ":failed"
		if err := b.Send(ctx, failed, payload); err != nil {
			b.log.Log(ctx, err.Error())
		}
	}()

	herr = handler(ctx, payload)
	return nil
}

// Handler is a broker handler.
type Handler func(ctx context.Context, payload string) error

// PopNextFailed pops next element from failed queue.
func (b *Broker) PopNextFailed(ctx context.Context, queue string) (string, error) {
	return b.redis.RPop(queue + ":failed").Result()
}
