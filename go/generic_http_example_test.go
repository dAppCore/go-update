package updater

import . "dappco.re/go"

func ExampleGetLatestUpdateFromURL() {
	server := NewHTTPTestServer(HandlerFunc(func(w ResponseWriter, r *Request) {
		WriteString(w, `{"version":"v1.2.0","url":"https://updates.example.com/app"}`)
	}))
	defer server.Close()
	result := GetLatestUpdateFromURL(server.URL)
	Println(result.OK)
}
