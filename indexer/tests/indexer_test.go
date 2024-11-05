package tests

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/adlio/schema"
	"github.com/cosmos/gogoproto/proto"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"
	tmlog "github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/types"

	// Register the Postgres database driver.
	"github.com/CosmWasm/wasmd/app/params"
	indexercodec "github.com/CosmWasm/wasmd/indexer/codec"
	indexertx "github.com/CosmWasm/wasmd/indexer/x/tx"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cometbft/cometbft/state/indexer/sink/psql"
	"github.com/cometbft/cometbft/state/txindex"
	sdk "github.com/cosmos/cosmos-sdk/types"
	cosmostx "github.com/cosmos/cosmos-sdk/types/tx"
	_ "github.com/lib/pq"
)

var (
	doPauseAtExit = flag.Bool("pause-at-exit", false,
		"If true, pause the test until interrupted at shutdown, to allow debugging")

	// A hook that test cases can call to obtain the shared database instance
	// used for testing the sink. This is initialized in TestMain (see below).
	testDB func() *sql.DB

	encodingConfig params.EncodingConfig
)

func init() {
	encodingConfig = indexercodec.MakeEncodingConfig()
}

const (
	user     = "admin"
	password = "root"
	port     = "5432"
	dsn      = "postgres://%s:%s@localhost:%s/%s?sslmode=disable"
	dbName   = "postgres"
	chainID  = "testing"

	viewBlockEvents = "block_events"
	viewTxEvents    = "tx_events"
)

func TestMarshalJson(t *testing.T) {
	fee := cosmostx.Fee{Amount: sdk.NewCoins(sdk.NewCoin("orai", math.NewInt(1))), GasLimit: 10000, Payer: sdk.AccAddress("orai1wsg0l9c6tr5uzjrhwhqch9tt4e77h0w28wvp3u").String(), Granter: sdk.AccAddress("orai1wsg0l9c6tr5uzjrhwhqch9tt4e77h0w28wvp3u").String()}
	feeBz, err := json.Marshal(fee)
	require.NoError(t, err)
	fmt.Println(string(feeBz))
	require.Equal(t, `{"amount":[{"denom":"orai","amount":"1"}],"gas_limit":10000,"payer":"orai1daexz6f3waekwvrv893nvarjx46h56njdpmksutrdquhgap5v5mnw6pswuersamkwqeh2g6rztw","granter":"orai1daexz6f3waekwvrv893nvarjx46h56njdpmksutrdquhgap5v5mnw6pswuersamkwqeh2g6rztw"}`, string(feeBz))
}

func TestMain(m *testing.M) {
	flag.Parse()

	// Set up docker and start a container running PostgreSQL.
	pool, err := dockertest.NewPool(os.Getenv("DOCKER_URL"))
	if err != nil {
		log.Fatalf("Creating docker pool: %v", err)
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "13",
		Env: []string{
			"POSTGRES_USER=" + user,
			"POSTGRES_PASSWORD=" + password,
			"POSTGRES_DB=" + dbName,
			"listen_addresses = '*'",
		},
		ExposedPorts: []string{port + "/tcp"},
	}, func(config *docker.HostConfig) {
		// set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{
			Name: "no",
		}
	})
	if err != nil {
		log.Fatalf("Starting docker pool: %v", err)
	}

	if *doPauseAtExit {
		log.Print("Pause at exit is enabled, containers will not expire")
	} else {
		const expireSeconds = 60
		_ = resource.Expire(expireSeconds)
		log.Printf("Container expiration set to %d seconds", expireSeconds)
	}

	// Connect to the database, clear any leftover data, and install the
	// indexing schema.
	conn := fmt.Sprintf(dsn, user, password, resource.GetPort(port+"/tcp"), dbName)
	var db *sql.DB

	if err := pool.Retry(func() error {
		sink, err := psql.NewEventSink(conn, chainID)
		if err != nil {
			return err
		}
		db = sink.DB() // set global for test use
		return db.Ping()
	}); err != nil {
		log.Fatalf("Connecting to database: %v", err)
	}

	if err := resetDatabase(db); err != nil {
		log.Fatalf("Flushing database: %v", err)
	}

	sm, err := readSchema()
	if err != nil {
		log.Fatalf("Reading schema: %v", err)
	}
	migrator := schema.NewMigrator()
	if err := migrator.Apply(db, sm); err != nil {
		log.Fatalf("Applying schema: %v", err)
	}

	// Set up the hook for tests to get the shared database handle.
	testDB = func() *sql.DB { return db }

	// Run the selected test cases.
	code := m.Run()

	// Clean up and shut down the database container.
	if *doPauseAtExit {
		log.Print("Testing complete, pausing for inspection. Send SIGINT to resume teardown")
		waitForInterrupt()
		log.Print("(resuming)")
	}
	log.Print("Shutting down database")
	if err := pool.Purge(resource); err != nil {
		log.Printf("WARNING: Purging pool failed: %v", err)
	}
	if err := db.Close(); err != nil {
		log.Printf("WARNING: Closing database failed: %v", err)
	}

	os.Exit(code)
}

