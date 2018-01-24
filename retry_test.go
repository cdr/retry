package retry

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestRetry(t *testing.T) {
	t.Run("Attempts", func(t *testing.T) {
		t.Parallel()
		t.Run("Respects count and sleeps between attempts", func(t *testing.T) {
			count := 0
			start := time.Now()

			const sleep = time.Millisecond * 10

			New(sleep).Attempts(5).Run(func() error {
				count++
				return errors.Errorf("asdfasdf")
			})

			assert.Equal(t, 5, count)
			assert.WithinDuration(t, start.Add(sleep*5), time.Now(), sleep)
		})

		t.Run("returns as soon as error is nil", func(t *testing.T) {
			start := time.Now()

			var count int
			New(time.Minute).Attempts(100).Run(func() error {
				count++
				return nil
			})

			assert.Equal(t, 1, count)
			assert.WithinDuration(t, time.Now(), start, time.Millisecond)
		})
	})
	t.Run("Backoff", func(t *testing.T) {
		t.Parallel()
		t.Run("return when nil", func(t *testing.T) {
			var count int
			err := New(time.Millisecond).Timeout(time.Minute).Backoff(time.Second).Run(func() error {
				count++
				if count == 10 {
					return nil
				}
				return io.EOF
			})

			assert.Equal(t, 10, count)
			assert.NoError(t, err)
		})
	})

	t.Run("Timeout", func(t *testing.T) {
		t.Parallel()
		t.Run("Respects timeout and sleeps between attempts", func(t *testing.T) {
			count := 0
			start := time.Now()

			const sleep = time.Millisecond * 10

			New(sleep).Timeout(sleep * 5).Run(func() error {
				count++
				return errors.Errorf("asdfasdf")
			})

			assert.Equal(t, 5, count)
			assert.WithinDuration(t, start.Add(sleep*5), time.Now(), sleep)
		})

		t.Run("returns as soon as error is nil", func(t *testing.T) {
			start := time.Now()

			New(time.Minute).Timeout(time.Hour).Run(func() error {
				return nil
			})

			assert.WithinDuration(t, time.Now(), start, time.Millisecond*10)
		})
	})

	t.Run("Cond", func(t *testing.T) {
		t.Parallel()
		count := 0

		err := New(time.Millisecond).Condition(OnErrors(io.ErrUnexpectedEOF, io.EOF)).Run(func() error {
			if count == 5 {
				return io.ErrNoProgress
			}
			count++
			if count%2 == 0 {
				return errors.WithStack(io.ErrUnexpectedEOF)
			}
			return io.EOF
		})

		assert.Equal(t, count, 5)
		assert.Equal(t, io.ErrNoProgress, err)

	})

	t.Run("Context", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())

		var count int

		err := New(time.Millisecond).Context(ctx).Run(func() error {
			if count == 3 {
				cancel()
			} else {
				count++
			}
			return io.EOF
		})

		assert.Equal(t, 3, count)
		assert.Equal(t, io.EOF, err)
	})

	t.Run("Jitter", func(t *testing.T) {
		t.Parallel()
		var count int

		var durs []time.Duration
		last := time.Now()
		New(time.Millisecond).Attempts(500).Jitter(0.9999).Run(func() error {
			durs = append(durs, time.Since(last))
			last = time.Now()
			count++
			return io.EOF
		})

		avg := avgDurations(durs)
		t.Logf("avg dur: %v", avg)

		if avg < time.Microsecond*900 || avg > time.Microsecond*1200 {
			t.Errorf("bad avg %v", avg)
		}
	})
}

func avgDurations(durs []time.Duration) time.Duration {
	var sum time.Duration
	for _, dur := range durs {
		sum += dur
	}
	return sum / time.Duration(len(durs))
}
