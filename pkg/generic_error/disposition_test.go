package generic_error

import "testing"

// TestCommonErrorDispositions is the audit referenced by whitemdesktop/docs/error-contract.md:
// every code declared in common_errors.go must land on the disposition documented there, either
// through the HTTP-status derivation rule or through an explicit override in
// CommonErrorDispositions. A code failing this test is a signal that either the derivation rule
// is wrong for that status, or the code needs (or no longer needs) an explicit override - not
// that the test is wrong.
func TestCommonErrorDispositions(t *testing.T) {

	m := &ErrorManagerBaseHttp{}
	m.Init()

	cases := []struct {
		code     string
		expected Disposition
	}{
		{ErrorCodeUnknown, DispositionRetry},
		{ErrorCodeInternalServerError, DispositionRetry},
		{ErrorCodeForbidden, DispositionPermanent},
		{ErrorCodeNotFound, DispositionPermanent},
		{ErrorCodeExternalServiceUnavailable, DispositionRetry},
		{ErrorCodeExternalServiceError, DispositionRetry},
		{ErrorCodeUnsupported, DispositionUnknown}, // unmapped HTTP status, no override
		{ErrorCodeExpired, DispositionRetry},       // explicit override - see CommonErrorDispositions
		{ErrorCodeRetryLater, DispositionRetry},
		{ErrorCodeOperationNotPermitted, DispositionPermanent},
		{ErrorCodeResourceBusy, DispositionRetry},
		{ErrorCodeUnimplemented, DispositionUnsupported},
		{ErrorCodeBadRequest, DispositionUnknown}, // unmapped HTTP status, no override
		{ErrorCodeForeignUnavailable, DispositionPermanent},
		{ErrorCodeIOAborted, DispositionPermanent},
		{ErrorCodeConflict, DispositionPermanent},
		{ErrorCodeUnavailable, DispositionRetry},
		// codes with no registered HTTP status and no override must never be silently
		// terminal - this is the whole reason ErrorDisposition does not fall back to the
		// manager's DefaultErrorProtocolCode.
		{ErrorCodeSuccess, DispositionUnknown},
		{ErrorCodeFormat, DispositionUnknown},
		{ErrorCodeFieldValue, DispositionUnknown},
		{ErrorCodeValidation, DispositionUnknown},
		{"totally_unregistered_code", DispositionUnknown},
	}

	for _, c := range cases {
		t.Run(c.code, func(t *testing.T) {
			got := m.ErrorDisposition(c.code)
			if got != c.expected {
				t.Errorf("ErrorDisposition(%q) = %q, want %q", c.code, got, c.expected)
			}
		})
	}
}

func TestDispositionFromHttpStatus(t *testing.T) {
	cases := []struct {
		status   int
		expected Disposition
	}{
		{200, DispositionUnknown},
		{400, DispositionPermanent},
		{401, DispositionUserAction},
		{403, DispositionPermanent},
		{404, DispositionPermanent},
		{408, DispositionRetryAfter},
		{409, DispositionPermanent},
		{425, DispositionRetryAfter},
		{429, DispositionRetryAfter},
		{499, DispositionPermanent},
		{500, DispositionRetry},
		{501, DispositionUnsupported},
		{502, DispositionRetry},
		{503, DispositionRetry},
	}
	for _, c := range cases {
		if got := dispositionFromHttpStatus(c.status); got != c.expected {
			t.Errorf("dispositionFromHttpStatus(%d) = %q, want %q", c.status, got, c.expected)
		}
	}
}

func TestDispositionIsTerminal(t *testing.T) {
	terminal := map[Disposition]bool{
		DispositionUnknown:     false,
		DispositionPermanent:   true,
		DispositionUnsupported: true,
		DispositionRetry:       false,
		DispositionRetryAfter:  false,
		DispositionUserAction:  false,
	}
	for d, want := range terminal {
		if got := d.IsTerminal(); got != want {
			t.Errorf("Disposition(%q).IsTerminal() = %v, want %v", d, got, want)
		}
	}
}
