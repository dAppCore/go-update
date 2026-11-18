package updater

import . "github.com/dappcore/go"

func ExampleNewHTTPClient() {
	client := NewHTTPClient()
	Println(client.Timeout.String())
}
