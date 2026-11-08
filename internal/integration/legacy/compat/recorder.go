package compat

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
)

// DBRecorder appends exchanges to integration.writeback_exchange.
type DBRecorder struct{ runner *db.Runner }

// NewDBRecorder returns a recorder over an integration-owned runner.
func NewDBRecorder(r *db.Runner) (*DBRecorder, error) {
	if r == nil || r.Module() != integration.Module {
		return nil, errors.New("compat: recorder runner must be owned by integration")
	}
	return &DBRecorder{runner: r}, nil
}

// Record implements Recorder.
func (d *DBRecorder) Record(ctx context.Context, e Exchange) error {
	id, err := ids.NewUUID()
	if err != nil {
		return err
	}
	var resp []byte
	if e.ResponseSHA256 != nil {
		resp = e.ResponseSHA256[:]
	}
	var status *int
	if e.HTTPStatus != 0 {
		status = &e.HTTPStatus
	}
	var outcome, corr *string
	if e.Outcome != "" {
		outcome = &e.Outcome
	}
	if e.CorrelationID != "" {
		corr = &e.CorrelationID
	}
	return d.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO integration.writeback_exchange (exchange_id, request_token, operation, contract, correlation_id,
			request_sha256, response_sha256, http_status, verdict, outcome, started_at, finished_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`, id, e.RequestToken, e.Operation, e.Contract, corr, e.RequestSHA256[:],
			resp, status, string(e.Verdict), outcome, e.StartedAt, e.FinishedAt)
		return err
	})
}

// HMACCredentials signs requests and verifies responses with two distinct
// HMAC-SHA256 keys (client and server). The key bytes are supplied by the
// caller (a secret file or manager) and are never logged or exposed.
type HMACCredentials struct {
	keyID             string
	clientKey, server []byte
}

// MinKeyBytes is the minimum HMAC key length.
const MinKeyBytes = 32

// NewHMACCredentials validates and returns credentials.
func NewHMACCredentials(keyID string, clientKey, serverKey []byte) (*HMACCredentials, error) {
	if !tokenPattern.MatchString(keyID) || len(clientKey) < MinKeyBytes || len(serverKey) < MinKeyBytes || hmac.Equal(clientKey, serverKey) {
		return nil, errors.New("compat: credentials need a key id and two distinct keys of at least 32 bytes")
	}
	return &HMACCredentials{keyID: keyID, clientKey: append([]byte(nil), clientKey...), server: append([]byte(nil), serverKey...)}, nil
}

// KeyID implements Credentials.
func (h *HMACCredentials) KeyID() string { return h.keyID }

func mac(key, body []byte) string {
	m := hmac.New(sha256.New, key)
	m.Write(body)
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

// Sign implements Credentials.
func (h *HMACCredentials) Sign(body []byte) (string, error) { return mac(h.clientKey, body), nil }

// VerifyResponse implements Credentials (constant-time).
func (h *HMACCredentials) VerifyResponse(body []byte, signature string) bool {
	return signature != "" && hmac.Equal([]byte(mac(h.server, body)), []byte(signature))
}

// String never reveals key material.
func (h *HMACCredentials) String() string { return "compat.HMACCredentials{key_id:" + h.keyID + "}" }
