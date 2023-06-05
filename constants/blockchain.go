package constants

type Blockchain string

const (
	BlockKey = "block"
)

const (
	UNKNOWN                        = "unknown"
	Base                Blockchain = "base" // Coinbase
	Binance_Smart_Chain Blockchain = "binance_smart_chain"
	Ethereum            Blockchain = "ethereum"
	Optimism            Blockchain = "optimism"
	Polygon             Blockchain = "polygon"
	Arbitrum            Blockchain = "arbitrum"
)
