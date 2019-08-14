package broker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redis"
	"github.com/yansal/youtube-ar/api/log"
)

func assertf(t *testing.T, ok bool, msg string, args ...interface{}) {
	t.Helper()
	if !ok {
		t.Errorf(msg, args...)
	}
}

type redisMock struct {
	brpoplpushFunc func(source, destination string, timeout time.Duration) *redis.StringCmd
	lremFunc       func(key string, count int64, value interface{}) *redis.IntCmd
	lpushFunc      func(key string, values ...interface{}) *redis.IntCmd
}

func (r redisMock) LPush(key string, values ...interface{}) *redis.IntCmd {
	return r.lpushFunc(key, values...)
}
func (r redisMock) BRPopLPush(source, destination string, timeout time.Duration) *redis.StringCmd {
	return r.brpoplpushFunc(source, destination, timeout)
}
func (r redisMock) LRem(key string, count int64, value interface{}) *redis.IntCmd {
	return r.lremFunc(key, count, value)
}
func (r redisMock) RPop(key string) *redis.StringCmd {
	return nil
}

type logMock struct {
	logFunc func(ctx context.Context, msg string, fields ...log.Field)
}

func (l logMock) Log(ctx context.Context, msg string, fields ...log.Field) {
	if l.logFunc != nil {
		l.logFunc(ctx, msg, fields...)
	}
}

const (
	queue       = "queue"
	queuetmp    = "queue:tmp"
	queuefailed = "queue:failed"
	payload     = "payload"
)

func TestBrokerReceiveErr(t *testing.T) {
	var (
		lremed, lpushed bool
		serr            = "err"
	)
	b := Broker{
		log: logMock{
			logFunc: func(ctx context.Context, msg string, fields ...log.Field) {
				assertf(t, msg == serr, `expected msg to be %q, got %q`, serr, msg)
			},
		},
		redis: redisMock{
			brpoplpushFunc: func(source, destination string, timeout time.Duration) *redis.StringCmd {
				assertf(t, source == queue, `expected source to be %q, got %q`, queue, source)
				assertf(t, destination == queuetmp, `expected destination to be %q, got %q`, queuetmp, destination)
				return redis.NewStringResult(payload, nil)
			},
			lremFunc: func(key string, count int64, value interface{}) *redis.IntCmd {
				lremed = true
				assertf(t, key == queuetmp, `expected key to be %q, got %q`, queuetmp, key)
				assertf(t, payload == value, `expected value to be %q, got %+v`, payload, value)
				return redis.NewIntResult(0, nil)
			},
			lpushFunc: func(key string, values ...interface{}) *redis.IntCmd {
				lpushed = true
				assertf(t, key == queuefailed, `expected key to be %q, got %q`, queuefailed, key)
				assertf(t, len(values) == 1, `expected values to have length 1, got %+v`, values)
				assertf(t, payload == values[0], `expected values[0] to be %q, got %+v`, payload, values[0])
				return redis.NewIntResult(0, nil)
			},
		},
	}
	handler := func(ctx context.Context, in string) error {
		assertf(t, in == payload, `expected payload to be %q, got %q`, payload, in)
		return errors.New(serr)
	}

	if err := b.Receive(context.Background(), queue, handler); err != nil {
		t.Fatal(err)
	}
	assertf(t, lremed, `expected lrem to be called`)
	assertf(t, lpushed, `expected lpush to be called`)
}

func TestBrokerReceiveNoErr(t *testing.T) {
	var lremed bool
	b := Broker{
		log: logMock{},
		redis: redisMock{
			brpoplpushFunc: func(source, destination string, timeout time.Duration) *redis.StringCmd {
				assertf(t, source == queue, `expected source to be %q, got %q`, queue, source)
				assertf(t, destination == queuetmp, `expected destination to be %q, got %q`, queuetmp, destination)
				return redis.NewStringResult(payload, nil)
			},
			lremFunc: func(key string, count int64, value interface{}) *redis.IntCmd {
				lremed = true
				assertf(t, key == queuetmp, `expected key to be %q, got %q`, queuetmp, key)
				assertf(t, payload == value, `expected value to be %q, got %+v`, payload, value)
				return redis.NewIntResult(0, nil)
			},
		},
	}
	handler := func(ctx context.Context, in string) error {
		assertf(t, in == payload, `expected payload to be %q, got %q`, payload, in)
		return nil
	}

	if err := b.Receive(context.Background(), queue, handler); err != nil {
		t.Fatal(err)
	}
	assertf(t, lremed, `expected lrem to be called`)
}

func TestBrokerReceivePanic(t *testing.T) {
	var (
		lremed, lpushed bool
		serr            = "panic"
	)
	b := Broker{
		log: logMock{
			logFunc: func(ctx context.Context, msg string, fields ...log.Field) {
				assertf(t, msg == serr, `expected msg to be %q, got %q`, serr, msg)
			},
		},
		redis: redisMock{
			brpoplpushFunc: func(source, destination string, timeout time.Duration) *redis.StringCmd {
				assertf(t, source == queue, `expected source to be %q, got %q`, queue, source)
				assertf(t, destination == queuetmp, `expected destination to be %q, got %q`, queuetmp, destination)
				return redis.NewStringResult(payload, nil)
			},
			lremFunc: func(key string, count int64, value interface{}) *redis.IntCmd {
				lremed = true
				assertf(t, key == queuetmp, `expected key to be %q, got %q`, queuetmp, key)
				assertf(t, payload == value, `expected value to be %q, got %+v`, payload, value)
				return redis.NewIntResult(0, nil)
			},
			lpushFunc: func(key string, values ...interface{}) *redis.IntCmd {
				lpushed = true
				assertf(t, key == queuefailed, `expected key to be %q, got %q`, queuefailed, key)
				assertf(t, len(values) == 1, `expected values to have length 1, got %+v`, values)
				assertf(t, payload == values[0], `expected values[0] to be %q, got %+v`, payload, values[0])
				return redis.NewIntResult(0, nil)
			},
		},
	}
	handler := func(ctx context.Context, in string) error {
		assertf(t, in == payload, `expected payload to be %q, got %q`, payload, in)
		panic(serr)
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("unexpected panic: %v", r)
		}
	}()
	if err := b.Receive(context.Background(), queue, handler); err != nil {
		t.Fatal(err)
	}
	assertf(t, lremed, `expected lrem to be called`)
	assertf(t, lpushed, `expected lpush to be called`)
}

func TestBrokerReceiveCancel(t *testing.T) {
	timeout := time.Millisecond
	b := Broker{
		redis: redisMock{
			brpoplpushFunc: func(string, string, time.Duration) *redis.StringCmd {
				<-time.After(2 * timeout)
				return redis.NewStringResult(payload, errors.New("test timeout"))
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	err := b.Receive(ctx, queue, nil)
	assertf(t, err == context.DeadlineExceeded,
		"expected to get %v, got %v", context.DeadlineExceeded, err)
}
