package keeper

import (
	"sync"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/CosmWasm/wasmd/x/wasm/types"
)

// ValidatorSetSource is a subset of the staking keeper
type ValidatorSetSource interface {
	ApplyAndReturnValidatorSetUpdates(sdk.Context) (updates []abci.ValidatorUpdate, err error)
}

// max function for two integers
func max(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

func (keeper *Keeper) concurrentCompileCode(codes *[]types.Code) (error, uint64) {
	var wg sync.WaitGroup
	codeLen := len((*codes))
	totalRoutines := int(1)
	// only add more goroutines if there are many code lens
	if codeLen > 100 {
		totalRoutines = int(20)
	}

	wg.Add(totalRoutines)
	maxCodeIDChan := make(chan uint64, max(uint64(codeLen), uint64(totalRoutines))) // Create a channel to collect results
	errChannel := make(chan error, 1)                                               // use buffered channel to remove blocking

	codesPerRoutine := codeLen / int(totalRoutines)
	subCodeLeftIndex := 0

	for i := 0; i < totalRoutines; i++ {
		subCodeLeftIndex = codesPerRoutine * i
		subCodeRightIndex := subCodeLeftIndex + codesPerRoutine
		if i == totalRoutines-1 {
			subCodeRightIndex = codeLen
		}
		go func(left, right int) {
			defer wg.Done()
			maxCodeID := uint64(0)
			for i := left; i < right; i++ {
				code := (*codes)[i]
				// slowest process. We only need to parallel this one
				err := keeper.compileWasmCode(code.CodeInfo, code.CodeBytes)
				if err != nil {
					errChannel <- sdkerrors.Wrapf(err, "code %d with id: %d", i, code.CodeID)
					break
				}
				if code.CodeID > maxCodeID {
					maxCodeID = code.CodeID
				}
			}
			maxCodeIDChan <- maxCodeID
		}(subCodeLeftIndex, subCodeRightIndex)
	}

	// Goroutine to close the channel after all other goroutines are done
	wg.Wait()            // Wait for all goroutines to finish
	close(maxCodeIDChan) // Close the channel
	close(errChannel)    // Close the channel

	err := <-errChannel
	if err != nil {
		return err, 0
	}

	maxCodeID := uint64(0)
	for result := range maxCodeIDChan {
		maxCodeID = max(maxCodeID, result)
	}

	return nil, maxCodeID
}

func (keeper *Keeper) concurrentImportContractState(ctx sdk.Context, contracts *[]types.Contract) error {
	var wg sync.WaitGroup
	contractsLen := len((*contracts))
	totalRoutines := int(1)
	// only add more goroutines if there are many code lens
	if contractsLen > 100 {
		totalRoutines = int(20)
	}

	wg.Add(totalRoutines)
	errChannel := make(chan error, 1)

	contractsPerRoutine := contractsLen / int(totalRoutines)
	subLeftIndex := 0

	var storeMutex sync.Mutex

	for i := 0; i < totalRoutines; i++ {
		subLeftIndex = contractsPerRoutine * i
		subRightIndex := subLeftIndex + contractsPerRoutine
		if i == totalRoutines-1 {
			subRightIndex = contractsLen
		}
		go func(left, right int) {
			// fmt.Println(left, right)
			defer wg.Done()
			for i := left; i < right; i++ {
				contract := (*contracts)[i]
				contractAddr, err := sdk.AccAddressFromBech32(contract.ContractAddress)
				if err != nil {
					errChannel <- sdkerrors.Wrapf(err, "address in contract number %d", i)
					break
				}
				err = keeper.importContractStateWithMutex(ctx, contractAddr, &contract.ContractState, &storeMutex)
				if err != nil {
					ctx.Logger().Error("err import contract: ", err)
					errChannel <- sdkerrors.Wrapf(err, "contract number %d", i)
					break
				}
			}
		}(subLeftIndex, subRightIndex)
	}
	// Goroutine to close the channel after all other goroutines are done
	wg.Wait()         // Wait for all goroutines to finish
	close(errChannel) // Close the channel

	err := <-errChannel
	if err != nil {
		return err
	}

	return nil
}

// InitGenesis sets supply information for genesis.
//
// CONTRACT: all types of accounts must have been already initialized/created
func InitGenesis(ctx sdk.Context, keeper *Keeper, data *types.GenesisState) ([]abci.ValidatorUpdate, error) {
	contractKeeper := NewGovPermissionKeeper(keeper)
	keeper.SetParams(ctx, data.Params)
	err, maxCodeID := keeper.concurrentCompileCode(&data.Codes)
	if err != nil {
		return nil, err
	}
	ctx.Logger().Debug("After compiling code")
	// after compiling, we store wasm info and pin code
	for i, code := range data.Codes {
		err := keeper.storeWasmCode(ctx, code.CodeID, &code.CodeInfo)
		if err != nil {
			return nil, err
		}
		if code.Pinned {
			if err := contractKeeper.PinCode(ctx, code.CodeID); err != nil {
				return nil, sdkerrors.Wrapf(err, "contract number %d", i)
			}
		}
	}
	ctx.Logger().Debug("After store wasm code")
	// allow GC to do its job cleaning Codes if possible
	data.Codes = nil

	maxContractID := len(data.Contracts)

	for i, contract := range data.Contracts {
		contractAddr, err := sdk.AccAddressFromBech32(contract.ContractAddress)
		if err != nil {
			return nil, sdkerrors.Wrapf(err, "address in contract number %d", i)
		}
		err = keeper.importContractWithoutState(ctx, contractAddr, &contract.ContractInfo, contract.ContractCodeHistory)
		if err != nil {
			return nil, sdkerrors.Wrapf(err, "contract number %d", i)
		}
	}

	err = keeper.concurrentImportContractState(ctx, &data.Contracts)
	if err != nil {
		return nil, err
	}

	for i, seq := range data.Sequences {
		err := keeper.importAutoIncrementID(ctx, seq.IDKey, seq.Value)
		if err != nil {
			return nil, sdkerrors.Wrapf(err, "sequence number %d", i)
		}
	}

	// sanity check seq values
	seqVal := keeper.PeekAutoIncrementID(ctx, types.KeyLastCodeID)
	if seqVal <= maxCodeID {
		return nil, sdkerrors.Wrapf(types.ErrInvalid, "seq %s with value: %d must be greater than: %d ", string(types.KeyLastCodeID), seqVal, maxCodeID)
	}
	seqVal = keeper.PeekAutoIncrementID(ctx, types.KeyLastInstanceID)
	if seqVal <= uint64(maxContractID) {
		return nil, sdkerrors.Wrapf(types.ErrInvalid, "seq %s with value: %d must be greater than: %d ", string(types.KeyLastInstanceID), seqVal, maxContractID)
	}

	ctx.Logger().Debug("After importing contracts")

	return nil, nil
}

// ExportGenesis returns a GenesisState for a given context and keeper.
func ExportGenesis(ctx sdk.Context, keeper *Keeper) *types.GenesisState {
	var genState types.GenesisState

	genState.Params = keeper.GetParams(ctx)

	keeper.IterateCodeInfos(ctx, func(codeID uint64, info types.CodeInfo) bool {
		bytecode, err := keeper.GetByteCode(ctx, codeID)
		if err != nil {
			panic(err)
		}
		genState.Codes = append(genState.Codes, types.Code{
			CodeID:    codeID,
			CodeInfo:  info,
			CodeBytes: bytecode,
			Pinned:    keeper.IsPinnedCode(ctx, codeID),
		})
		return false
	})

	keeper.IterateContractInfo(ctx, func(addr sdk.AccAddress, contract types.ContractInfo) bool {
		var state []types.Model
		keeper.IterateContractState(ctx, addr, func(key, value []byte) bool {
			state = append(state, types.Model{Key: key, Value: value})
			return false
		})

		contractCodeHistory := keeper.GetContractHistory(ctx, addr)

		genState.Contracts = append(genState.Contracts, types.Contract{
			ContractAddress:     addr.String(),
			ContractInfo:        contract,
			ContractState:       state,
			ContractCodeHistory: contractCodeHistory,
		})
		return false
	})

	for _, k := range [][]byte{types.KeyLastCodeID, types.KeyLastInstanceID} {
		genState.Sequences = append(genState.Sequences, types.Sequence{
			IDKey: k,
			Value: keeper.PeekAutoIncrementID(ctx, k),
		})
	}

	return &genState
}
