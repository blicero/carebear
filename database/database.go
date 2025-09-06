// /home/krylon/go/src/github.com/blicero/carebear/database/database.go
// -*- mode: go; coding: utf-8; -*-
// Created on 05. 07. 2025 by Benjamin Walkenhorst
// (c) 2025 Benjamin Walkenhorst
// Time-stamp: <2025-09-06 15:54:30 krylon>

package database

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/blicero/carebear/common"
	"github.com/blicero/carebear/database/query"
	"github.com/blicero/carebear/logdomain"
	"github.com/blicero/carebear/model"
	"github.com/blicero/carebear/model/info"
	"github.com/blicero/krylib"
	_ "github.com/mattn/go-sqlite3" // Import the database driver
)

var (
	openLock sync.Mutex
	idCnt    int64
)

// ErrTxInProgress indicates that an attempt to initiate a transaction failed
// because there is already one in progress.
var ErrTxInProgress = errors.New("A Transaction is already in progress")

// ErrNoTxInProgress indicates that an attempt was made to finish a
// transaction when none was active.
var ErrNoTxInProgress = errors.New("There is no transaction in progress")

// ErrEmptyUpdate indicates that an update operation would not change any
// values.
var ErrEmptyUpdate = errors.New("Update operation does not change any values")

// ErrInvalidValue indicates that one or more parameters passed to a method
// had values that are invalid for that operation.
var ErrInvalidValue = errors.New("Invalid value for parameter")

// ErrObjectNotFound indicates that an Object was not found in the database.
var ErrObjectNotFound = errors.New("object was not found in database")

// ErrInvalidSavepoint is returned when a user of the Database uses an unkown
// (or expired) savepoint name.
var ErrInvalidSavepoint = errors.New("that save point does not exist")

// If a query returns an error and the error text is matched by this regex, we
// consider the error as transient and try again after a short delay.
var retryPat = regexp.MustCompile("(?i)database is (?:locked|busy)")

// worthARetry returns true if an error returned from the database
// is matched by the retryPat regex.
func worthARetry(e error) bool {
	return retryPat.MatchString(e.Error())
} // func worthARetry(e error) bool

// retryDelay is the amount of time we wait before we repeat a database
// operation that failed due to a transient error.
const retryDelay = 25 * time.Millisecond

func waitForRetry() {
	time.Sleep(retryDelay)
} // func waitForRetry()

// Database is the storage backend.
//
// It is not safe to share a Database instance between goroutines, however
// opening multiple connections to the same Database is safe.
type Database struct {
	id            int64
	db            *sql.DB
	tx            *sql.Tx
	log           *log.Logger
	path          string
	spNameCounter int
	spNameCache   map[string]string
	queries       map[query.ID]*sql.Stmt
}

// Open opens a Database. If the database specified by the path does not exist,
// yet, it is created and initialized.
func Open(path string) (*Database, error) {
	var (
		err      error
		dbExists bool
		db       = &Database{
			path:          path,
			spNameCounter: 1,
			spNameCache:   make(map[string]string),
			queries:       make(map[query.ID]*sql.Stmt),
		}
	)

	openLock.Lock()
	defer openLock.Unlock()
	idCnt++
	db.id = idCnt

	if db.log, err = common.GetLogger(logdomain.Database); err != nil {
		return nil, err
	} else if common.Debug {
		db.log.Printf("[DEBUG] Open database %s\n", path)
	}

	var connstring = fmt.Sprintf("%s?_locking=NORMAL&_journal=WAL&_fk=1&recursive_triggers=0",
		path)

	if dbExists, err = krylib.Fexists(path); err != nil {
		db.log.Printf("[ERROR] Failed to check if %s already exists: %s\n",
			path,
			err.Error())
		return nil, err
	} else if db.db, err = sql.Open("sqlite3", connstring); err != nil {
		db.log.Printf("[ERROR] Failed to open %s: %s\n",
			path,
			err.Error())
		return nil, err
	}

	if !dbExists {
		if err = db.initialize(); err != nil {
			var e2 error
			if e2 = db.db.Close(); e2 != nil {
				db.log.Printf("[CRITICAL] Failed to close database: %s\n",
					e2.Error())
				return nil, e2
			} else if e2 = os.Remove(path); e2 != nil {
				db.log.Printf("[CRITICAL] Failed to remove database file %s: %s\n",
					db.path,
					e2.Error())
			}
			return nil, err
		}
		db.log.Printf("[INFO] Database at %s has been initialized\n",
			path)
	}

	return db, nil
} // func Open(path string) (*Database, error)

