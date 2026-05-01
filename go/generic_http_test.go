package updater

import . "dappco.re/go"

func TestGenericHttp_GetLatestUpdateFromURL_Good(t *T) {
	server := NewHTTPTestServer(HandlerFunc(func(w ResponseWriter, r *Request) {
		AssertEqual(t, "/latest.json", r.URL.Path)
		write := WriteString(w, `{"version":"v1.2.0","url":"https://updates.example.com/app"}`)
		AssertTrue(t, write.OK)
	}))
	defer server.Close()

	result := GetLatestUpdateFromURL(server.URL)

	AssertTrue(t, result.OK)
	info := result.Value.(*GenericUpdateInfo)
	AssertEqual(t, "v1.2.0", info.Version)
	AssertEqual(t, "https://updates.example.com/app", info.URL)
}

func TestGenericHttp_GetLatestUpdateFromURL_Bad(t *T) {
	result := GetLatestUpdateFromURL("://bad-url")

	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "invalid base URL")
}

func TestGenericHttp_GetLatestUpdateFromURL_Ugly(t *T) {
	server := NewHTTPTestServer(HandlerFunc(func(w ResponseWriter, r *Request) {
		write := WriteString(w, `{"version":`)
		AssertTrue(t, write.OK)
	}))
	defer server.Close()

	result := GetLatestUpdateFromURL(server.URL)

	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "failed to parse latest.json")
}
