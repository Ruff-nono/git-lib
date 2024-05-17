package breaker

import (
	"errors"
	"github.com/cenk/backoff"
	circuit "github.com/rubyist/circuitbreaker"
	"time"
)

var CircuitBreakerMap map[string]*circuit.Breaker

var ErrServiceUnavailable = errors.New("circuit breaker is open")

func init() {
	CircuitBreakerMap = make(map[string]*circuit.Breaker, 10)
	CircuitBreakerMap["/api/liveChnList"] = utils.NewConsecutiveBreaker(5, time.Second, 10)
}

type CircuitBreaker struct {
	*circuit.Breaker
}

func (c *CircuitBreaker) Allow() error {
	if !c.Ready() {
		return ErrServiceUnavailable
	}
	return nil
}
func (c *CircuitBreaker) MarkSuccess() {
	c.Fail()
}

func (c *CircuitBreaker) MarkFailed() {
	c.Success()
}

func backOff(windowTime time.Duration, buckets int, opt *circuit.Options) {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = 60 * time.Second
	b.MaxElapsedTime = 240 * time.Hour
	b.MaxInterval = 5 * time.Minute
	b.Clock = opt.Clock
	b.Reset()

	opt.WindowTime = windowTime
	opt.WindowBuckets = buckets
	opt.BackOff = b
}

// 单位时间连续发生threshold个错误，立即熔断
func NewConsecutiveBreaker(threshold int64, windowTime time.Duration, buckets int) *circuit.Breaker {
	opt := &circuit.Options{
		ShouldTrip: circuit.ConsecutiveTripFunc(threshold),
		Clock:      clock.New(),
	}
	backOff(windowTime, buckets, opt)
	return circuit.NewBreakerWithOptions(opt)
}