func TestIndexing(t *testing.T) {
	t.Run("IndexBlockEvents", func(t *testing.T) {
		indexer := psql.NewEventSinkFromDB(testDB(), chainID)
		require.NoError(t, indexer.IndexBlockEvents(newTestBlockEvents()))

		verifyBlock(t, 1)
		verifyBlock(t, 2)

		verifyNotImplemented(t, "hasBlock", func() (bool, error) { return indexer.HasBlock(1) })
		verifyNotImplemented(t, "hasBlock", func() (bool, error) { return indexer.HasBlock(2) })

		verifyNotImplemented(t, "block search", func() (bool, error) {
			v, err := indexer.SearchBlockEvents(context.Background(), nil)
			return v != nil, err
		})

		require.NoError(t, verifyTimeStamp(psql.TableBlocks))

		// Attempting to reindex the same events should gracefully succeed.
		require.NoError(t, indexer.IndexBlockEvents(newTestBlockEvents()))
	})

	t.Run("IndexTxEvents", func(t *testing.T) {
		indexer := psql.NewEventSinkFromDB(testDB(), chainID)

		txResult := txResultWithEvents([]abci.Event{
			psql.MakeIndexedEvent("account.number", "1"),
			psql.MakeIndexedEvent("account.owner", "Ivan"),
			psql.MakeIndexedEvent("account.owner", "Yulieta"),

			{Type: "", Attributes: []abci.EventAttribute{
				{
					Key:   "not_allowed",
					Value: "Vlad",
					Index: true,
				},
			}},
		})
		require.NoError(t, indexer.IndexTxEvents([]*abci.TxResult{txResult}))

		txr, err := loadTxResult(types.Tx(txResult.Tx).Hash())
		require.NoError(t, err)
		assert.Equal(t, txResult, txr)

		require.NoError(t, verifyTimeStamp(psql.TableTxResults))
		require.NoError(t, verifyTimeStamp(viewTxEvents))

		verifyNotImplemented(t, "getTxByHash", func() (bool, error) {
			txr, err := indexer.GetTxByHash(types.Tx(txResult.Tx).Hash())
			return txr != nil, err
		})
		verifyNotImplemented(t, "tx search", func() (bool, error) {
			txr, err := indexer.SearchTxEvents(context.Background(), nil)
			return txr != nil, err
		})

		// try to insert the duplicate tx events.
		err = indexer.IndexTxEvents([]*abci.TxResult{txResult})
		require.NoError(t, err)

		// test loading tx events
		height := uint64(1)
		txEvent, err := loadTxEvents(height)
		require.NoError(t, err)
		txHash := fmt.Sprintf("%X", types.Tx(txResult.Tx).Hash())
		require.Equal(t, txEvent, &indexertx.TxEvent{Height: height, ChainId: chainID, Type: "tx", Key: "hash", Value: txHash})
	})

	t.Run("IndexerService", func(t *testing.T) {
		indexer := psql.NewEventSinkFromDB(testDB(), chainID)

		// event bus
		eventBus := types.NewEventBus()
		err := eventBus.Start()
		require.NoError(t, err)
		t.Cleanup(func() {
			if err := eventBus.Stop(); err != nil {
				t.Error(err)
			}
		})

		service := txindex.NewIndexerService(indexer.TxIndexer(), indexer.BlockIndexer(), eventBus, true)
		service.SetLogger(tmlog.TestingLogger())
		err = service.Start()
		require.NoError(t, err)
		t.Cleanup(func() {
			if err := service.Stop(); err != nil {
				t.Error(err)
			}
		})

		// publish block with txs
		err = eventBus.PublishEventNewBlockEvents(types.EventDataNewBlockEvents{
			Height: 1,
			NumTxs: 2,
		})
		require.NoError(t, err)
		txResult1 := &abci.TxResult{
			Height: 1,
			Index:  uint32(0),
			Tx:     types.Tx("foo"),
			Result: abci.ExecTxResult{Code: 0},
		}
		err = eventBus.PublishEventTx(types.EventDataTx{TxResult: *txResult1})
		require.NoError(t, err)
		txResult2 := &abci.TxResult{
			Height: 1,
			Index:  uint32(1),
			Tx:     types.Tx("bar"),
			Result: abci.ExecTxResult{Code: 1},
		}
		err = eventBus.PublishEventTx(types.EventDataTx{TxResult: *txResult2})
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)
		require.True(t, service.IsRunning())
	})

	t.Run("IndexCosmWasmTxs", func(t *testing.T) {
		indexer := psql.NewEventSinkFromDB(testDB(), chainID)

		txResult := wasmTxResultWithEvents([]abci.Event{
			psql.MakeIndexedEvent("account.number", "1"),
			psql.MakeIndexedEvent("account.owner", "Ivan"),
			psql.MakeIndexedEvent("account.owner", "Yulieta"),

			{Type: "", Attributes: []abci.EventAttribute{
				{
					Key:   "not_allowed",
					Value: "Vlad",
					Index: true,
				},
			}},
		})
		require.NoError(t, indexer.IndexTxEvents([]*abci.TxResult{txResult}))

		// try indexing tx requests
		txs := [][]byte{}
		txs = append(txs, txResult.Tx)
		customTxEventSink := indexertx.NewTxEventSinkIndexer(indexer, encodingConfig)
		time := time.Now()
		err := customTxEventSink.InsertModuleEvents(&abci.RequestFinalizeBlock{Txs: txs, Time: time}, &abci.ResponseFinalizeBlock{Events: []abci.Event{}})
		require.NoError(t, err)

		// txr, err := loadTxResult(types.Tx(txResult.Tx).Hash())
		// require.NoError(t, err)
		// assert.Equal(t, txResult, txr)

		// require.NoError(t, verifyTimeStamp(psql.TableTxResults))
		// require.NoError(t, verifyTimeStamp(viewTxEvents))

		// verifyNotImplemented(t, "getTxByHash", func() (bool, error) {
		// 	txr, err := indexer.GetTxByHash(types.Tx(txResult.Tx).Hash())
		// 	return txr != nil, err
		// })
		// verifyNotImplemented(t, "tx search", func() (bool, error) {
		// 	txr, err := indexer.SearchTxEvents(context.Background(), nil)
		// 	return txr != nil, err
		// })

		// // try to insert the duplicate tx events.
		// err = indexer.IndexTxEvents([]*abci.TxResult{txResult})
		// require.NoError(t, err)

		// // test loading tx events
		// height := uint64(1)
		// txEvent, err := loadTxEvents(height)
		// require.NoError(t, err)
		// txHash := fmt.Sprintf("%X", types.Tx(txResult.Tx).Hash())
		// require.Equal(t, txEvent, &indexertx.TxEvent{Height: height, ChainId: chainID, Type: "tx", Key: "hash", Value: txHash})
	})
}

