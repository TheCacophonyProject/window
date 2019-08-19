// Copyright 2018 The Cacophony Project. All rights reserved.
// Use of this source code is governed by the Apache License Version 2.0;
// see the LICENSE file for further details.

package window

import (
	"testing"
	"time"

	sunrise "github.com/nathan-osman/go-sunrise"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testLatitude  = -41
	testLongitude = 175
)

func TestNoWindow(t *testing.T) {
	zero := time.Time{}.Format(hourMinuteFormat)
	w, err := New(zero, zero, 0, 0)
	assert.NoError(t, err)
	assert.True(t, w.Active())
}

func TestSameStartEnd(t *testing.T) {
	// Treat this as "no window"
	now := time.Now().Format(hourMinuteFormat)
	w, err := New(now, now, 0, 0)
	require.NoError(t, err)
	assert.True(t, w.Active())
	assert.Equal(t, time.Duration(0), w.Until())
}

func TestStartLessThanEnd(t *testing.T) {
	w, err := New(mkTime(9, 10), mkTime(17, 30), 0, 0)
	assert.NoError(t, err)
	interval := time.Duration(30 * time.Minute)
	w.Now = mkNow(9, 9)
	assert.False(t, w.Active())
	assert.Equal(t, time.Minute, w.Until())

	assert.Equal(t, time.Duration(-1), w.UntilNextInterval(interval))
	assert.Equal(t, time.Duration(0), w.UntilEnd())

	w.Now = mkNow(9, 10)
	assert.True(t, w.Active())
	assert.Equal(t, time.Duration(0), w.Until())
	assert.Equal(t, time.Duration(30*time.Minute), w.UntilNextInterval(interval))

	w.Now = mkNow(12, 0)
	assert.True(t, w.Active())
	assert.Equal(t, time.Duration(0), w.Until())
	assert.Equal(t, time.Duration(10*time.Minute), w.UntilNextInterval(interval))
	assert.Equal(t, time.Duration((5*60+30)*time.Minute), w.UntilEnd())

	w.Now = mkNow(17, 29)
	assert.True(t, w.Active())
	assert.Equal(t, time.Duration(0), w.Until())
	assert.Equal(t, time.Duration(-1), w.UntilNextInterval(interval))
	assert.Equal(t, time.Minute, w.UntilEnd())

	w.Now = mkNow(17, 30)
	assert.False(t, w.Active())
	assert.Equal(t, 940*time.Minute, w.Until())
	assert.Equal(t, time.Duration(-1), w.UntilNextInterval(interval))
	assert.Equal(t, time.Duration(0), w.UntilEnd())
}

func TestStartGreaterThanEnd(t *testing.T) {
	// Window goes over midnight
	w, err := New(mkTime(22, 10), mkTime(9, 50), 0, 0)
	assert.NoError(t, err)
	interval := time.Duration(30 * time.Minute)

	w.Now = mkNow(22, 9)
	assert.False(t, w.Active())
	assert.Equal(t, time.Minute, w.Until())
	assert.Equal(t, time.Duration(-1), w.UntilNextInterval(interval))

	w.Now = mkNow(22, 10)
	assert.True(t, w.Active())
	assert.Equal(t, time.Duration(0), w.Until())

	w.Now = mkNow(23, 59)
	assert.True(t, w.Active())
	assert.Equal(t, time.Duration(0), w.Until())
	assert.Equal(t, time.Duration(11*time.Minute), w.UntilNextInterval(interval))
	assert.Equal(t, time.Duration((9*60+51)*time.Minute), w.UntilEnd())

	w.Now = mkNow(0, 0)
	assert.True(t, w.Active())
	assert.Equal(t, time.Duration(0), w.Until())
	assert.Equal(t, time.Duration(10*time.Minute), w.UntilNextInterval(interval))

	w.Now = mkNow(0, 1)
	assert.True(t, w.Active())
	assert.Equal(t, time.Duration(0), w.Until())
	assert.Equal(t, time.Duration(9*time.Minute), w.UntilNextInterval(interval))

	w.Now = mkNow(2, 0)
	assert.True(t, w.Active())
	assert.Equal(t, time.Duration(0), w.Until())

	w.Now = mkNow(9, 49)
	assert.True(t, w.Active())
	assert.Equal(t, time.Duration(0), w.Until())
	assert.Equal(t, time.Duration(time.Minute), w.UntilEnd())

	w.Now = mkNow(9, 49)
	assert.True(t, w.Active())
	assert.Equal(t, time.Duration(0), w.Until())
	assert.Equal(t, time.Duration(-1), w.UntilNextInterval(interval))

	w.Now = mkNow(9, 50)
	assert.False(t, w.Active())
	assert.Equal(t, 740*time.Minute, w.Until())
	assert.Equal(t, time.Duration(-1), w.UntilNextInterval(interval))
}

func TestMorningToMorning(t *testing.T) {
	// Window not active just between 10am and 11am each day.
	w, err := New(mkTime(11, 0), mkTime(10, 0), 0, 0)
	assert.NoError(t, err)

	w.Now = mkNow(9, 59)
	assert.True(t, w.Active())
	assert.Equal(t, time.Duration(0), w.Until())
	assert.Equal(t, time.Minute, w.UntilEnd())

	w.Now = mkNow(10, 0)
	assert.False(t, w.Active())
	assert.Equal(t, time.Hour, w.Until())

	w.Now = mkNow(10, 59)
	assert.False(t, w.Active())
	assert.Equal(t, time.Minute, w.Until())

	w.Now = mkNow(11, 0)
	assert.True(t, w.Active())
	assert.Equal(t, time.Duration(0), w.Until())
	assert.Equal(t, 23*time.Hour, w.UntilEnd())

	w.Now = mkNow(18, 0)
	assert.True(t, w.Active())
	assert.Equal(t, time.Duration(0), w.Until())
	assert.Equal(t, 16*time.Hour, w.UntilEnd())
}

