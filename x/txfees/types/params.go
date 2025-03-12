package types

func DefaultParams() Params {
	return Params{
		TokenBaseDenom:       "orai",
		PriceContractAddress: "",
	}
}

func (p Params) Validate() error {
	return nil
}
