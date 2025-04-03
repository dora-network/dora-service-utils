package math

import (
	"fmt"
	"time"
)

// UnixFromDate converts an RFC1123 date, as we would expect in an asset coupon date or maturity date,
// into unix seconds. Error on invalid format.
func UnixFromDate(date string) (int64, error) {
	t, err := time.Parse(time.RFC1123, date)
	if err != nil {
		return 0, fmt.Errorf("%s is not a valid RFC1123 time: %s", date, err.Error())
	}
	return t.Unix(), nil
}

// DateFromUnix converts a unix timestamp into an RFC1123 date.
func DateFromUnix(unixSeconds int64) string {
	return time.Unix(unixSeconds, 0).Format(time.RFC1123)
}
