package types

import "cosmossdk.io/math"

type QueryOraidexSpotPriceRequest struct {
}

type QueryOraidexSpotPriceResponse struct {
}

// TODO: implement
func BuildQueryOraidexSpotPriceRequest() []byte {
	return []byte{}
}

// TODO: implement
func GetOraidexSpotPriceResponse(data []byte) math.LegacyDec {
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