func (db *Database) initialize() error {
	var err error
	var tx *sql.Tx

	if common.Debug {
		db.log.Printf("[DEBUG] Initialize fresh database at %s\n",
			db.path)
	}

	if tx, err = db.db.Begin(); err != nil {
		db.log.Printf("[ERROR] Cannot begin transaction: %s\n",
			err.Error())
		return err
	}

	for _, q := range qinit {
		db.log.Printf("[TRACE] Execute init query:\n%s\n",
			q)
		if _, err = tx.Exec(q); err != nil {
			db.log.Printf("[ERROR] Cannot execute init query: %s\n%s\n",
				err.Error(),
				q)
			if rbErr := tx.Rollback(); rbErr != nil {
				db.log.Printf("[CANTHAPPEN] Cannot rollback transaction: %s\n",
					rbErr.Error())
				return rbErr
			}
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		db.log.Printf("[CANTHAPPEN] Failed to commit init transaction: %s\n",
			err.Error())
		return err
	}

	return nil
} // func (db *Database) initialize() error

// Close closes the database.
// If there is a pending transaction, it is rolled back.
func (db *Database) Close() error {
	// I wonder if would make more snese to panic() if something goes wrong

	var err error

	if db.tx != nil {
		if err = db.tx.Rollback(); err != nil {
			db.log.Printf("[CRITICAL] Cannot roll back pending transaction: %s\n",
				err.Error())
			return err
		}
		db.tx = nil
	}

	for key, stmt := range db.queries {
		if err = stmt.Close(); err != nil {
			db.log.Printf("[CRITICAL] Cannot close statement handle %s: %s\n",
				key,
				err.Error())
			return err
		}
		delete(db.queries, key)
	}

	if err = db.db.Close(); err != nil {
		db.log.Printf("[CRITICAL] Cannot close database: %s\n",
			err.Error())
	}

	db.db = nil
	return nil
} // func (db *Database) Close() error

func (db *Database) getQuery(id query.ID) (*sql.Stmt, error) {
	var (
		stmt  *sql.Stmt
		found bool
		err   error
	)

	if stmt, found = db.queries[id]; found {
		return stmt, nil
	} else if _, found = qdb[id]; !found {
		return nil, fmt.Errorf("Unknown Query %d",
			id)
	}

	db.log.Printf("[TRACE] Prepare query %s\n", id)

PREPARE_QUERY:
	if stmt, err = db.db.Prepare(qdb[id]); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto PREPARE_QUERY
		}

		db.log.Printf("[ERROR] Cannot parse query %s: %s\n%s\n",
			id,
			err.Error(),
			qdb[id])
		return nil, err
	}

	db.queries[id] = stmt
	return stmt, nil
} // func (db *Database) getQuery(query.ID) (*sql.Stmt, error)

func (db *Database) resetSPNamespace() {
	db.spNameCounter = 1
	db.spNameCache = make(map[string]string)
} // func (db *Database) resetSPNamespace()

func (db *Database) generateSPName(name string) string {
	var spname = fmt.Sprintf("Savepoint%05d",
		db.spNameCounter)

	db.spNameCache[name] = spname
	db.spNameCounter++
	return spname
} // func (db *Database) generateSPName() string

// PerformMaintenance performs some maintenance operations on the database.
// It cannot be called while a transaction is in progress and will block
// pretty much all access to the database while it is running.
func (db *Database) PerformMaintenance() error {
	var mQueries = []string{
		"PRAGMA wal_checkpoint(TRUNCATE)",
		"VACUUM",
		"REINDEX",
		"ANALYZE",
	}
	var err error

	if db.tx != nil {
		return ErrTxInProgress
	}

	for _, q := range mQueries {
		if _, err = db.db.Exec(q); err != nil {
			db.log.Printf("[ERROR] Failed to execute %s: %s\n",
				q,
				err.Error())
		}
	}

	return nil
} // func (db *Database) PerformMaintenance() error

// Begin begins an explicit database transaction.
// Only one transaction can be in progress at once, attempting to start one,
// while another transaction is already in progress will yield ErrTxInProgress.
func (db *Database) Begin() error {
	var err error

	db.log.Printf("[DEBUG] Database#%d Begin Transaction\n",
		db.id)

	if db.tx != nil {
		return ErrTxInProgress
	}

BEGIN_TX:
	for db.tx == nil {
		if db.tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				continue BEGIN_TX
			} else {
				db.log.Printf("[ERROR] Failed to start transaction: %s\n",
					err.Error())
				return err
			}
		}
	}

	db.resetSPNamespace()

	return nil
} // func (db *Database) Begin() error

