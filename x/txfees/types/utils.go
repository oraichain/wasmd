package types

import (
	"encoding/json"

	"cosmossdk.io/math"
)

//	'{"get_sqrt_price": {
//		"denom": ""
//	}}'

type OraidexQueryMsgRequest struct {
	GetSqrtPrice QueryOraidexSqrtPriceRequest `json:"get_sqrt_price"`
}

type QueryOraidexSqrtPriceResponse struct {
	SqrtPrice string `json:"data"`
}

type QueryOraidexSqrtPriceRequest struct {
	Denom string `json:"denom"`
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

func GetOraidexSpotPriceResponse(input []byte) (math.LegacyDec, error) {
	var data QueryOraidexSqrtPriceResponse
	err := json.Unmarshal(input, &data)
	if err != nil {
		return math.LegacyDec{}, err
	}

	var sqrtPrice math.Int
	// this sqrtPrice is decimal 24. We will truncate to decimal 9 by remove 15 character in the last
	lengthDecimal := len(data.SqrtPrice)
	if lengthDecimal > 15 {
		strPrice := data.SqrtPrice[:(lengthDecimal - 15)]
		sqrtPrice, _ = math.NewIntFromString(strPrice)

	} else {
		sqrtPrice = math.ZeroInt()
	}

	decSqrtPrice := math.LegacyNewDecFromIntWithPrec(sqrtPrice, 9)

	return decSqrtPrice, nil
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
