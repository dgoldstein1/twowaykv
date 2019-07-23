package server

import (
	badger "github.com/dgraph-io/badger"
	"os"
	"strconv"
)

// connects to both keyToValue and valueToKey store
func ConnectToDb() (*badger.DB, *badger.DB, error) {
	dir := os.Getenv("GRAPH_DB_STORE_DIR")
	// setup db properties
	options := badger.Options{
		Dir:                     dir + "/keysToValues",
		ValueDir:                dir + "/keysToValues",
		LevelOneSize:            256 << 20,
		LevelSizeMultiplier:     10,
		MaxLevels:               7,
		MaxTableSize:            64 << 20,
		NumCompactors:           2, // Compactions can be expensive. Only run 2.
		NumLevelZeroTables:      5,
		NumLevelZeroTablesStall: 10,
		NumMemtables:            5,
		SyncWrites:              true,
		NumVersionsToKeep:       1,
		ValueLogFileSize:        1<<30 - 1,
		ValueLogMaxEntries:      1000000,
		ValueThreshold:          32,
		Truncate:                false,
	}
	// create keys => values DB
	keysToValuesDB, err := badger.Open(options)
	if err != nil {
		return nil, nil, err
	}
	// create values => keys DB
	options.Dir = dir + "/valuesToKeys"
	options.ValueDir = dir + "/valuesToKeys"
	valuesToKeysDB, err := badger.Open(options)
	return keysToValuesDB, valuesToKeysDB, err
}

// writes entry to both dbs
func WriteEntry(k2v *badger.DB, v2k *badger.DB, e Entry) error {
	// cast value as int -> byte(string)
	val := []byte(strconv.Itoa(e.Value))
	key := []byte(e.Key)
	// update k2v with k : v
	err := k2v.Update(func(txn *badger.Txn) error {
		return txn.Set(key, val)
	})
	// write v:k
	err = v2k.Update(func(txn *badger.Txn) error {
		return txn.Set(val, key)
	})
	return err
}

// retrieves entry using either key or value
func GetEntry(k2v *badger.DB, v2k *badger.DB, e Entry) (Entry, error) {
	if e.Key == "" {
		// lookup on value
		err := v2k.View(func(txn *badger.Txn) error {
			val := []byte(strconv.Itoa(e.Value))
			item, err := txn.Get(val)
			if err != nil {
				return err
			}
			key, err := item.Value()
			e.Key = string(key)
			return err
		})
		if err != nil {
			return Entry{}, err
		}
		// else lookup on key
	} else {
		err := k2v.View(func(txn *badger.Txn) error {
			item, err := txn.Get([]byte(e.Key))
			if err != nil {
				return err
			}
			v, err := item.Value()
			// cast as int
			value, _ := strconv.Atoi(string(v))
			e.Value = value
			return err
		})
		if err != nil {
			return e, err
		}
	}

	return e, nil
}