// SavepointCreate creates a savepoint with the given name.
//
// Savepoints only make sense within a running transaction, and just like
// with explicit transactions, managing them is the responsibility of the
// user of the Database.
//
// Creating a savepoint without a surrounding transaction is not allowed,
// even though SQLite allows it.
//
// For details on how Savepoints work, check the excellent SQLite
// documentation, but here's a quick guide:
//
// Savepoints are kind-of-like transactions within a transaction: One
// can create a savepoint, make some changes to the database, and roll
// back to that savepoint, discarding all changes made between
// creating the savepoint and rolling back to it. Savepoints can be
// quite useful, but there are a few things to keep in mind:
//
//   - Savepoints exist within a transaction. When the surrounding transaction
//     is finished, all savepoints created within that transaction cease to exist,
//     no matter if the transaction is commited or rolled back.
//
//   - When the database is recovered after being interrupted during a
//     transaction, e.g. by a power outage, the entire transaction is rolled back,
//     including all savepoints that might exist.
//
//   - When a savepoint is released, nothing changes in the state of the
//     surrounding transaction. That means rolling back the surrounding
//     transaction rolls back the entire transaction, regardless of any
//     savepoints within.
//
//   - Savepoints do not nest. Releasing a savepoint releases it and *all*
//     existing savepoints that have been created before it. Rolling back to a
//     savepoint removes that savepoint and all savepoints created after it.
func (db *Database) SavepointCreate(name string) error {
	var err error

	db.log.Printf("[DEBUG] SavepointCreate(%s)\n",
		name)

	if db.tx == nil {
		return ErrNoTxInProgress
	}

SAVEPOINT:
	// It appears that the SAVEPOINT statement does not support placeholders.
	// But I do want to used named savepoints.
	// And I do want to use the given name so that no SQL injection
	// becomes possible.
	// It would be nice if the database package or at least the SQLite
	// driver offered a way to escape the string properly.
	// One possible solution would be to use names generated by the
	// Database instead of user-defined names.
	//
	// But then I need a way to use the Database-generated name
	// in rolling back and releasing the savepoint.
	// I *could* use the names strictly inside the Database, store them in
	// a map or something and hand out a key to that name to the user.
	// Since savepoint only exist within one transaction, I could even
	// re-use names from one transaction to the next.
	//
	// Ha! I could accept arbitrary names from the user, generate a
	// clean name, and store these in a map. That way the user can
	// still choose names that are outwardly visible, but they do
	// not touch the Database itself.
	//
	//if _, err = db.tx.Exec("SAVEPOINT ?", name); err != nil {
	// if _, err = db.tx.Exec("SAVEPOINT " + name); err != nil {
	// 	if worthARetry(err) {
	// 		waitForRetry()
	// 		goto SAVEPOINT
	// 	}

	// 	db.log.Printf("[ERROR] Failed to create savepoint %s: %s\n",
	// 		name,
	// 		err.Error())
	// }

	var internalName = db.generateSPName(name)

	var spQuery = "SAVEPOINT " + internalName

	if _, err = db.tx.Exec(spQuery); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto SAVEPOINT
		}

		db.log.Printf("[ERROR] Failed to create savepoint %s: %s\n",
			name,
			err.Error())
	}

	return err
} // func (db *Database) SavepointCreate(name string) error

// SavepointRelease releases the Savepoint with the given name, and all
// Savepoints created before the one being release.
func (db *Database) SavepointRelease(name string) error {
	var (
		err                   error
		internalName, spQuery string
		validName             bool
	)

	db.log.Printf("[DEBUG] SavepointRelease(%s)\n",
		name)

	if db.tx != nil {
		return ErrNoTxInProgress
	}

	if internalName, validName = db.spNameCache[name]; !validName {
		db.log.Printf("[ERROR] Attempt to release unknown Savepoint %q\n",
			name)
		return ErrInvalidSavepoint
	}

	db.log.Printf("[DEBUG] Release Savepoint %q (%q)",
		name,
		db.spNameCache[name])

	spQuery = "RELEASE SAVEPOINT " + internalName

SAVEPOINT:
	if _, err = db.tx.Exec(spQuery); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto SAVEPOINT
		}

		db.log.Printf("[ERROR] Failed to release savepoint %s: %s\n",
			name,
			err.Error())
	} else {
		delete(db.spNameCache, internalName)
	}

	return err
} // func (db *Database) SavepointRelease(name string) error

// SavepointRollback rolls back the running transaction to the given savepoint.
func (db *Database) SavepointRollback(name string) error {
	var (
		err                   error
		internalName, spQuery string
		validName             bool
	)

	db.log.Printf("[DEBUG] SavepointRollback(%s)\n",
		name)

	if db.tx != nil {
		return ErrNoTxInProgress
	}

	if internalName, validName = db.spNameCache[name]; !validName {
		return ErrInvalidSavepoint
	}

	spQuery = "ROLLBACK TO SAVEPOINT " + internalName

SAVEPOINT:
	if _, err = db.tx.Exec(spQuery); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto SAVEPOINT
		}

		db.log.Printf("[ERROR] Failed to create savepoint %s: %s\n",
			name,
			err.Error())
	}

	delete(db.spNameCache, name)
	return err
} // func (db *Database) SavepointRollback(name string) error

// Rollback terminates a pending transaction, undoing any changes to the
// database made during that transaction.
// If no transaction is active, it returns ErrNoTxInProgress
func (db *Database) Rollback() error {
	var err error

	db.log.Printf("[DEBUG] Database#%d Roll back Transaction\n",
		db.id)

	if db.tx == nil {
		return ErrNoTxInProgress
	} else if err = db.tx.Rollback(); err != nil {
		return fmt.Errorf("Cannot roll back database transaction: %s",
			err.Error())
	}

	db.tx = nil
	db.resetSPNamespace()

	return nil
} // func (db *Database) Rollback() error