func TestStop(t *testing.T) {
	indexer := psql.NewEventSinkFromDB(testDB(), chainID)
	require.NoError(t, indexer.Stop())
}

// newTestBlock constructs a fresh copy of a new block event containing
// known test values to exercise the indexer.
func newTestBlockEvents() types.EventDataNewBlockEvents {
	return types.EventDataNewBlockEvents{
		Height: 1,
		Events: []abci.Event{
			psql.MakeIndexedEvent("begin_event.proposer", "FCAA001"),
			psql.MakeIndexedEvent("thingy.whatzit", "O.O"),
			psql.MakeIndexedEvent("end_event.foo", "100"),
			psql.MakeIndexedEvent("thingy.whatzit", "-.O"),
		},
	}
}

// readSchema loads the indexing database schema file
func readSchema() ([]*schema.Migration, error) {
	filename := filepath.Join("../", "db_sql", "schema.sql")
	contents, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read sql file from '%s': %w", filename, err)
	}

	return []*schema.Migration{{
		ID:     time.Now().Local().String() + " db schema",
		Script: string(contents),
	}}, nil
}

// resetDB drops all the data from the test database.
func resetDatabase(db *sql.DB) error {
	_, err := db.Exec(`DROP TABLE IF EXISTS blocks,tx_results,events,attributes CASCADE;`)
	if err != nil {
		return fmt.Errorf("dropping tables: %v", err)
	}
	_, err = db.Exec(`DROP VIEW IF EXISTS event_attributes,block_events,tx_events CASCADE;`)
	if err != nil {
		return fmt.Errorf("dropping views: %v", err)
	}
	return nil
}

// txResultWithEvents constructs a fresh transaction result with fixed values
// for testing, that includes the specified events.
func txResultWithEvents(events []abci.Event) *abci.TxResult {
	return &abci.TxResult{
		Height: 1,
		Index:  0,
		Tx:     types.Tx("HELLO WORLD"),
		Result: abci.ExecTxResult{
			Data:   []byte{0},
			Code:   abci.CodeTypeOK,
			Log:    "",
			Events: events,
		},
	}
}

