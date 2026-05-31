// Package audit emits a single-line, key=value audit record for every
// privileged command invocation (Tesla owner-api proxies and TeslaMate
// logging proxies). The format is intentionally terse and stable so it
// can be grep'd, shipped to a SIEM, or parsed by logfmt-aware tooling
// without per-deployment customisation.
//
// Body content is never logged — request bodies can carry pin codes
// (valet, speed-limit) and access tokens are entirely off-limits. We log
// presence (req_bytes / resp_bytes) instead.
package audit

import (
	"fmt"
	"log"
	"strings"
)

// Outcome enumerates the terminal states of a command invocation. Stable
// values; downstream parsers can switch on them.
const (
	OutcomeSuccess  = "success"
	OutcomeDenied   = "denied"
	OutcomeError    = "error"
	OutcomeUpstream = "upstream_error"
)

// Reason codes for OutcomeDenied / OutcomeError. Stable, machine-friendly.
const (
	ReasonCommandsDisabled  = "commands_disabled"
	ReasonUnauthorized      = "unauthorized"
	ReasonInvalidCarID      = "invalid_car_id"
	ReasonNotInAllowList    = "not_in_allowlist"
	ReasonMissingEncKey     = "missing_encryption_key"
	ReasonTokenDecryptFail  = "token_decrypt_failed"
	ReasonDBLookupFailed    = "db_lookup_failed"
	ReasonRequestBuildFail  = "request_build_failed"
	ReasonReadBodyFail      = "read_body_failed"
	ReasonUpstreamReachFail = "upstream_unreachable"
	ReasonUpstreamNonJSON   = "upstream_non_json"
)

// Event is one audit record. Fields default to zero-values; only populated
// fields are emitted, so a deny-before-DB-lookup record stays compact.
type Event struct {
	Action       string // command_list, command_exec, logging_list, logging_exec
	Method       string // HTTP method of the inbound request (GET/POST/PUT)
	CarID        int    // 0 = not yet parsed
	Command      string // /command/door_unlock, /wake_up, /logging/resume, ""
	ClientIP     string
	UserAgent    string
	Outcome      string
	Reason       string // free-form for OutcomeUpstream; one of Reason* for the rest
	UpstreamURL  string // sanitised — never includes Authorization header / token
	UpstreamCode int    // 0 if upstream was never reached
	ReqBytes     int    // size of request body forwarded upstream
	RespBytes    int    // size of response body received from upstream
	DurationMS   int64  // wall-clock from request hitting the handler to outcome
	ErrDetail    string // free-form diagnostic — kept short, redact-safe
}

// Log emits the event as a single log line. Format:
//
//	[audit] action=command_exec method=POST car_id=1 cmd="/wake_up" \
//	        client_ip=10.0.0.1 outcome=success upstream_status=200 \
//	        upstream="https://owner-api.teslamotors.com/api/1/vehicles/.../wake_up" \
//	        dur_ms=128 req_bytes=0 resp_bytes=412
//
// Caller passes Event by value to make zero-args misuse obvious. Empty
// fields are omitted; unknown outcomes still log so we never silently drop.
func Log(e Event) {
	var b strings.Builder
	b.Grow(256)
	b.WriteString("[audit]")
	writeKV(&b, "action", e.Action)
	writeKV(&b, "method", e.Method)
	if e.CarID != 0 {
		writeKVInt(&b, "car_id", e.CarID)
	}
	writeKV(&b, "cmd", e.Command)
	writeKV(&b, "client_ip", e.ClientIP)
	writeKV(&b, "outcome", e.Outcome)
	writeKV(&b, "reason", e.Reason)
	if e.UpstreamCode != 0 {
		writeKVInt(&b, "upstream_status", e.UpstreamCode)
	}
	writeKV(&b, "upstream", e.UpstreamURL)
	if e.DurationMS > 0 {
		writeKVInt64(&b, "dur_ms", e.DurationMS)
	}
	if e.ReqBytes > 0 {
		writeKVInt(&b, "req_bytes", e.ReqBytes)
	}
	if e.RespBytes > 0 {
		writeKVInt(&b, "resp_bytes", e.RespBytes)
	}
	writeKV(&b, "user_agent", e.UserAgent)
	writeKV(&b, "err", e.ErrDetail)
	log.Println(b.String())
}

func writeKV(b *strings.Builder, k, v string) {
	if v == "" {
		return
	}
	b.WriteByte(' ')
	b.WriteString(k)
	b.WriteByte('=')
	if needsQuote(v) {
		fmt.Fprintf(b, "%q", v)
	} else {
		b.WriteString(v)
	}
}

func writeKVInt(b *strings.Builder, k string, v int) {
	fmt.Fprintf(b, " %s=%d", k, v)
}

func writeKVInt64(b *strings.Builder, k string, v int64) {
	fmt.Fprintf(b, " %s=%d", k, v)
}

// needsQuote returns true when the value contains characters that would
// confuse a logfmt parser (whitespace, ", =) or is empty.
func needsQuote(s string) bool {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case ' ', '\t', '\n', '\r', '"', '=':
			return true
		}
	}
	return false
}