// Commit ends the active transaction, making any changes made during that
// transaction permanent and visible to other connections.
// If no transaction is active, it returns ErrNoTxInProgress
func (db *Database) Commit() error {
	var err error

	db.log.Printf("[DEBUG] Database#%d Commit Transaction\n",
		db.id)

	if db.tx == nil {
		return ErrNoTxInProgress
	} else if err = db.tx.Commit(); err != nil {
		return fmt.Errorf("Cannot commit transaction: %s",
			err.Error())
	}

	db.resetSPNamespace()
	db.tx = nil
	return nil
} // func (db *Database) Commit() error

// NetworkAdd adds a Network to the Database.
func (db *Database) NetworkAdd(n *model.Network) error {
	const qid query.ID = query.NetworkAdd
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Failed to prepare query %s: %s\n",
			qid,
			err.Error())
		panic(err)
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(n.Addr.String(), n.Description); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("Cannot add Network %s to database: %w",
				n.Addr,
				err)
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	} else {
		var id int64

		defer rows.Close()

		if !rows.Next() {
			// CANTHAPPEN
			db.log.Printf("[ERROR] Query %s did not return a value\n",
				qid)
			return fmt.Errorf("Query %s did not return a value", qid)
		} else if err = rows.Scan(&id); err != nil {
			var ex = fmt.Errorf("Failed to get ID for newly added host %s: %w",
				n.Addr,
				err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return ex
		}

		n.ID = id
		return nil
	}
} // func (db *Database) NetworkAdd(n *model.Network) error

// NetworkUpdateScanStamp sets a Network's LastScan timestamp in the Database.
func (db *Database) NetworkUpdateScanStamp(n *model.Network, t time.Time) error {
	const qid query.ID = query.NetworkUpdateScanStamp
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Failed to prepare query %s: %s\n",
			qid,
			err.Error())
		panic(err)
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var (
		res         sql.Result
		numAffected int64
	)

EXEC_QUERY:
	if res, err = stmt.Exec(t.Unix(), n.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("Cannot update LastScan timestamp of Network %s (%d): %w",
				n.Addr,
				n.ID,
				err)
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	} else if numAffected, err = res.RowsAffected(); err != nil {
		err = fmt.Errorf("Failed to query query result for number of affected rows: %w",
			err)
		db.log.Printf("[ERROR] %s\n", err.Error())
		return err
	} else if numAffected != 1 {
		db.log.Printf("[ERROR] Update LastScan timestamp of Network %s (%d) affected 0 rows\n",
			n.Addr,
			n.ID)
	} else {
		n.LastScan = t
	}

	return nil
} // func (db *Database) NetworkUpdateScanStamp(n *model.Network) error

// NetworkUpdateDesc updates a Network's Description.
func (db *Database) NetworkUpdateDesc(n *model.Network, desc string) error {
	const qid query.ID = query.NetworkUpdateDesc
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Failed to prepare query %s: %s\n",
			qid,
			err.Error())
		panic(err)
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var (
		res         sql.Result
		numAffected int64
	)

EXEC_QUERY:
	if res, err = stmt.Exec(desc, n.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("Cannot update LastScan timestamp of Network %s (%d): %w",
				n.Addr,
				n.ID,
				err)
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	} else if numAffected, err = res.RowsAffected(); err != nil {
		err = fmt.Errorf("Failed to query query result for number of affected rows: %w",
			err)
		db.log.Printf("[ERROR] %s\n", err.Error())
		return err
	} else if numAffected != 1 {
		db.log.Printf("[ERROR] Update LastScan timestamp of Network %s (%d) affected 0 rows\n",
			n.Addr,
			n.ID)
	} else {
		n.Description = desc
	}

	return nil
} // func (db *Database) NetworkUpdateDesc(n *model.Network, desc string) error

