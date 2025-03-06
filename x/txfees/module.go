package txfees

import (
	"context"
	"encoding/json"

	"cosmossdk.io/api/tendermint/abci"
	"cosmossdk.io/core/appmodule"
	"github.com/CosmWasm/wasmd/x/txfees/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"
)

var (
	_ module.AppModule      = AppModule{}
	_ module.AppModuleBasic = AppModuleBasic{}
	_ appmodule.AppModule   = AppModule{}
)

type AppModule struct {
	AppModuleBasic
}

func NewAppModuleBasic() AppModuleBasic {
	return AppModuleBasic{}
}

// Name returns the x/txfees module's name.
func (AppModuleBasic) Name() string {
	return types.ModuleName
}

func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {}

// RegisterInterfaces registers the module's interface types
func (a AppModuleBasic) RegisterInterfaces(reg cdctypes.InterfaceRegistry) {}

// DefaultGenesis returns the x/txfees module's default genesis state.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return nil
}

// ValidateGenesis performs genesis state validation for the x/txfees module.
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	return nil
}

// RegisterRESTRoutes registers the capability module's REST service handlers.
func (AppModuleBasic) RegisterRESTRoutes(_ client.Context, _ *mux.Router) {
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the module.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
}

// GetTxCmd returns the x/txfees module's root tx command.
func (a AppModuleBasic) GetTxCmd() *cobra.Command {
	return nil
}

// GetQueryCmd returns the x/txfees module's root query command.
func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return nil
}

type AppModuleBasic struct{}

func NewAppModule() AppModule {
	return AppModule{
		AppModuleBasic: NewAppModuleBasic(),
	}
}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (am AppModule) IsOnePerModuleType() { // marker
}

// IsAppModule implements the appmodule.AppModule interface.
func (am AppModule) IsAppModule() { // marker
}

// Name returns the x/txfees module's name.
func (am AppModule) Name() string {
	return am.AppModuleBasic.Name()
}

// QuerierRoute returns the x/txfees module's query routing key.
func (AppModule) QuerierRoute() string { return types.QuerierRoute }

// RegisterServices registers a GRPC query service to respond to the
// module-specific GRPC queries.
func (am AppModule) RegisterServices(cfg module.Configurator) {
}

// RegisterInvariants registers the x/tokenfactory module's invariants.
func (am AppModule) RegisterInvariants(_ sdk.InvariantRegistry) {}

// InitGenesis performs the x/tokenfactory module's genesis initialization. It
// returns no validator updates.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, gs json.RawMessage) []abci.ValidatorUpdate {
	return []abci.ValidatorUpdate{}
}

// ExportGenesis returns the x/tokenfactory module's exported genesis state as raw
// JSON bytes.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	return nil
}

// ConsensusVersion implements ConsensusVersion.
func (AppModule) ConsensusVersion() uint64 { return 1 }

// BeginBlock executes all ABCI BeginBlock logic respective to the tokenfactory module.
func (am AppModule) BeginBlock(_ context.Context) error {
	return nil
}

// EndBlock executes all ABCI EndBlock logic respective to the tokenfactory module. It
// returns no validator updates.
func (am AppModule) EndBlock(_ context.Context) ([]abci.ValidatorUpdate, error) {
	return nil, nil
}
