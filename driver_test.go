package tokenfile

import (
	"path/filepath"
	"testing"
	"time"

	. "github.com/infrago/base"
)

func newTestFileDriver(t *testing.T) *fileDriver {
	t.Helper()
	return &fileDriver{path: filepath.Join(t.TempDir(), "token.db")}
}

func TestExpiredPayloadIsNotStored(t *testing.T) {
	driver := newTestFileDriver(t)
	expired := time.Now().Add(-time.Second).Unix()

	if err := driver.SavePayload("tid-expired", Map{"uid": "u1"}, expired); err != nil {
		t.Fatalf("save payload: %v", err)
	}
	if _, ok, err := driver.LoadPayload("tid-expired"); err != nil || ok {
		t.Fatalf("expected expired payload to be absent, ok=%v err=%v", ok, err)
	}
}

func TestCurrentSecondPayloadIsStillReadable(t *testing.T) {
	driver := newTestFileDriver(t)
	exp := time.Now().Unix()

	if err := driver.SavePayload("tid-current", Map{"uid": "u1"}, exp); err != nil {
		t.Fatalf("save payload: %v", err)
	}
	if payload, ok, err := driver.LoadPayload("tid-current"); err != nil || !ok || payload["uid"] != "u1" {
		t.Fatalf("expected current-second payload, payload=%v ok=%v err=%v", payload, ok, err)
	}
}

func TestExpiredRevokeIsNotStored(t *testing.T) {
	driver := newTestFileDriver(t)
	expired := time.Now().Add(-time.Second).Unix()

	if err := driver.RevokeTokenID("tid-expired", expired); err != nil {
		t.Fatalf("revoke token id: %v", err)
	}
	if ok, err := driver.RevokedTokenID("tid-expired"); err != nil || ok {
		t.Fatalf("expected expired revoke to be absent, ok=%v err=%v", ok, err)
	}
}

func TestCurrentSecondRevokeIsStillReadable(t *testing.T) {
	driver := newTestFileDriver(t)
	exp := time.Now().Unix()

	if err := driver.RevokeTokenID("tid-current", exp); err != nil {
		t.Fatalf("revoke token id: %v", err)
	}
	if ok, err := driver.RevokedTokenID("tid-current"); err != nil || !ok {
		t.Fatalf("expected current-second revoke, ok=%v err=%v", ok, err)
	}
}

func TestConfigurePayloadCodec(t *testing.T) {
	driver := newTestFileDriver(t)

	driver.Configure(Map{"codec": "ignored", "store_codec": "custom"})
	if driver.payloadCodec() != "custom" {
		t.Fatalf("expected custom payload codec, got %q", driver.payloadCodec())
	}
}
