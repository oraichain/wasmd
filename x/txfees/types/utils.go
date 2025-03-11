package types

import "cosmossdk.io/math"

// '{"get_sqrt_price": {}}'
type QueryOraidexSpotPriceRequest struct {
}

// {"data":"632447252548187055131688"}
type QueryOraidexSpotPriceResponse struct {
}

// TODO: implement
func BuildQueryOraidexSpotPriceRequest() []byte {
	return []byte{}
}

// TODO: implement
func GetOraidexSpotPriceResponse(data []byte) math.LegacyDec {
	//
	return math.LegacyDec{}
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
