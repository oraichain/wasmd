package interchaintest

type QueryMsg struct {
	// IBC Hooks
	GetCount *GetCountQuery `json:"get_count,omitempty"`
}

type GetCountQuery struct{}

type GetCountResponse struct {
	Data CountObj `json: "data"`
}

type CountObj struct {
	Count uint64 `json:"count"`
}
