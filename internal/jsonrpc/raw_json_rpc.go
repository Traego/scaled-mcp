package jsonrpc

type RawJSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  any         `json:"params"`
	ID      interface{} `json:"id,omitempty"` // Can be a string, number or null
}

type RawJSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type RawJSONRPCResponse struct {
	JSONRPC string           `json:"jsonrpc"`
	Result  interface{}      `json:"result,omitempty"`
	Error   *RawJSONRPCError `json:"error,omitempty"`
	ID      interface{}      `json:"id,omitempty"`
}
