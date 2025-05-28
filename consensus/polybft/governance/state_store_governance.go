package governance

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/0xPolygon/polygon-edge/chain"
	bolt "go.etcd.io/bbolt"
)

var (
	clientConfigBucket = []byte("clientConfig")
	clientConfigKey    = []byte("clientConfigKey")

	errClientConfigNotFound = errors.New("client (polybft) config not found in db")
)

type eventsRaw [][]byte

// Bolt db schema:
//
// governance events/
// |--> epoch -> slice of contractsapi.EventAbi
// |--> fork name hash -> block from which is active
// |--> clientConfigKey -> *PolyBFTConfig
type GovernanceStore struct {
	db *bolt.DB
}

func newGovernanceStore(db *bolt.DB, dbTx *bolt.Tx) (*GovernanceStore, error) {
	store := &GovernanceStore{db: db}

	initFn := func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(clientConfigBucket); err != nil {
			return fmt.Errorf("failed to create bucket=%s: %w",
				string(clientConfigBucket), err)
		}

		return nil
	}

	var err error

	if dbTx == nil {
		err = db.Update(initFn)
	} else {
		err = initFn(dbTx)
	}

	return store, err
}

// insertClientConfig inserts client (polybft) config to bolt db
func (g *GovernanceStore) insertClientConfig(config *chain.Params, dbTx *bolt.Tx) error {
	insertFn := func(tx *bolt.Tx) error {
		raw, err := json.Marshal(config)
		if err != nil {
			return err
		}

		return tx.Bucket(clientConfigBucket).Put(clientConfigKey, raw)
	}

	if dbTx == nil {
		return g.db.Update(func(tx *bolt.Tx) error {
			return insertFn(tx)
		})
	}

	return insertFn(dbTx)
}

// getClientConfig returns client (polybft) config from bolt db
func (g *GovernanceStore) getClientConfig(dbTx *bolt.Tx) (*chain.Params, error) {
	var (
		config *chain.Params
		err    error
	)

	getFn := func(tx *bolt.Tx) error {
		val := tx.Bucket(clientConfigBucket).Get(clientConfigKey)
		if val == nil {
			return errClientConfigNotFound
		}

		return json.Unmarshal(val, &config)
	}

	if dbTx == nil {
		err = g.db.View(func(tx *bolt.Tx) error {
			return getFn(tx)
		})
	} else {
		err = getFn(dbTx)
	}

	return config, err
}
