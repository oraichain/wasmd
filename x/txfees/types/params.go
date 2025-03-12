package types

func DefaultParams() Params {
	return Params{
		PriceContractAddress: "",
	}
}

func (p Params) Validate() error {
	return nil
}
