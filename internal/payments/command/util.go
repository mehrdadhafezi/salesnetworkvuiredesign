package command

import (
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

func itoa(v int64) string { return strconv.FormatInt(v, 10) }

func timeKey(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

func truncate300(s string) string {
	if len(s) <= 300 {
		return s
	}
	r := []rune(s)
	for len(string(r)) > 300 {
		r = r[:len(r)-1]
	}
	return string(r)
}
