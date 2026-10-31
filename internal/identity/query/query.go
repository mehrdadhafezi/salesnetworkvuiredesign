// Package query is the identity read service. Every call runs in a
// read-only, identity-owned transaction, so a read can never write
// (PostgreSQL rejects any write in a READ ONLY transaction). It implements
// the published identity resolver and grant reader.
package query

import (
	"context"
	"errors"
	"fmt"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/store"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pagination"
)

// ErrInvalidQuery reports a filter, sort, cursor or limit outside the allowlist.
var ErrInvalidQuery = errors.New("identity/query: invalid query")

// Service reads the identity model.
type Service struct{ runner *db.Runner }

var (
	_ api.UserResolver = (*Service)(nil)
	_ api.GrantReader  = (*Service)(nil)
)

// New returns the read service over an identity-owned runner.
func New(runner *db.Runner) *Service { return &Service{runner: runner} }

func (s *Service) read(ctx context.Context, fn func(ctx context.Context, tx db.Tx) error) error {
	return s.runner.InTx(ctx, db.TxOptions{ReadOnly: true, Isolation: db.RepeatableRead}, fn)
}

// ListParams are the allowlisted list inputs.
type ListParams struct {
	Filter store.UserFilter
	Sort   string
	Cursor *string
	Limit  *int
}

// Page is one page of users.
type Page struct {
	Users      []store.User
	NextCursor *string
}

// ListUsers returns one page.
func (s *Service) ListUsers(ctx context.Context, p ListParams) (Page, error) {
	if p.Sort == "" {
		p.Sort = "display_name"
	}
	sort, ok := store.UserSorts[p.Sort]
	if !ok {
		return Page{}, fmt.Errorf("%w: sort %q", ErrInvalidQuery, p.Sort)
	}
	if p.Filter.Presence != nil && *p.Filter.Presence != "PRESENT" && *p.Filter.Presence != "ABSENT_IN_LATEST" {
		return Page{}, fmt.Errorf("%w: presence", ErrInvalidQuery)
	}
	if p.Filter.Query != nil && (len(*p.Filter.Query) == 0 || len(*p.Filter.Query) > 100) {
		return Page{}, fmt.Errorf("%w: q length", ErrInvalidQuery)
	}
	limit, err := pagination.Limit(p.Limit)
	if err != nil {
		return Page{}, fmt.Errorf("%w: %v", ErrInvalidQuery, err)
	}
	var afterV, afterID *string
	if p.Cursor != nil {
		c, err := pagination.Decode(*p.Cursor, p.Sort)
		if err != nil {
			return Page{}, fmt.Errorf("%w: %v", ErrInvalidQuery, err)
		}
		afterV, afterID = &c.Value, &c.ID
	}
	var rows []store.User
	if err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		rows, err = store.ListUsers(ctx, tx, p.Filter, sort, afterV, afterID, limit)
		return err
	}); err != nil {
		return Page{}, err
	}
	page := Page{Users: rows}
	if len(rows) > limit {
		page.Users = rows[:limit]
		last := page.Users[limit-1]
		v := ""
		if last.DisplayName != nil {
			v = *last.DisplayName
		}
		cur, err := pagination.Encode(pagination.Cursor{Sort: p.Sort, Value: v, ID: last.UserID})
		if err != nil {
			return Page{}, err
		}
		page.NextCursor = &cur
	}
	return page, nil
}

// GetUser returns one account.
func (s *Service) GetUser(ctx context.Context, id string) (store.User, bool, error) {
	var (
		u  store.User
		ok bool
	)
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		u, ok, err = store.GetUser(ctx, tx, id)
		return err
	})
	return u, ok, err
}

// ResolveUsers implements api.UserResolver.
func (s *Service) ResolveUsers(ctx context.Context, system string, keys []string) (map[string]string, error) {
	out := map[string]string{}
	if len(keys) == 0 {
		return out, nil
	}
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		out, err = store.UserIDsByKey(ctx, tx, system, keys)
		return err
	})
	return out, err
}

// Grants implements api.GrantReader.
func (s *Service) Grants(ctx context.Context) ([]api.Grant, error) {
	var out []api.Grant
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		out, err = store.Grants(ctx, tx)
		return err
	})
	return out, err
}

// Counts returns reconciliation counts.
func (s *Service) Counts(ctx context.Context) (map[string]int64, error) {
	var out map[string]int64
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		out, err = store.Counts(ctx, tx)
		return err
	})
	return out, err
}

// Vocabulary returns the stored capability and role reference data.
func (s *Service) Vocabulary(ctx context.Context) (map[string]string, map[string][]string, error) {
	var (
		caps    map[string]string
		bundles map[string][]string
	)
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		caps, bundles, err = store.Vocabulary(ctx, tx)
		return err
	})
	return caps, bundles, err
}
