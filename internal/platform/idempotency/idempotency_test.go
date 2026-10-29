package idempotency_test

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/idempotency"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pgtest"
)

func ctx(t *testing.T) context.Context {
	t.Helper()
	c, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return c
}

var at, _ = timeutil.ParseRFC3339("2026-10-29T10:00:00Z")

func req(key, body string) idempotency.Request {
	return idempotency.Request{Scope: "syn_alpha.create_widget", Key: key, Hash: idempotency.HashRequest([]byte(body))}
}

func TestRequestValidation(t *testing.T) {
	if idempotency.HashRequest([]byte("a")) != idempotency.HashRequest([]byte("a")) || idempotency.HashRequest([]byte("a")) == idempotency.HashRequest([]byte("b")) {
		t.Fatal("hash must be deterministic and input-sensitive")
	}
	good := req("synthetic-key-0001", "{}")
	if err := good.Validate(); err != nil {
		t.Fatal(err)
	}
	bad := []idempotency.Request{
		{Scope: "Bad", Key: good.Key, Hash: good.Hash},
		{Scope: good.Scope, Key: "short", Hash: good.Hash},
		{Scope: good.Scope, Key: "has space here!", Hash: good.Hash},
		{Scope: good.Scope, Key: strings.Repeat("k", 129), Hash: good.Hash},
		{Scope: good.Scope, Key: good.Key},
	}
	for i, r := range bad {
		if err := r.Validate(); !errors.Is(err, idempotency.ErrInvalidRequest) {
			t.Fatalf("request %d accepted: %v", i, err)
		}
	}
	for s, want := range map[idempotency.State]string{idempotency.Acquired: "acquired", idempotency.Replay: "replay",
		idempotency.Conflict: "conflict", idempotency.InProgress: "in_progress", idempotency.State(0): "invalid"} {
		if s.String() != want {
			t.Fatalf("state %d = %s", s, s)
		}
	}
}

