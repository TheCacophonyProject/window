// Copyright 2018 The Cacophony Project. All rights reserved.
// Use of this source code is governed by the Apache License Version 2.0;
// see the LICENSE file for further details.

package window

import (
	"fmt"
	"time"

	sunrise "github.com/nathan-osman/go-sunrise"
)

const (
	hourMinuteFormat = "15:04"
)

// New creates a Window instance which represents a recurring window
// between two times of day. If `start` is after `end` then the time
// window is assumed to cross over midnight. If `start` and `end` are
// the same then the window is always active.
func New(start, end string, lat, long float64) (*Window, error) {
	startTime, err := parseAbsOrRelField(start)
	if err != nil {
		return nil, err
	}
	endTime, err := parseAbsOrRelField(end)
	if err != nil {
		return nil, err
	}

	if !startTime.Relative && start == end {
		return &Window{NoWindow: true}, nil
	}

	return &Window{
		start:     startTime,
		end:       endTime,
		Latitude:  lat,
		Longitude: long,
		Now:       time.Now,
	}, nil
}

// Window represents a recurring window between two times of day.
// The Now field can be use to override the time source (for testing).
type Window struct {
	start *absOrRelTime
	end   *absOrRelTime

	Latitude  float64
	Longitude float64

	Now func() time.Time

	NoWindow bool
}

func parseAbsOrRelField(timeStr string) (*absOrRelTime, error) {
	t := &absOrRelTime{}

	absTime, err := time.Parse("15:04", timeStr)
	if err == nil {
		t.Time = absTime
		t.Relative = false
		return t, nil
	}

	duration, err := time.ParseDuration(timeStr)
	if err == nil {
		t.RelativeDuration = duration
		t.Relative = true
		return t, nil
	}

	return nil, fmt.Errorf("could not parse '%s' as a time or duration", timeStr)
}

type absOrRelTime struct {
	Time             time.Time
	Relative         bool
	RelativeDuration time.Duration
}

// NextEnd will give the next time the window will end.
func (w *Window) NextEnd() time.Time {
	if w.end.Relative {
		return w.nextRelativeEnd()
	}
	return nextAbsTime(w.Now(), w.end.Time)
}

// NextStart will give the next time the windiw will start.
func (w *Window) NextStart() time.Time {
	if w.start.Relative {
		return w.nextRelativeStart()
	}
	return nextAbsTime(w.Now(), w.start.Time)
}

// PreviousStart will give the time the window last started.
func (w *Window) PreviousStart() time.Time {
	if w.start.Relative {
		return w.previousRelativeStart()
	}
	return nextAbsTime(w.Now().Add(-24*time.Hour), w.start.Time)
}

func (w *Window) relativeSunriseOn(year int, month time.Month, day int) time.Time {
	if !w.start.Relative {
		return time.Time{}
	}
	sr, _ := sunrise.SunriseSunset(w.Latitude, w.Longitude, year, month, day)
	return sr.Add(w.end.RelativeDuration)
}

func (w *Window) relativeSunsetOn(year int, month time.Month, day int) time.Time {
	if !w.end.Relative {
		return time.Time{}
	}
	_, ss := sunrise.SunriseSunset(w.Latitude, w.Longitude, year, month, day)
	return ss.Add(w.start.RelativeDuration)
}

func (w *Window) nextRelativeEnd() time.Time {
	now := w.Now()
	t := w.relativeSunriseOn(now.Year(), now.Month(), now.Day())
	if t.After(now) {
		return t
	}
	return w.relativeSunriseOn(now.Year(), now.Month(), now.Day()+1)
}

func (w *Window) nextRelativeStart() time.Time {
	now := w.Now()
	t := w.relativeSunsetOn(now.Year(), now.Month(), now.Day())
	if t.After(now) {
		return t
	}
	return w.relativeSunsetOn(now.Year(), now.Month(), now.Day()+1)
}

func (w *Window) previousRelativeStart() time.Time {
	now := w.Now()
	t := w.relativeSunsetOn(now.Year(), now.Month(), now.Day())
	if t.Before(now) {
		return t
	}
	return w.relativeSunsetOn(now.Year(), now.Month(), now.Day()-1)
}

func nextAbsTime(now, absTime time.Time) time.Time {
	absTime = setTimeHourAndMinute(now, absTime.Hour(), absTime.Minute())
	if absTime.After(now) {
		return absTime
	}
	now = addDay(now)
	return setTimeHourAndMinute(now, absTime.Hour(), absTime.Minute())
}

func setTimeHourAndMinute(t time.Time, hour, minute int) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), hour, minute, 0, 0, t.Location())
}

// Active returns true if the time window is currently active.
func (w *Window) Active() bool {
	if w.NoWindow {
		return true
	}
	return w.NextEnd().Before(w.NextStart())
}

// Until returns the duration until the next time window starts.
func (w *Window) Until() time.Duration {
	if w.NoWindow || w.Active() {
		return time.Duration(0)
	}
	return w.NextStart().Sub(w.Now())
}

func addDay(t time.Time) time.Time {
	return t.Add(24 * time.Hour)
}

// UntilEnd returns the duration until the end of the time window.
func (w *Window) UntilEnd() time.Duration {
	if w.NoWindow || !w.Active() {
		return time.Duration(0)
	}
	return w.NextEnd().Sub(w.Now())
}

// UntilNextInterval gets when the next interval starts.
// Only works when window is currently active.
func (w *Window) UntilNextInterval(interval time.Duration) time.Duration {
	if w.NoWindow || !w.Active() {
		return time.Duration(-1)
	}

	start := w.PreviousStart()
	end := w.NextEnd()
	elapsedTime := w.Now().Sub(start)
	nextInterval := start.Add(elapsedTime.Truncate(interval) + interval)

	if end.After(nextInterval) {
		return nextInterval.Sub(w.Now())
	}

	return time.Duration(-1)
}

func (w Window) String() string {
	s := fmt.Sprint("window starts at ")
	s = s + " and ends at "

	return fmt.Sprintf("window starts at %s and ends at %s", w.start.string("sunset"), w.end.string("sunrise"))
}

func (t absOrRelTime) string(relativeTo string) string {
	var s string
	if t.Relative {
		if t.RelativeDuration < 0 {
			s = s + fmt.Sprintf("%v before %s", -1*t.RelativeDuration, relativeTo)
		} else if t.RelativeDuration > 0 {
			s = s + fmt.Sprintf("%v after %s", t.RelativeDuration, relativeTo)
		} else {
			s = s + fmt.Sprintf("at %s", relativeTo)
		}
	} else {
		s = s + fmt.Sprintf("%v ", t.Time.Format(hourMinuteFormat))
	}
	return s
}
