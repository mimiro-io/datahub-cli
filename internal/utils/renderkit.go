package utils

import "time"

var (
	// load up some variables used for formatting dates
	loc, _ = time.LoadLocation("Local")
	format = "2006-01-02 15:04:05"
)

func Date(dt time.Time) string {
	localTime := dt.In(loc)
	return localTime.Format(format)
}