// txResultWithEvents constructs a fresh transaction result with fixed values
// for testing, that includes the specified events.
func wasmTxResultWithEvents(events []abci.Event) *abci.TxResult {

	txBuilder := encodingConfig.TxConfig.NewTxBuilder()
	grant := "orai1wsg0l9c6tr5uzjrhwhqch9tt4e77h0w28wvp3u"
	instantiateMsg := wasmtypes.MsgInstantiateContract{
		Sender: grant,
		CodeID: 0,
		Label:  "label",
		Funds:  sdk.NewCoins(sdk.NewCoin("orai", math.NewInt(100))),
		Msg:    []byte(wasmtypes.RawContractMessage{}),
		Admin:  grant,
	}

	if err := txBuilder.SetMsgs(&instantiateMsg); err != nil {
		panic(err)
	}
	// txBuilder.SetMemo("hello world")
	// txBuilder.SetFeeGranter(sdk.MustAccAddressFromBech32(grant))
	// txBuilder.SetFeePayer(sdk.MustAccAddressFromBech32(grant))
	txBuilder.SetGasLimit(10000000)
	txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin("orai", math.NewInt(1))))
	// txBuilder.SetMemo("memo foobar")
	tx := txBuilder.GetTx()
	txBz, err := encodingConfig.TxConfig.TxEncoder()(tx)
	if err != nil {
		panic(err)
	}

	return &abci.TxResult{
		Height: 1,
		Index:  0,
		Tx:     txBz,
		Result: abci.ExecTxResult{
			Data:   []byte{0},
			Code:   abci.CodeTypeOK,
			Log:    "",
			Events: events,
		},
	}
}

func loadTxResult(hash []byte) (*abci.TxResult, error) {
	hashString := fmt.Sprintf("%X", hash)
	var resultData []byte
	if err := testDB().QueryRow(`
SELECT tx_result FROM `+psql.TableTxResults+` WHERE tx_hash = $1;
`, hashString).Scan(&resultData); err != nil {
		return nil, fmt.Errorf("lookup transaction for hash %q failed: %v", hashString, err)
	}

	txr := new(abci.TxResult)
	if err := proto.Unmarshal(resultData, txr); err != nil {
		return nil, fmt.Errorf("unmarshaling txr: %v", err)
	}

	return txr, nil
}

func loadTxEvents(height uint64) (*indexertx.TxEvent, error) {
	var Height uint64
	var ChainId string
	var Type string
	var Key string
	var Value string
	if err := testDB().QueryRow(`
SELECT height, chain_id, type, key, value FROM `+viewTxEvents+` WHERE height = $1;
`, height).Scan(&Height, &ChainId, &Type, &Key, &Value); err != nil {
		return nil, fmt.Errorf("lookup tx event for height %d failed: %v", height, err)
	}

	return &indexertx.TxEvent{Height: Height, ChainId: ChainId, Type: Type, Key: Key, Value: Value}, nil
}

func verifyTimeStamp(tableName string) error {
	return testDB().QueryRow(fmt.Sprintf(`
SELECT DISTINCT %[1]s.created_at
  FROM %[1]s
  WHERE %[1]s.created_at >= $1;
`, tableName), time.Now().Add(-2*time.Second)).Err()
}

func verifyBlock(t *testing.T, height int64) {
	// Check that the blocks table contains an entry for this height.
	if err := testDB().QueryRow(`
SELECT height FROM `+psql.TableBlocks+` WHERE height = $1;
`, height).Err(); err == sql.ErrNoRows {
		t.Errorf("No block found for height=%d", height)
	} else if err != nil {
		t.Fatalf("Database query failed: %v", err)
	}

	// Verify the presence of begin_block and end_block events.
	if err := testDB().QueryRow(`
SELECT type, height, chain_id FROM `+viewBlockEvents+`
  WHERE height = $1 AND type = $2 AND chain_id = $3;
`, height, psql.EventTypeFinalizeBlock, chainID).Err(); err == sql.ErrNoRows {
		t.Errorf("No %q event found for height=%d", psql.EventTypeFinalizeBlock, height)
	} else if err != nil {
		t.Fatalf("Database query failed: %v", err)
	}
}

// verifyNotImplemented calls f and verifies that it returns both a
// false-valued flag and a non-nil error whose string matching the expected
// "not supported" message with label prefixed.
func verifyNotImplemented(t *testing.T, label string, f func() (bool, error)) {
	t.Helper()
	t.Logf("Verifying that %q reports it is not implemented", label)

	want := label + " is not supported via the postgres event sink"
	ok, err := f()
	assert.False(t, ok)
	require.NotNil(t, err)
	assert.Equal(t, want, err.Error())
}

// waitForInterrupt blocks until a SIGINT is received by the process.
func waitForInterrupt() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch
}
