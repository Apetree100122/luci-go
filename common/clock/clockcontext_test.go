// Copyright 2015 The LUCI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package clock

import (
	"context"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

// manualClock is a partial Clock implementation that allows us to release
// blocking calls.
type manualClock struct {
	Clock

	now             time.Time
	timeoutCallback func(time.Duration) bool
	testFinishedC   chan struct{}
}

func (mc *manualClock) Now() time.Time {
	return mc.now
}

func (mc *manualClock) NewTimer(c context.Context) Timer {
	return &manualTimer{
		ctx:     c,
		mc:      mc,
		resultC: make(chan TimerResult),
	}
}

type manualTimer struct {
	Timer

	ctx     context.Context
	mc      *manualClock
	resultC chan TimerResult

	running bool
	stopC   chan struct{}
}

func (mt *manualTimer) GetC() <-chan TimerResult { return mt.resultC }

func (mt *manualTimer) Reset(d time.Duration) bool {
	running := mt.Stop()
	mt.stopC, mt.running = make(chan struct{}), true

	go func() {
		ar := TimerResult{}
		defer func() {
			mt.resultC <- ar
		}()

		// If we are instructed to immediately timeout, do so.
		if cb := mt.mc.timeoutCallback; cb != nil && cb(d) {
			return
		}

		select {
		case <-mt.ctx.Done():
			ar.Err = mt.ctx.Err()
		case <-mt.mc.testFinishedC:
			break
		}
	}()
	return running
}

func (mt *manualTimer) Stop() bool {
	if !mt.running {
		return false
	}

	mt.running = false
	close(mt.stopC)
	return true
}

func wait(c context.Context) error {
	<-c.Done()
	return c.Err()
}

func TestClockContext(t *testing.T) {
	t.Parallel()

	Convey(`A manual testing clock`, t, func() {
		mc := manualClock{
			now:           time.Date(2016, 1, 1, 0, 0, 0, 0, time.Local),
			testFinishedC: make(chan struct{}),
		}
		defer close(mc.testFinishedC)

		Convey(`A context with a deadline wrapping a cancellable parent`, func() {
			Convey(`Successfully reports its deadline.`, func() {
				cctx, _ := context.WithCancel(Set(context.Background(), &mc))
				ctx, _ := WithTimeout(cctx, 10*time.Millisecond)

				deadline, ok := ctx.Deadline()
				So(ok, ShouldBeTrue)
				So(deadline.After(mc.now), ShouldBeTrue)
			})

			Convey(`Will successfully time out.`, func() {
				mc.timeoutCallback = func(time.Duration) bool {
					return true
				}

				cctx, _ := context.WithCancel(Set(context.Background(), &mc))
				ctx, _ := WithTimeout(cctx, 10*time.Millisecond)
				So(wait(ctx).Error(), ShouldEqual, context.DeadlineExceeded.Error())
			})

			Convey(`Will successfully cancel with its cancel func.`, func() {
				cctx, _ := context.WithCancel(Set(context.Background(), &mc))
				ctx, cf := WithTimeout(cctx, 10*time.Millisecond)

				go func() {
					cf()
				}()
				So(wait(ctx), ShouldEqual, context.Canceled)
			})

			Convey(`Will successfully cancel if the parent is canceled.`, func() {
				cctx, pcf := context.WithCancel(Set(context.Background(), &mc))
				ctx, _ := WithTimeout(cctx, 10*time.Millisecond)

				go func() {
					pcf()
				}()
				So(wait(ctx), ShouldEqual, context.Canceled)
			})
		})

		Convey(`A context with a deadline wrapping a parent with a shorter deadline`, func() {
			cctx, _ := context.WithTimeout(context.Background(), 10*time.Millisecond)
			ctx, cf := WithTimeout(cctx, 1*time.Hour)

			Convey(`Will successfully time out.`, func() {
				mc.timeoutCallback = func(d time.Duration) bool {
					return d == 10*time.Millisecond
				}

				So(wait(ctx).Error(), ShouldEqual, context.DeadlineExceeded.Error())
			})

			Convey(`Will successfully cancel with its cancel func.`, func() {
				go func() {
					cf()
				}()
				So(wait(ctx), ShouldEqual, context.Canceled)
			})
		})

		Convey(`A context with a deadline in the past`, func() {
			ctx, _ := WithDeadline(context.Background(), mc.now.Add(-time.Second))

			Convey(`Will time out immediately.`, func() {
				So(wait(ctx).Error(), ShouldEqual, context.DeadlineExceeded.Error())
			})
		})
	})
}