// NetworkGetAll loads all Networks from the Database.
func (db *Database) NetworkGetAll() ([]*model.Network, error) {
	const qid query.ID = query.NetworkGetAll
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec
	var networks = make([]*model.Network, 0)

	for rows.Next() {
		var (
			stamp int64
			addr  string
			n     = new(model.Network)
		)

		if err = rows.Scan(&n.ID, &addr, &n.Description, &stamp); err != nil {
			var ex = fmt.Errorf("Failed to scan row: %w", err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return nil, ex
		} else if _, n.Addr, err = net.ParseCIDR(addr); err != nil {
			var ex = fmt.Errorf("Failed to parse Network address %q: %w",
				addr,
				err)
			return nil, ex
		}

		n.LastScan = time.Unix(stamp, 0)
		networks = append(networks, n)
	}

	return networks, nil
} // func (db *Database) NetworkGetAll() ([]*model.Network, error)

// NetworkGetByID loads the Network with the given ID (if it exists).
func (db *Database) NetworkGetByID(id int64) (*model.Network, error) {
	const qid query.ID = query.NetworkGetByID
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(id); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	if rows.Next() {
		var (
			stamp int64
			addr  string
			n     = &model.Network{ID: id}
		)

		if err = rows.Scan(&addr, &n.Description, &stamp); err != nil {
			var ex = fmt.Errorf("Failed to scan row: %w", err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return nil, ex
		} else if _, n.Addr, err = net.ParseCIDR(addr); err != nil {
			var ex = fmt.Errorf("Cannot parse Network address %q: %w",
				addr,
				err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return nil, ex
		}

		n.LastScan = time.Unix(stamp, 0)
		return n, nil
	}

	return nil, nil
} // func (db *Database) NetworkGetByID(id int64) (*model.Network, error)

// NetworkGetByAddr looks up a Networks by its address.
func (db *Database) NetworkGetByAddr(addr *net.IPNet) (*model.Network, error) {
	const qid query.ID = query.NetworkGetByID
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(addr.String()); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	if rows.Next() {
		var (
			stamp int64
			n     = &model.Network{Addr: addr}
		)

		if err = rows.Scan(&n.ID, &n.Description, &stamp); err != nil {
			var ex = fmt.Errorf("Failed to scan row: %w", err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return nil, ex
		}

		n.LastScan = time.Unix(stamp, 0)
		return n, nil
	}

	return nil, nil
} // func (db *Database) NetworkGetByAddr(addr string) (*model.Network, error)

// NetworkDevCnt returns a map that contains the number of devices per network.
func (db *Database) NetworkDevCnt() (map[int64]int, error) {
	const qid query.ID = query.NetworkDevCnt
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec
	var cntMap = make(map[int64]int)

	for rows.Next() {
		var (
			netID int64
			cnt   int
		)

		if err = rows.Scan(&netID, &cnt); err != nil {
			var ex = fmt.Errorf("Failed to scan data from row: %w", err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return nil, ex
		}

		cntMap[netID] = cnt
	}

	return cntMap, nil
} // func (db *Database) NetworkDevCnt() (map[int64]int, error)

// DeviceAdd registers a new device in the database.
func (db *Database) DeviceAdd(dev *model.Device) error {
	const qid query.ID = query.DeviceAdd
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Failed to prepare query %s: %s\n",
			qid,
			err.Error())
		panic(err)
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(dev.Name, dev.NetID, dev.AddrStr(), dev.BigHead); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("Cannot add Host %s/%s to database: %w",
				dev.Name,
				dev.Addr,
				err)
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	} else {
		var id int64

		defer rows.Close()

		if !rows.Next() {
			// CANTHAPPEN
			db.log.Printf("[ERROR] Query %s did not return a value\n",
				qid)
			return fmt.Errorf("Query %s did not return a value", qid)
		} else if err = rows.Scan(&id); err != nil {
			var ex = fmt.Errorf("Failed to get ID for newly added host %s/%s: %w",
				dev.Name,
				dev.AddrStr(),
				err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return ex
		}

		dev.ID = id
		return nil
	}
} // func (db *Database) DeviceAdd(dev *model.Device) error

// DeviceUpdateLastSeen updates a Device's last_seen timestamp in the database.
func (db *Database) DeviceUpdateLastSeen(dev *model.Device, t time.Time) error {
	const qid query.ID = query.DeviceUpdateLastSeen
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Failed to prepare query %s: %s\n",
			qid,
			err.Error())
		panic(err)
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var (
		res         sql.Result
		numAffected int64
	)

EXEC_QUERY:
	if res, err = stmt.Exec(t.Unix(), dev.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("Cannot update LastSeen timestamp of Device %s (%d): %w",
				dev.Name,
				dev.ID,
				err)
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	} else if numAffected, err = res.RowsAffected(); err != nil {
		err = fmt.Errorf("Failed to query query result for number of affected rows: %w",
			err)
		db.log.Printf("[ERROR] %s\n", err.Error())
		return err
	} else if numAffected != 1 {
		db.log.Printf("[ERROR] Update LastSeen timestamp of Device %s (%d) affected 0 rows\n",
			dev.Name,
			dev.ID)
	} else {
		dev.LastSeen = t
	}

	return nil
} // func (db *Database) DeviceUpdateLastSeen(dev *model.Device, t time.Time) error

// DeviceUpdateOS sets the OS field of a Device.
func (db *Database) DeviceUpdateOS(dev *model.Device, osname string) error {
	const qid query.ID = query.DeviceUpdateOS
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Failed to prepare query %s: %s\n",
			qid,
			err.Error())
		panic(err)
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var (
		res         sql.Result
		numAffected int64
	)

EXEC_QUERY:
	if res, err = stmt.Exec(osname, dev.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("Cannot update LastSeen timestamp of Device %s (%d): %w",
				dev.Name,
				dev.ID,
				err)
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	} else if numAffected, err = res.RowsAffected(); err != nil {
		err = fmt.Errorf("Failed to query query result for number of affected rows: %w",
			err)
		db.log.Printf("[ERROR] %s\n", err.Error())
		return err
	} else if numAffected != 1 {
		db.log.Printf("[ERROR] Update LastSeen timestamp of Device %s (%d) affected 0 rows\n",
			dev.Name,
			dev.ID)
	}

	dev.OS = osname

	return nil
} // func (db *Database) DeviceUpdateOS(dev *model.Device, osname string) error

// DeviceGetAll loads all Devices from the Database.
func (db *Database) DeviceGetAll(bigheadOnly bool) ([]*model.Device, error) {
	const qid query.ID = query.DeviceGetAll
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec
	var devices = make([]*model.Device, 0)

	for rows.Next() {
		var (
			stamp int64
			addr  string
			dev   = new(model.Device)
		)

		if err = rows.Scan(
			&dev.ID,
			&dev.NetID,
			&dev.Name,
			&addr,
			&dev.OS,
			&dev.BigHead,
			&stamp); err != nil {
			var ex = fmt.Errorf("Failed to scan row: %w", err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return nil, ex
		}

		if bigheadOnly && !dev.BigHead {
			continue
		}

		var alist = make([]string, 0, 2)

		if err = json.Unmarshal([]byte(addr), &alist); err != nil {
			var ex = fmt.Errorf("Cannot device addresses for Device %s: %w",
				dev.Name,
				err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return nil, ex
		}

		dev.Addr = make([]net.Addr, len(alist))
		for idx, astr := range alist {
			var ip net.IP

			if ip = net.ParseIP(astr); ip == nil {
				var ex = fmt.Errorf("Cannot parse IP address of Device %s (%d): %q",
					dev.Name,
					dev.ID,
					astr)
				return nil, ex
			}

			var addr = &net.IPAddr{IP: ip}
			dev.Addr[idx] = addr
		}

		dev.LastSeen = time.Unix(stamp, 0)

		devices = append(devices, dev)
	}

	return devices, nil
} // func (db *Database) DeviceGetAll(bigheadOnly bool) ([]*model.Device, error)

// DeviceGetByID loads a Device by its ID, if it exists.
func (db *Database) DeviceGetByID(id int64) (*model.Device, error) {
	const qid query.ID = query.DeviceGetByID
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(id); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	if rows.Next() {
		var (
			stamp int64
			addr  string
			dev   = &model.Device{ID: id}
		)

		if err = rows.Scan(&dev.NetID, &dev.Name, &addr, &dev.OS, &dev.BigHead, &stamp); err != nil {
			var ex = fmt.Errorf("Failed to scan row: %w", err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return nil, err
		}

		var alist = make([]string, 0, 2)

		if err = json.Unmarshal([]byte(addr), &alist); err != nil {
			var ex = fmt.Errorf("Cannot device addresses for Device %s: %w",
				dev.Name,
				err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return nil, ex
		}

		dev.Addr = make([]net.Addr, len(alist))
		for idx, astr := range alist {
			var ip net.IP

			if ip = net.ParseIP(astr); ip == nil {
				var ex = fmt.Errorf("Cannot parse IP address of Device %s (%d): %q",
					dev.Name,
					dev.ID,
					astr)
				return nil, ex
			}

			var addr = &net.IPAddr{IP: ip}
			dev.Addr[idx] = addr
		}

		dev.LastSeen = time.Unix(stamp, 0)
		return dev, nil
	}

	return nil, nil
} // func (db *Database) DeviceGetByID(id int64) (*model.Device, error)

// DeviceGetByName loads a Device by its ID, if it exists.
func (db *Database) DeviceGetByName(name string) (*model.Device, error) {
	const qid query.ID = query.DeviceGetByName
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(name); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	if rows.Next() {
		var (
			stamp int64
			addr  string
			dev   = &model.Device{Name: name}
		)

		if err = rows.Scan(&dev.ID, &dev.NetID, &addr, &dev.OS, &dev.BigHead, &stamp); err != nil {
			var ex = fmt.Errorf("Failed to scan row: %w", err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return nil, ex
		}

		dev.LastSeen = time.Unix(stamp, 0)

		var alist = make([]string, 0, 2)

		if err = json.Unmarshal([]byte(addr), &alist); err != nil {
			var ex = fmt.Errorf("Cannot device addresses for Device %s: %w",
				dev.Name,
				err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return nil, ex
		}

		dev.Addr = make([]net.Addr, len(alist))
		for idx, astr := range alist {
			var ip net.IP

			if ip = net.ParseIP(astr); ip == nil {
				var ex = fmt.Errorf("Cannot parse IP address of Device %s (%d): %q",
					dev.Name,
					dev.ID,
					astr)
				return nil, ex
			}

			var addr = &net.IPAddr{IP: ip}
			dev.Addr[idx] = addr
		}

		return dev, nil
	}

	return nil, nil
} // func (db *Database) DeviceGetByName(name string) (*model.Device, error)

// DeviceGetByNetwork returns all Devices that belong to the given Network.
func (db *Database) DeviceGetByNetwork(network *model.Network) ([]*model.Device, error) {
	const qid query.ID = query.DeviceGetByNetwork
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(network.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec
	var devs = make([]*model.Device, 0)

	for rows.Next() {
		var (
			stamp int64
			addr  string
			dev   = &model.Device{NetID: network.ID}
		)

		if err = rows.Scan(&dev.ID, &dev.Name, &addr, &dev.OS, &dev.BigHead, &stamp); err != nil {
			var ex = fmt.Errorf("Failed to scan row: %w", err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return nil, ex
		}

		dev.LastSeen = time.Unix(stamp, 0)

		var alist = make([]string, 0, 2)

		if err = json.Unmarshal([]byte(addr), &alist); err != nil {
			var ex = fmt.Errorf("Cannot device addresses for Device %s: %w",
				dev.Name,
				err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return nil, ex
		}

		dev.Addr = make([]net.Addr, len(alist))
		for idx, astr := range alist {
			var ip net.IP

			if ip = net.ParseIP(astr); ip == nil {
				var ex = fmt.Errorf("Cannot parse IP address of Device %s (%d): %q",
					dev.Name,
					dev.ID,
					astr)
				return nil, ex
			}

			var addr = &net.IPAddr{IP: ip}
			dev.Addr[idx] = addr
		}

		devs = append(devs, dev)
	}

	return devs, nil
} // func (db *Database) DeviceGetByName(name string) (*model.Device, error)

// UptimeAdd adds an uptime/sysload measurement to the Database.
func (db *Database) UptimeAdd(u *model.Uptime) error {
	const qid query.ID = query.UptimeAdd
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Failed to prepare query %s: %s\n",
			qid,
			err.Error())
		panic(err)
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(u.DevID, u.Timestamp.Unix(), u.Uptime, u.Load[0], u.Load[1], u.Load[2]); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("Cannot add Uptime for Device %d: %w",
				u.DevID,
				err)
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	} else {
		var id int64

		defer rows.Close()

		if !rows.Next() {
			// CANTHAPPEN
			db.log.Printf("[ERROR] Query %s did not return a value\n",
				qid)
			return fmt.Errorf("Query %s did not return a value", qid)
		} else if err = rows.Scan(&id); err != nil {
			var ex = fmt.Errorf("Failed to get ID for newly added Uptime: %w",
				err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return ex
		}

		u.ID = id
		return nil
	}
} // func (db *Database) UptimeAdd(u *model.Uptime) error

// UptimeGetByDevice returns the <cnt> most recent uptime value for the given device.
// Pass cnt = -1 to get all.
func (db *Database) UptimeGetByDevice(d *model.Device, cnt int) ([]*model.Uptime, error) {
	const qid query.ID = query.UptimeGetByDevice
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(d.ID, cnt); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec
	var data = make([]*model.Uptime, 0, 16)

	for rows.Next() {
		var (
			stamp int64
			up    = &model.Uptime{DevID: d.ID}
		)

		if err = rows.Scan(&up.ID, &stamp, &up.Load[0], &up.Load[1], &up.Load[2]); err != nil {
			var ex = fmt.Errorf("Failed to scan row: %w", err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return nil, ex
		}

		up.Timestamp = time.Unix(stamp, 0)
		data = append(data, up)
	}

	return data, nil
} // func (db *Database) UptimeGetByDevice(d *model.Device) ([]*model.Uptime, error)

// UpdatesAdd adds a set of pending updates for a Device to the database.
func (db *Database) UpdatesAdd(u *model.Updates) error {
	const qid query.ID = query.UpdatesAdd
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Failed to prepare query %s: %s\n",
			qid,
			err.Error())
		panic(err)
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var (
		rows  *sql.Rows
		upStr string
		buf   []byte
	)

	if buf, err = json.Marshal(u.AvailableUpdates); err != nil {
		db.log.Printf("[ERROR] Failed to serialize AvailableUpdates: %s\n\n%s\n\n",
			err.Error(),
			strings.Join(u.AvailableUpdates, "\n"))
	}

	upStr = string(buf)

EXEC_QUERY:
	if rows, err = stmt.Query(u.DevID, u.Timestamp.Unix(), upStr); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("Cannot add Uptime for Device %d: %w",
				u.DevID,
				err)
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	} else {
		var id int64

		defer rows.Close()

		if !rows.Next() {
			// CANTHAPPEN
			db.log.Printf("[ERROR] Query %s did not return a value\n",
				qid)
			return fmt.Errorf("Query %s did not return a value", qid)
		} else if err = rows.Scan(&id); err != nil {
			var ex = fmt.Errorf("Failed to get ID for newly added Uptime: %w",
				err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return ex
		}

		u.ID = id
		return nil
	}
} // func (db *Database) UpdatesAdd(d *model.Device, u *model.Updates) error

// UpdatesGetByDevice loads the sets of available updates for the given Device in reverse
// chronological order (i.e. most recent first), up to the given maximum number of sets.
// To get all sets, pass max = -1.
func (db *Database) UpdatesGetByDevice(d *model.Device, max int64) ([]*model.Updates, error) {
	const qid query.ID = query.UpdatesGetByDevice
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(d.ID, max); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec
	var data = make([]*model.Updates, 0, 16)

	for rows.Next() {
		var (
			stamp int64
			up    = &model.Updates{DevID: d.ID}
			upstr string
		)

		if err = rows.Scan(&up.ID, &stamp, &upstr); err != nil {
			var ex = fmt.Errorf("Failed to scan row: %w", err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return nil, ex
		} else if err = json.Unmarshal([]byte(upstr), &up.AvailableUpdates); err != nil {
			var ex = fmt.Errorf("Failed to parse AvailableUpdates from JSON: %w\n\n%s",
				err,
				upstr)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return nil, ex
		}

		up.Timestamp = time.Unix(stamp, 0)
		data = append(data, up)
	}

	return data, nil
} // func (db *Database) UpdatesGetByDevice(d *model.Device, max int64) ([]*model.Updates, error)

// UpdatesGetRecent loads the most recent set of updates for each Device.
func (db *Database) UpdatesGetRecent() ([]*model.Updates, error) {
	const qid query.ID = query.UpdatesGetRecent
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec
	var data = make([]*model.Updates, 0, 16)

	for rows.Next() {
		var (
			stamp int64
			up    = new(model.Updates)
			upstr string
		)

		if err = rows.Scan(&up.ID, &up.DevID, &stamp, &upstr); err != nil {
			var ex = fmt.Errorf("Failed to scan row: %w", err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return nil, ex
		} else if err = json.Unmarshal([]byte(upstr), &up.AvailableUpdates); err != nil {
			var ex = fmt.Errorf("Failed to parse AvailableUpdates from JSON: %w\n\n%s",
				err,
				upstr)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return nil, ex
		}

		up.Timestamp = time.Unix(stamp, 0)
		data = append(data, up)
	}

	return data, nil
} // func (db *Database) UpdatesGetRecent() ([]*model.Updates, error)

func (db *Database) DiskFreeAdd(dev *model.Device, free *model.DiskFree) error {
	const qid query.ID = query.InfoAdd
	var (
		err  error
		stmt *sql.Stmt
	)

	if dev.ID != free.DevID {
		return fmt.Errorf("DiskFree info does not belong to Device %s",
			dev.Name)
	} else if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Failed to prepare query %s: %s\n",
			qid,
			err.Error())
		panic(err)
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var (
		rows    *sql.Rows
		freeStr string
		buf     []byte
	)

	if buf, err = json.Marshal(free.PercentFree); err != nil {
		db.log.Printf("[ERROR] Failed to serialize free disk space: %s\n\n%d\n\n",
			err.Error(),
			free.PercentFree)
	}

	freeStr = string(buf)

EXEC_QUERY:
	if rows, err = stmt.Query(free.DevID, free.Timestamp.Unix(), info.DiskFree, freeStr); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("Cannot add Uptime for Device %s: %w",
				dev.Name,
				err)
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	} else {
		var id int64

		defer rows.Close()

		if !rows.Next() {
			// CANTHAPPEN
			db.log.Printf("[ERROR] Query %s did not return a value\n",
				qid)
			return fmt.Errorf("Query %s did not return a value", qid)
		} else if err = rows.Scan(&id); err != nil {
			var ex = fmt.Errorf("Failed to get ID for newly added Uptime: %w",
				err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return ex
		}

		free.ID = id
		return nil
	}
} // func (db *Database) DiskFreeAdd(dev *model.Device, free *model.DiskFree) error

func (db *Database) DiskFreeGet() (map[int64]*model.DiskFree, error) {
	const qid query.ID = query.InfoGetRecent
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(info.DiskFree); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec
	var data = make(map[int64]*model.DiskFree)

	for rows.Next() {
		var (
			stamp   int64
			free    *model.DiskFree = &model.DiskFree{}
			dataStr string
		)

		if err = rows.Scan(&free.ID, &free.DevID, &stamp, &dataStr); err != nil {
			var ex = fmt.Errorf("Failed to scan row: %w", err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return nil, ex
		} else if err = json.Unmarshal([]byte(dataStr), &free.PercentFree); err != nil {
			var ex = fmt.Errorf("Failed to parse free disk space from JSON: %w\n\n%s",
				err,
				dataStr)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return nil, ex
		}

		free.Timestamp = time.Unix(stamp, 0)
		data[free.DevID] = free
	}

	return data, nil
} // func (db *Database) DiskFreeGet(dev *model.Device) (*model.DiskFree, error)
