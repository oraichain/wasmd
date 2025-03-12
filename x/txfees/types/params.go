package types

import fmt "fmt"

func DefaultParams() Params {
	return Params{
		TokenBaseDenom:       "orai",
		PriceContractAddress: "orai149eps23rjaedn8m9qlcgcx3py5h49fx7ehtjla3yjp5amgmdppwq74ml4j",
	}
}

func (p Params) Validate() error {
	if err := validateString(p.TokenBaseDenom); err != nil {
		return err
	}
	if err := validateString(p.PriceContractAddress); err != nil {
		return err
	}

	return nil
}

func validateString(i interface{}) error {
	_, ok := i.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type string: %T", i)
	}

	return nil
}