func TestSettingLatLong(t *testing.T) {
	lat := 123.0
	long := 80.0
	w, err := New("1h", "1h", lat, long)
	require.NoError(t, err)
	assert.Equal(t, lat, w.latitude)
	assert.Equal(t, long, w.longitude)
}

func TestParsingOfWindow(t *testing.T) {
	w, err := New("20:10", "08:00", 0, 0)
	require.NoError(t, err)
	require.False(t, w.start.Relative)
	assert.Equal(t, w.start.Time.Hour(), 20)
	assert.Equal(t, w.start.Time.Minute(), 10)
	require.False(t, w.end.Relative)
	assert.Equal(t, w.end.Time.Hour(), 8)
	assert.Equal(t, w.end.Time.Minute(), 0)

	w, err = New("-1h20m", "10:31", 0, 0)
	require.NoError(t, err)
	require.True(t, w.start.Relative)
	assert.Equal(t, w.start.RelativeDuration, -1*(time.Hour+time.Minute*20))
	require.False(t, w.end.Relative)
	assert.Equal(t, w.end.Time.Hour(), 10)
	assert.Equal(t, w.end.Time.Minute(), 31)

	w, err = New("21:59", "3h", 0, 0)
	require.NoError(t, err)
	require.False(t, w.start.Relative)
	assert.Equal(t, w.start.Time.Hour(), 21)
	assert.Equal(t, w.start.Time.Minute(), 59)
	require.True(t, w.end.Relative)
	assert.Equal(t, w.end.RelativeDuration, 3*time.Hour)

	w, err = New("30m", "-1h45m", 0, 0)
	require.NoError(t, err)
	require.True(t, w.start.Relative)
	assert.Equal(t, w.start.RelativeDuration, 30*time.Minute)
	require.True(t, w.end.Relative)
	assert.Equal(t, w.end.RelativeDuration, -1*(time.Hour+45*time.Minute))

	_, err = New("abc", "1:30", 0, 0)
	assert.Error(t, err)
	_, err = New("1:30", "abc", 0, 0)
	assert.Error(t, err)
	_, err = New("-1a", "1:30", 0, 0)
	assert.Error(t, err)
}

func TestSunriseSunset(t *testing.T) {
	nzTimeLoc, err := time.LoadLocation("NZ")
	require.NoError(t, err)
	notActiveNowDate := mkNowDate(2000, 1, 2, 12, 0, nzTimeLoc)
	w, err := New("-1h", "2h", testLatitude, testLongitude) // Window one hour before sunset to two hours after sunrise
	require.NoError(t, err)
	w.Now = notActiveNowDate
	require.Equal(t, time.Duration(-1*time.Hour), w.start.RelativeDuration)
	require.Equal(t, time.Duration(2*time.Hour), w.end.RelativeDuration)
	_, todaySunset := sunrise.SunriseSunset(testLatitude, testLongitude, 2000, 1, 2)
	tomorrowSunrise, tomorrowSunset := sunrise.SunriseSunset(testLatitude, testLongitude, 2000, 1, 3)
	_, yesterdaySunset := sunrise.SunriseSunset(testLatitude, testLongitude, 2000, 1, 1)

	assert.Equal(t, yesterdaySunset.Add(-1*time.Hour), w.PreviousStart())
	assert.Equal(t, todaySunset.Add(-1*time.Hour), w.NextStart())
	assert.Equal(t, tomorrowSunrise.Add(2*time.Hour), w.NextEnd())
	assert.False(t, w.Active())
	assert.Equal(t, w.NextStart().Sub(notActiveNowDate()), w.Until())
	assert.Equal(t, time.Duration(0), w.UntilEnd())
	assert.Equal(t, time.Duration(-1), w.UntilNextInterval(5*time.Minute))

	activeNowDate := mkNowDate(2000, 1, 2, 21, 1, nzTimeLoc)
	w.Now = activeNowDate
	assert.Equal(t, todaySunset.Add(-1*time.Hour), w.PreviousStart())
	assert.Equal(t, tomorrowSunset.Add(-1*time.Hour), w.NextStart())
	assert.Equal(t, tomorrowSunrise.Add(2*time.Hour), w.NextEnd())
	assert.True(t, w.Active())
	assert.Equal(t, time.Duration(0), w.Until())
	assert.Equal(t, w.NextEnd().Sub(activeNowDate()), w.UntilEnd())
	timeUntil14thInterval := w.PreviousStart().Add(time.Duration(14 * 5 * time.Minute)).Sub(activeNowDate())
	assert.Equal(t, timeUntil14thInterval, w.UntilNextInterval(5*time.Minute))
}

func mkTime(hour, minute int) string {
	return time.Date(1, 1, 1, hour, minute, 0, 0, time.UTC).Format(hourMinuteFormat)
}

func mkNow(hour, minute int) func() time.Time {
	return func() time.Time {
		return time.Date(2017, 1, 2, hour, minute, 0, 0, time.UTC)
	}
}

func mkNowDate(year int, month time.Month, day, hour, minute int, loc *time.Location) func() time.Time {
	return func() time.Time {
		return time.Date(year, month, day, hour, minute, 0, 0, loc)
	}
}
