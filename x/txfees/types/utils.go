package types

import (
	"encoding/json"

	"cosmossdk.io/math"
)

//	'{"get_sqrt_price": {
//		"quote_token": ""
//	}}'

type OraidexQueryMsgRequest struct {
	GetSqrtPrice QueryOraidexSqrtPriceRequest `json:"get_sqrt_price"`
}

type QueryOraidexSqrtPriceResponse struct {
	SqrtPrice string `json:"data"`
}

type QueryOraidexSqrtPriceRequest struct {
	Denom string `json:"quote_token"`
}

func BuildQueryOraidexSpotPriceRequest(denom string) ([]byte, error) {
	req := OraidexQueryMsgRequest{
		GetSqrtPrice: QueryOraidexSqrtPriceRequest{
			Denom: denom,
		},
	}

	bz, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	return bz, nil
}

func GetOraidexSqrtPriceResponse(input []byte) (math.LegacyDec, error) {
	strSqrtPrice := string(input)[1:(len(string(input)) - 1)]
	// this sqrtPrice is decimal 24. We will truncate to decimal 6 by remove 18 character in the last
	lengthDecimal := len(strSqrtPrice)
	if lengthDecimal > 18 {
		strPrice := strSqrtPrice[:(lengthDecimal - 18)]
		sqrtPrice, ok := math.NewIntFromString(strPrice)
		if !ok {
			panic("bigIntOverflows")
		}
		decSqrtPrice := math.LegacyNewDecFromIntWithPrec(sqrtPrice, 6)
		return decSqrtPrice, nil

	}

	return math.LegacyNewDecFromIntWithPrec(math.OneInt(), 6), nil
}

type QueryOraidexTwapRequest struct {
}

type QueryOraidexTwapResponse struct {
}

func BuildQueryOraidexTwapRequest() []byte {
	return []byte{}
}

func ParseOraidexTwapResponse() {

}
