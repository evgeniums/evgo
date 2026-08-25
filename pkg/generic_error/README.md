# generic_error

Server-side half of the error contract described in
`whitemdesktop/docs/error-contract.md` (in the neutralmdesktop repo, alongside the whitemclient
consumer of this package): every `Error` carries a stable string `Code`, a `Family` namespace, and a
`Disposition` telling the client whether the failure is terminal or worth retrying.

`Disposition` is derived automatically from each code's registered HTTP status
(`ErrorManager.ErrorDisposition`, see `error_manager.go`) unless a package explicitly overrides it
via `AddErrorDispositions`. `Family` is assigned once per declaring package via
`ErrorsExtenderBase.SetFamily`, not per code.

See the doc above for the full wire encoding (gRPC headers, REST JSON, hatn native protocol) and the
rules consumers must follow.
