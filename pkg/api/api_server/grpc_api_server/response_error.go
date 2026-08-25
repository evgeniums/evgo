package grpc_api_server

import (
	"strconv"

	"github.com/evgeniums/evgo/pkg/generic_error"
	"google.golang.org/grpc/metadata"
)

// Default header names error responses carry - mirror the `default:` struct tags on
// ServerConfig's STATUS_HEADER et al (see server.go). A server configured with non-default
// header names cannot use ResponseError and must reconstruct errors from its own ServerConfig
// instead.
const (
	DefaultStatusHeader           = "x-hatn-status"
	DefaultErrorFamilyHeader      = "x-hatn-efamily"
	DefaultErrorDescriptionHeader = "x-hatn-edescription"
	DefaultErrorDetailsHeader     = "x-hatn-edetails"
	DefaultErrorDispositionHeader = "x-hatn-edisposition"
	DefaultErrorRetryAfterHeader  = "x-hatn-eretry-after"
	DefaultSuccessStatus          = "success"
)

// ResponseError reconstructs a generic_error.Error from gRPC response metadata the way
// Handler.fillResponse wrote it - status/family/description/details/disposition/retry_after, see
// whitemdesktop/docs/error-contract.md for the full contract. rpcErr is the error grpc.Invoke
// itself returned; it becomes the reconstructed error's Original(), and is returned as-is when
// hdr carries no status header at all (the request never reached a handler, so there is nothing
// to reconstruct from).
func ResponseError(hdr metadata.MD, rpcErr error) error {
	code := firstHeader(hdr, DefaultStatusHeader)
	if code == "" || code == DefaultSuccessStatus {
		return rpcErr
	}
	gerr := generic_error.New(code, firstHeader(hdr, DefaultErrorDescriptionHeader))
	gerr.SetDetails(firstHeader(hdr, DefaultErrorDetailsHeader))
	gerr.SetFamily(firstHeader(hdr, DefaultErrorFamilyHeader))
	if d := firstHeader(hdr, DefaultErrorDispositionHeader); d != "" {
		gerr.SetDisposition(generic_error.Disposition(d))
	}
	if ra := firstHeader(hdr, DefaultErrorRetryAfterHeader); ra != "" {
		if v, err := strconv.Atoi(ra); err == nil {
			gerr.SetRetryAfter(v)
		}
	}
	gerr.SetOriginal(rpcErr)
	return gerr
}

func firstHeader(hdr metadata.MD, key string) string {
	vs := hdr.Get(key)
	if len(vs) == 0 {
		return ""
	}
	return vs[0]
}
