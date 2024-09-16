package registry

import (
	"fmt"

	pcommon "github.com/CosmWasm/wasmd/precompile/common"
	"github.com/CosmWasm/wasmd/precompile/contracts/wasmd"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/precompile/contract"
	"github.com/ethereum/go-ethereum/precompile/modules"
)

var (
	// WasmdContractAddress the primary noop contract address for testing
	WasmdContractAddress = common.HexToAddress("0x9000000000000000000000000000000000000001")
)

// init registers stateful precompile contracts with the global precompile registry
// defined in kava-labs/go-ethereum/precompile/modules
func InitializePrecompiles(wasmdKeeper pcommon.WasmdKeeper, wasmdViewKeeper pcommon.WasmdViewKeeper, evmKeeper pcommon.EVMKeeper) {
	wasmdContract, err := wasmd.NewContract(wasmdKeeper, wasmdViewKeeper, evmKeeper)
	if err != nil {
		panic(fmt.Errorf("error creating contract for address %s: %w", WasmdContractAddress, err))
	}

	register(WasmdContractAddress, wasmdContract)
}

// register accepts a 0x address string and a stateful precompile contract constructor, instantiates the
// precompile contract via the constructor, and registers it with the precompile module registry.
//
// This panics if the contract can not be created or the module can not be registered
func register(moduleAddress common.Address, contract contract.StatefulPrecompiledContract) {

	// if already found then return
	_, found := modules.GetPrecompileModuleByAddress(moduleAddress)

	if found {
		return
	}

	module := modules.Module{
		Address:  moduleAddress,
		Contract: contract,
	}

	modules.RegisterModule(module)

}
