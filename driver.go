package tokenfile

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	. "github.com/infrago/base"
	"github.com/infrago/infra"
	"github.com/infrago/token"
	"github.com/tidwall/buntdb"
)

type fileDriver struct {
	mutex sync.Mutex
	db    *buntdb.DB
	path  string
}

func init() {
	token.RegisterDriver("file", &fileDriver{
		path: "store/token.db",
	})
}

func (d *fileDriver) Configure(setting Map) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if v, ok := setting["file_path"].(string); ok && strings.TrimSpace(v) != "" {
		d.path = strings.TrimSpace(v)
	}
	if v, ok := setting["path"].(string); ok && strings.TrimSpace(v) != "" {
		d.path = strings.TrimSpace(v)
	}
	if v, ok := setting["file_dir"].(string); ok && strings.TrimSpace(v) != "" {
		d.path = filepath.Join(strings.TrimSpace(v), "token.db")
	}
	if v, ok := setting["dir"].(string); ok && strings.TrimSpace(v) != "" {
		d.path = filepath.Join(strings.TrimSpace(v), "token.db")
	}
}

func (d *fileDriver) Open() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if d.db != nil {
		return nil
	}
	if dir := filepath.Dir(d.path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	db, err := buntdb.Open(d.path)
	if err != nil {
		return err
	}
	d.db = db
	return nil
}

func (d *fileDriver) Close() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if d.db == nil {
		return nil
	}
	err := d.db.Close()
	d.db = nil
	return err
}

func (d *fileDriver) SavePayload(_ *infra.Meta, tokenID string, payload Map, exp int64) error {
	tokenID = strings.TrimSpace(tokenID)
	if tokenID == "" {
		return nil
	}
	db, err := d.ensureDB()
	if err != nil {
		return err
	}
	bts, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(d.keyPayload(tokenID), string(bts), d.setOptions(exp))
		return err
	})
}

func (d *fileDriver) LoadPayload(_ *infra.Meta, tokenID string) (Map, bool, error) {
	tokenID = strings.TrimSpace(tokenID)
	if tokenID == "" {
		return nil, false, nil
	}
	db, err := d.ensureDB()
	if err != nil {
		return nil, false, err
	}
	var raw string
	err = db.View(func(tx *buntdb.Tx) error {
		val, err := tx.Get(d.keyPayload(tokenID))
		if err != nil {
			return err
		}
		raw = val
		return nil
	})
	if err == buntdb.ErrNotFound {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	out := Map{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, false, err
	}
	return out, true, nil
}

func (d *fileDriver) DeletePayload(_ *infra.Meta, tokenID string) error {
	tokenID = strings.TrimSpace(tokenID)
	if tokenID == "" {
		return nil
	}
	db, err := d.ensureDB()
	if err != nil {
		return err
	}
	return db.Update(func(tx *buntdb.Tx) error {
		_, err := tx.Delete(d.keyPayload(tokenID))
		if err == buntdb.ErrNotFound {
			return nil
		}
		return err
	})
}

func (d *fileDriver) RevokeToken(_ *infra.Meta, token string, exp int64) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	db, err := d.ensureDB()
	if err != nil {
		return err
	}
	return db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(d.keyRevokeToken(token), "1", d.setOptions(exp))
		return err
	})
}

func (d *fileDriver) RevokeTokenID(_ *infra.Meta, tokenID string, exp int64) error {
	tokenID = strings.TrimSpace(tokenID)
	if tokenID == "" {
		return nil
	}
	db, err := d.ensureDB()
	if err != nil {
		return err
	}
	return db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(d.keyRevokeTokenID(tokenID), "1", d.setOptions(exp))
		return err
	})
}

func (d *fileDriver) RevokedToken(_ *infra.Meta, token string) (bool, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return false, nil
	}
	db, err := d.ensureDB()
	if err != nil {
		return false, err
	}
	err = db.View(func(tx *buntdb.Tx) error {
		_, err := tx.Get(d.keyRevokeToken(token))
		return err
	})
	if err == buntdb.ErrNotFound {
		return false, nil
	}
	return err == nil, err
}

func (d *fileDriver) RevokedTokenID(_ *infra.Meta, tokenID string) (bool, error) {
	tokenID = strings.TrimSpace(tokenID)
	if tokenID == "" {
		return false, nil
	}
	db, err := d.ensureDB()
	if err != nil {
		return false, err
	}
	err = db.View(func(tx *buntdb.Tx) error {
		_, err := tx.Get(d.keyRevokeTokenID(tokenID))
		return err
	})
	if err == buntdb.ErrNotFound {
		return false, nil
	}
	return err == nil, err
}

func (d *fileDriver) ensureDB() (*buntdb.DB, error) {
	d.mutex.Lock()
	db := d.db
	d.mutex.Unlock()
	if db != nil {
		return db, nil
	}
	if err := d.Open(); err != nil {
		return nil, err
	}
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.db, nil
}

func (d *fileDriver) setOptions(exp int64) *buntdb.SetOptions {
	if exp <= 0 {
		return nil
	}
	ttl := time.Until(time.Unix(exp, 0))
	if ttl <= 0 {
		ttl = time.Second
	}
	return &buntdb.SetOptions{Expires: true, TTL: ttl}
}

func (d *fileDriver) keyPayload(tokenID string) string {
	return "payload:" + tokenID
}

func (d *fileDriver) keyRevokeToken(token string) string {
	return "revoke:token:" + hashToken(token)
}

func (d *fileDriver) keyRevokeTokenID(tokenID string) string {
	return "revoke:tokenid:" + tokenID
}

func hashToken(token string) string {
	sum := sha1.Sum([]byte(token))
	return hex.EncodeToString(sum[:])
}

func parseInt(v string, def int) int {
	v = strings.TrimSpace(v)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
