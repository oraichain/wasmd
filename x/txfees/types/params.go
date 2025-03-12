package types

// TODO: Default params
func DefaultParams() Params {
	return Params{
		PriceContractAddress: "",
	}
}

func (p Params) Validate() error {
	return nil
}
