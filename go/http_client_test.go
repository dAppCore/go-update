package updater

import . "dappco.re/go"

func TestHttpClient_NewHTTPClient_Configured(t *T) {
	client := NewHTTPClient()

	AssertNotNil(t, client)
	AssertEqual(t, defaultHTTPTimeout, client.Timeout)
}
