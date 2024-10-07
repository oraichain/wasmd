package registry

import (
	pcommon "github.com/CosmWasm/wasmd/precompile/common"
	"github.com/CosmWasm/wasmd/precompile/contracts/addr"
	"github.com/CosmWasm/wasmd/precompile/contracts/bank"
	"github.com/CosmWasm/wasmd/precompile/contracts/json"
	"github.com/CosmWasm/wasmd/precompile/contracts/wasmd"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/precompile/contract"
	"github.com/ethereum/go-ethereum/precompile/modules"
)

var (
	// WasmdContractAddress the primary noop contract address for testing
	WasmdContractAddress = common.HexToAddress("0x9000000000000000000000000000000000000001")
	JsonContractAddress  = common.HexToAddress("0x9000000000000000000000000000000000000002")
	AddrContractAddress  = common.HexToAddress("0x9000000000000000000000000000000000000003")
	BankContractAddress  = common.HexToAddress("0x9000000000000000000000000000000000000004")
)

// init registers stateful precompile contracts with the global precompile registry
// defined in kava-labs/go-ethereum/precompile/modules
func InitializePrecompiles(wasmdKeeper pcommon.WasmdKeeper, wasmdViewKeeper pcommon.WasmdViewKeeper, evmKeeper pcommon.EVMKeeper, bankKeeper pcommon.BankKeeper, accountKeeper pcommon.AccountKeeper) {
	register(WasmdContractAddress, wasmd.NewContract(wasmdKeeper, wasmdViewKeeper, evmKeeper))
	register(JsonContractAddress, json.NewContract())
	register(AddrContractAddress, addr.NewContract(evmKeeper))
	register(BankContractAddress, bank.NewContract(evmKeeper, bankKeeper, accountKeeper))

}

// register accepts a 0x address string and a stateful precompile contract constructor, instantiates the
// precompile contract via the constructor, and registers it with the precompile module registry.
func register(moduleAddress common.Address, contract contract.StatefulPrecompiledContract) {

	// do not check found, allowing override
	module := modules.Module{
		Address:  moduleAddress,
		Contract: contract,
	}

	modules.RegisterModule(module)

}
