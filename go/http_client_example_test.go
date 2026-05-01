package updater

import . "dappco.re/go"

func ExampleNewHTTPClient() {
	client := NewHTTPClient()
	Println(client.Timeout.String())
}
