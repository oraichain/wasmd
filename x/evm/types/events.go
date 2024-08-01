package types

// Evm module events
const (
	EventTypeEthereumTx = TypeMsgEthereumTx
	EventTypeBlockBloom = "block_bloom"
	EventTypeTxLog      = "tx_log"

	EventTypeSetMappingEvmAddress    = "set_mapping_evm_address"
	EventTypeDeleteMappingEvmAddress = "delete_mapping_evm_address"

	AttributeKeyContractAddress = "contract"
	AttributeKeyRecipient       = "recipient"
	AttributeKeyTxHash          = "txHash"
	AttributeKeyEthereumTxHash  = "ethereumTxHash"
	AttributeKeyTxIndex         = "txIndex"
	AttributeKeyTxGasUsed       = "txGasUsed"
	AttributeKeyTxType          = "txType"
	AttributeKeyTxLog           = "txLog"
	// tx failed in eth vm execution
	AttributeKeyEthereumTxFailed = "ethereumTxFailed"
	AttributeValueCategory       = ModuleName
	AttributeKeyEthereumBloom    = "bloom"

	MetricKeyTransitionDB = "transition_db"
	MetricKeyStaticCall   = "static_call"

	// Event Attributes - EVM Mapping address
	AttributeKeyCosmosAddress = "cosmos_address"
	AttributeKeyEvmAddress    = "evm_address"
	AttributeKeyPubkey        = "pubkey"
)
