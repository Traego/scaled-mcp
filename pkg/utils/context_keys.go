package utils

type sessionIdCtxKey string
type authInfoCtxKey string
type traceIdCtxKey string

var SessionIdCtx sessionIdCtxKey = "session_id"
var AuthInfoCtx authInfoCtxKey = "auth"
var TraceIdCtx traceIdCtxKey = "trace_id"
