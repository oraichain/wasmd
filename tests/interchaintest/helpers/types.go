package helpers

import (
	"time"

	types1 "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type ParamChange struct {
	Subspace string `json:"subspace"`
	Key      string `json:"key"`
	Value    any    `json:"value"`
}

// TokenFactory type
type QueryTokenFactoryParamsResponse struct {
	Params TokenFactoryParams `json:"params"`
}

type TokenFactoryParams struct {
	DenomCreationFee        sdk.Coins `json:"denom_creation_fee"`
	DenomCreationGasConsume string    `json:"denom_creation_gas_consume,omitempty"`
}

type Account struct {
	AccountNumber uint64   `json:"account_number"`
	Address       string   `json:"address"`
	Name          string   `json:"name"`
	Permissions   []string `json:"permissions"`
	PublicKey     string   `json:"public_key"`
	Sequence      uint64   `json:"sequence"`
}

type ModuleAccount struct {
	Type  string  `json:"type"`
	Value Account `json:"value"`
}

type AuthModuleAccounts struct {
	Accounts []ModuleAccount `json:"accounts"`
}

type QueryDenomsFromCreatorResponse struct {
	Denoms []string `json:"denoms"`
}

type QueryDenomAuthorityMetadataResponse struct {
	AuthorityMetadata DenomAuthorityMetadata `json:"authority_metadata"`
}

type DenomAuthorityMetadata struct {
	Admin string `json:"admin"`
}

type QueryWasmGasLessContracts struct {
	ContractAddresses []string     `json:"contract_addresses"`
	Pagination        PageResponse `json:"pagination"`
}

type QueryTxfeesTokenExchangeRate struct {
	BaseDenom  string `json:"base_denom"`
	QuoteDenom string `json:"quote_denom"`
	Rate       string `json:"rate"`
}

type PageResponse struct {
	NextKey []byte `json:"next_key"`
	Total   string `json:"total"`
}

type HackatomExampleInitMsg struct {
	Verifier    string `json:"verifier"`
	Beneficiary string `json:"beneficiary"`
}

type QuerySqrtPriceResponse struct {
	Data string `json:"data,omitempty"`
}

type QueryProposalResponse struct {
	Proposal Proposal `protobuf:"bytes,1,opt,name=proposal,proto3" json:"proposal"`
}

type Proposal struct {
	// id defines the unique id of the proposal.
	Id string `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	// messages are the arbitrary messages to be executed if the proposal passes.
	Messages []*types1.Any `protobuf:"bytes,2,rep,name=messages,proto3" json:"messages,omitempty"`
	// status defines the proposal status.
	Status string `protobuf:"varint,3,opt,name=status,proto3,enum=cosmos.gov.v1.ProposalStatus" json:"status,omitempty"`
	// final_tally_result is the final tally result of the proposal. When
	// querying a proposal via gRPC, this field is not populated until the
	// proposal's voting period has ended.
	FinalTallyResult *TallyResult `protobuf:"bytes,4,opt,name=final_tally_result,json=finalTallyResult,proto3" json:"final_tally_result,omitempty"`
	// submit_time is the time of proposal submission.
	SubmitTime *time.Time `protobuf:"bytes,5,opt,name=submit_time,json=submitTime,proto3,stdtime" json:"submit_time,omitempty"`
	// deposit_end_time is the end time for deposition.
	DepositEndTime *time.Time `protobuf:"bytes,6,opt,name=deposit_end_time,json=depositEndTime,proto3,stdtime" json:"deposit_end_time,omitempty"`
	// total_deposit is the total deposit on the proposal.
	TotalDeposit []sdk.Coin `protobuf:"bytes,7,rep,name=total_deposit,json=totalDeposit,proto3" json:"total_deposit"`
	// voting_start_time is the starting time to vote on a proposal.
	VotingStartTime *time.Time `protobuf:"bytes,8,opt,name=voting_start_time,json=votingStartTime,proto3,stdtime" json:"voting_start_time,omitempty"`
	// voting_end_time is the end time of voting on a proposal.
	VotingEndTime *time.Time `protobuf:"bytes,9,opt,name=voting_end_time,json=votingEndTime,proto3,stdtime" json:"voting_end_time,omitempty"`
	// metadata is any arbitrary metadata attached to the proposal.
	// the recommended format of the metadata is to be found here:
	// https://docs.cosmos.network/v0.47/modules/gov#proposal-3
	Metadata string `protobuf:"bytes,10,opt,name=metadata,proto3" json:"metadata,omitempty"`
	// title is the title of the proposal
	//
	// Since: cosmos-sdk 0.47
	Title string `protobuf:"bytes,11,opt,name=title,proto3" json:"title,omitempty"`
	// summary is a short summary of the proposal
	//
	// Since: cosmos-sdk 0.47
	Summary string `protobuf:"bytes,12,opt,name=summary,proto3" json:"summary,omitempty"`
	// proposer is the address of the proposal sumbitter
	//
	// Since: cosmos-sdk 0.47
	Proposer string `protobuf:"bytes,13,opt,name=proposer,proto3" json:"proposer,omitempty"`
	// expedited defines if the proposal is expedited
	//
	// Since: cosmos-sdk 0.50
	Expedited bool `protobuf:"varint,14,opt,name=expedited,proto3" json:"expedited,omitempty"`
	// failed_reason defines the reason why the proposal failed
	//
	// Since: cosmos-sdk 0.50
	FailedReason string `protobuf:"bytes,15,opt,name=failed_reason,json=failedReason,proto3" json:"failed_reason,omitempty"`
}

type TallyResult struct {
	// yes_count is the number of yes votes on a proposal.
	YesCount string `protobuf:"bytes,1,opt,name=yes_count,json=yesCount,proto3" json:"yes_count,omitempty"`
	// abstain_count is the number of abstain votes on a proposal.
	AbstainCount string `protobuf:"bytes,2,opt,name=abstain_count,json=abstainCount,proto3" json:"abstain_count,omitempty"`
	// no_count is the number of no votes on a proposal.
	NoCount string `protobuf:"bytes,3,opt,name=no_count,json=noCount,proto3" json:"no_count,omitempty"`
	// no_with_veto_count is the number of no with veto votes on a proposal.
	NoWithVetoCount string `protobuf:"bytes,4,opt,name=no_with_veto_count,json=noWithVetoCount,proto3" json:"no_with_veto_count,omitempty"`
}
