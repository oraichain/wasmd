package tx_test

import (
	"fmt"
	"regexp"
	"testing"

	"cosmossdk.io/math"
	"github.com/CosmWasm/wasmd/app/params"
	indexercodec "github.com/CosmWasm/wasmd/indexer/codec"
	"github.com/CosmWasm/wasmd/indexer/x/tx"
	abci "github.com/cometbft/cometbft/abci/types"
	cometbftindexer "github.com/cometbft/cometbft/state/indexer"
	"github.com/cometbft/cometbft/state/txindex/kv"
	"github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	cosmostx "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/stretchr/testify/require"
)

var (
	encodingConfig params.EncodingConfig
)

func init() {
	encodingConfig = indexercodec.MakeEncodingConfig()
}

func TestMarshalJson(t *testing.T) {
	fee := cosmostx.Fee{Amount: sdk.NewCoins(sdk.NewCoin("orai", math.NewInt(1))), GasLimit: 10000, Payer: sdk.AccAddress("orai1wsg0l9c6tr5uzjrhwhqch9tt4e77h0w28wvp3u").String(), Granter: sdk.AccAddress("orai1wsg0l9c6tr5uzjrhwhqch9tt4e77h0w28wvp3u").String()}
	feeBz, err := encodingConfig.Codec.MarshalJSON(&fee)
	require.NoError(t, err)
	fmt.Println(string(feeBz))
	require.Equal(t, `{"amount":[{"denom":"orai","amount":"1"}],"gas_limit":"10000","payer":"orai1daexz6f3waekwvrv893nvarjx46h56njdpmksutrdquhgap5v5mnw6pswuersamkwqeh2g6rztw","granter":"orai1daexz6f3waekwvrv893nvarjx46h56njdpmksutrdquhgap5v5mnw6pswuersamkwqeh2g6rztw"}`, string(feeBz))
}

func TestUnMarshalEventsString(t *testing.T) {
	events := `[{"type":"coin_received","attributes":[{"key":"receiver","value":"orai16ukn20yqph0d5n4dhwxg5xmfz4wr2gwgqnw3pg"},{"key":"amount","value":"1orai"},{"key":"msg_index","value":"0"}]},{"type":"message","attributes":[{"key":"action","value":"/cosmos.bank.v1beta1.MsgSend"},{"key":"sender","value":"orai1wpyljf0pgewpaleundtm0yp4lv4kmxaj6y3weg"},{"key":"module","value":"bank"},{"key":"msg_index","value":"0"},{"key":"sender","value":"orai1wpyljf0pgewpaleundtm0yp4lv4kmxaj6y3weg"},{"key":"msg_index","value":"0"}]},{"type":"tx","attributes":[{"key":"hash","value":"93BD79267C253031FE341641AADE420B65C31307098EDB8BD8022DE2B8433143"},{"key":"height","value":"35"},{"key":"fee","value":""},{"key":"fee_payer","value":"orai1wpyljf0pgewpaleundtm0yp4lv4kmxaj6y3weg"},{"key":"acc_seq","value":"7049f925e1465c1eff3c9b57b79035fb2b6d9bb22f32\\n"},{"key":"signature","value":"R9ARuR1tCCZVtq7dVv+ZIHl7gHIN1bV92IN+XVEnffQPTK+H/5P0ZZ3p8FbPAK+mZ30XzwPZQkJmUjY5KuQmuw=="}]},{"type":"coin_spent","attributes":[{"key":"spender","value":"orai1wpyljf0pgewpaleundtm0yp4lv4kmxaj6y3weg"},{"key":"amount","value":"1orai"},{"key":"msg_index","value":"0"}]},{"type":"transfer","attributes":[{"key":"recipient","value":"orai16ukn20yqph0d5n4dhwxg5xmfz4wr2gwgqnw3pg"},{"key":"sender","value":"orai1wpyljf0pgewpaleundtm0yp4lv4kmxaj6y3weg"},{"key":"amount","value":"1orai"},{"key":"msg_index","value":"0"}]}]`

	var eventsProto []abci.Event
	err := encodingConfig.Amino.UnmarshalJSON([]byte(events), &eventsProto)
	require.NoError(t, err)
}

func TestCreateHeightRangeWhereConditions(t *testing.T) {
	heightInfo := kv.HeightInfo{}
	heightInfo.SetHeight(2)
	testCases := []struct {
		name         string
		queryRanges  cometbftindexer.QueryRanges
		heightInfo   kv.HeightInfo
		expectedVals []interface{}
		expectedSQL  string
	}{
		{
			name: "Inclusive lower bound, exclusive upper bound",
			queryRanges: cometbftindexer.QueryRanges{
				types.TxHeightKey: cometbftindexer.QueryRange{
					LowerBound:        int64(5),
					UpperBound:        int64(10),
					Key:               types.TxHeightKey,
					IncludeLowerBound: true,
					IncludeUpperBound: false,
				},
			},
			heightInfo:   heightInfo,
			expectedVals: []interface{}{int64(5), int64(10)},
			expectedSQL:  "WHEREtx_requests.height>=$1ANDtx_requests.height<$2",
		},
		{
			name: "Exclusive lower and upper bounds",
			queryRanges: cometbftindexer.QueryRanges{
				types.TxHeightKey: cometbftindexer.QueryRange{
					LowerBound:        int64(5),
					UpperBound:        int64(10),
					Key:               types.TxHeightKey,
					IncludeLowerBound: false,
					IncludeUpperBound: false,
				},
			},
			heightInfo:   heightInfo,
			expectedVals: []interface{}{int64(5), int64(10)},
			expectedSQL:  "WHEREtx_requests.height>$1ANDtx_requests.height<$2",
		},
		{
			name: "Inclusive upper bound only",
			queryRanges: cometbftindexer.QueryRanges{
				types.TxHeightKey: cometbftindexer.QueryRange{
					UpperBound:        int64(15),
					Key:               types.TxHeightKey,
					IncludeLowerBound: false,
					IncludeUpperBound: true,
				},
			},
			heightInfo:   heightInfo,
			expectedVals: []interface{}{int64(15)},
			expectedSQL:  "WHEREtx_requests.height<=$1",
		},
		{
			name: "Exclusive lower bound only",
			queryRanges: cometbftindexer.QueryRanges{
				types.TxHeightKey: cometbftindexer.QueryRange{
					LowerBound:        int64(20),
					Key:               types.TxHeightKey,
					IncludeLowerBound: false,
					IncludeUpperBound: true,
				},
			},
			heightInfo:   heightInfo,
			expectedVals: []interface{}{int64(20)},
			expectedSQL:  "WHEREtx_requests.height>$1",
		},
		{
			name:         "Equal only",
			queryRanges:  cometbftindexer.QueryRanges{},
			heightInfo:   heightInfo,
			expectedVals: []interface{}{int64(2)},
			expectedSQL:  "WHEREtx_requests.height=$1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			heightRange, ok := tc.queryRanges[types.TxHeightKey]
			tc.heightInfo.SetheightRange(heightRange)
			query, vals, _ := tx.CreateHeightRangeWhereConditions(tc.heightInfo, ok)
			require.Equal(t, tc.expectedVals, vals)

			re := regexp.MustCompile(`\s+`)
			trimmedQuery := re.ReplaceAllString(query, "")
			fmt.Println(trimmedQuery)
			require.Equal(t, tc.expectedSQL, trimmedQuery)
		})
	}
}