func acquire(t *testing.T, r *db.Runner, rq idempotency.Request, complete json.RawMessage) idempotency.Acquisition {
	t.Helper()
	var a idempotency.Acquisition
	err := r.InTx(ctx(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error {
		var err error
		if a, err = idempotency.Acquire(c, tx, rq, at); err != nil {
			return err
		}
		if a.State == idempotency.Acquired && complete != nil {
			return idempotency.Complete(c, tx, rq, complete, at)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestReplayConflictAndInProgress(t *testing.T) {
	env := pgtest.New(t)
	alpha := env.Runner(t, pgtest.Alpha)
	first := req("synthetic-key-0001", `{"label":"a"}`)
	if a := acquire(t, alpha, first, json.RawMessage(`{"widget":1}`)); a.State != idempotency.Acquired {
		t.Fatalf("first = %v", a.State)
	}
	replay := acquire(t, alpha, first, nil)
	if replay.State != idempotency.Replay || string(replay.Result) != `{"widget": 1}` && string(replay.Result) != `{"widget":1}` {
		t.Fatalf("replay = %v %s", replay.State, replay.Result)
	}
	if a := acquire(t, alpha, req(first.Key, `{"label":"different"}`), nil); a.State != idempotency.Conflict {
		t.Fatalf("different hash = %v, want conflict", a.State)
	}
	// The same key in another scope is independent.
	other := first
	other.Scope = "syn_alpha.other_command"
	if a := acquire(t, alpha, other, nil); a.State != idempotency.Acquired {
		t.Fatalf("other scope = %v", a.State)
	}
	// A committed record that was never completed is IN_PROGRESS (unknown), never re-acquired.
	if a := acquire(t, alpha, other, nil); a.State != idempotency.InProgress {
		t.Fatalf("uncompleted = %v, want in_progress", a.State)
	}
	// A rolled-back acquisition leaves no record behind.
	rolled := req("synthetic-key-0002", "{}")
	_ = alpha.InTx(ctx(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error {
		if _, err := idempotency.Acquire(c, tx, rolled, at); err != nil {
			return err
		}
		return errors.New("command failed")
	})
	if a := acquire(t, alpha, rolled, json.RawMessage(`{}`)); a.State != idempotency.Acquired {
		t.Fatalf("after rollback = %v, want acquired", a.State)
	}
}

func TestCompleteRules(t *testing.T) {
	env := pgtest.New(t)
	alpha := env.Runner(t, pgtest.Alpha)
	rq := req("synthetic-key-0003", "{}")
	err := alpha.InTx(ctx(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error {
		if err := idempotency.Complete(c, tx, rq, nil, at); !errors.Is(err, idempotency.ErrNotAcquired) {
			t.Errorf("complete before acquire err = %v", err)
		}
		if _, err := idempotency.Acquire(c, tx, rq, at); err != nil {
			return err
		}
		if err := idempotency.Complete(c, tx, req(rq.Key, "other"), nil, at); !errors.Is(err, idempotency.ErrNotAcquired) {
			t.Errorf("complete with other hash err = %v", err)
		}
		if err := idempotency.Complete(c, tx, rq, json.RawMessage(`{bad`), at); !errors.Is(err, idempotency.ErrInvalidRequest) {
			t.Errorf("invalid JSON err = %v", err)
		}
		big := json.RawMessage(`"` + strings.Repeat("x", idempotency.MaxResultBytes) + `"`)
		if err := idempotency.Complete(c, tx, rq, big, at); !errors.Is(err, idempotency.ErrResultTooLarge) {
			t.Errorf("large result err = %v", err)
		}
		if err := idempotency.Complete(c, tx, rq, nil, at); err != nil {
			return err
		}
		if err := idempotency.Complete(c, tx, rq, nil, at); !errors.Is(err, idempotency.ErrNotAcquired) {
			t.Errorf("double complete err = %v", err)
		}
		if _, err := idempotency.Acquire(c, tx, idempotency.Request{}, at); !errors.Is(err, idempotency.ErrInvalidRequest) {
			t.Errorf("invalid acquire err = %v", err)
		}
		if _, err := idempotency.Acquire(c, tx, rq, timeutil.Instant{}); !errors.Is(err, idempotency.ErrInvalidRequest) {
			t.Errorf("zero time err = %v", err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	err = alpha.InTx(ctx(t), db.TxOptions{ReadOnly: true}, func(c context.Context, tx db.Tx) error {
		if _, err := idempotency.Acquire(c, tx, rq, at); !errors.Is(err, idempotency.ErrReadOnly) {
			t.Errorf("read-only acquire err = %v", err)
		}
		if err := idempotency.Complete(c, tx, rq, nil, at); !errors.Is(err, idempotency.ErrReadOnly) {
			t.Errorf("read-only complete err = %v", err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestConcurrentAcquisition proves exactly one concurrent caller performs the
// command; every other caller with the same request replays its result.
func TestConcurrentAcquisition(t *testing.T) {
	env := pgtest.New(t)
	alpha := env.Runner(t, pgtest.Alpha)
	rq := req("synthetic-key-0004", `{"label":"same"}`)
	var acquired, replayed, executed atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			err := alpha.InTx(context.Background(), db.TxOptions{}, func(c context.Context, tx db.Tx) error {
				a, err := idempotency.Acquire(c, tx, rq, at)
				if err != nil {
					return err
				}
				switch a.State {
				case idempotency.Acquired:
					acquired.Add(1)
					executed.Add(1)
					time.Sleep(100 * time.Millisecond) // hold the key while others wait
					if _, err := tx.Exec(c, "INSERT INTO widget (label) VALUES ('once')"); err != nil {
						return err
					}
					return idempotency.Complete(c, tx, rq, json.RawMessage(`{"ok":true}`), at)
				case idempotency.Replay:
					replayed.Add(1)
				default:
					t.Errorf("unexpected state %v", a.State)
				}
				return nil
			})
			if err != nil {
				t.Errorf("concurrent acquire: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()
	if acquired.Load() != 1 || executed.Load() != 1 || replayed.Load() != 9 {
		t.Fatalf("acquired=%d executed=%d replayed=%d", acquired.Load(), executed.Load(), replayed.Load())
	}
	var widgets int
	_ = alpha.InTx(ctx(t), db.TxOptions{ReadOnly: true}, func(c context.Context, tx db.Tx) error {
		return tx.QueryRow(c, "SELECT count(*) FROM widget").Scan(&widgets)
	})
	if widgets != 1 {
		t.Fatalf("command executed %d times", widgets)
	}
}

// TestConcurrentDifferentRequestsConflict: one wins, every other hash conflicts.
func TestConcurrentDifferentRequestsConflict(t *testing.T) {
	env := pgtest.New(t)
	alpha := env.Runner(t, pgtest.Alpha)
	var acquired, conflicts atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			rq := req("synthetic-key-0005", `{"n":`+strconv.Itoa(i)+`}`)
			err := alpha.InTx(context.Background(), db.TxOptions{}, func(c context.Context, tx db.Tx) error {
				a, err := idempotency.Acquire(c, tx, rq, at)
				if err != nil {
					return err
				}
				switch a.State {
				case idempotency.Acquired:
					acquired.Add(1)
					time.Sleep(50 * time.Millisecond)
					return idempotency.Complete(c, tx, rq, nil, at)
				case idempotency.Conflict:
					conflicts.Add(1)
				default:
					t.Errorf("unexpected state %v", a.State)
				}
				return nil
			})
			if err != nil {
				t.Errorf("acquire: %v", err)
			}
		}(i)
	}
	close(start)
	wg.Wait()
	if acquired.Load() != 1 || conflicts.Load() != 5 {
		t.Fatalf("acquired=%d conflicts=%d", acquired.Load(), conflicts.Load())
	}
}
