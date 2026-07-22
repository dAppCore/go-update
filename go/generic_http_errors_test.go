package updater

import . "dappco.re/go"

func TestGenericHttpErrors_BadStatus(t *T) {
	// A non-200 status from the update server surfaces a status-code failure.
	server := NewHTTPTestServer(HandlerFunc(func(w ResponseWriter, r *Request) {
		w.WriteHeader(503)
	}))
	defer server.Close()

	result := GetLatestUpdateFromURL(server.URL)

	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "status code 503")
}

func TestGenericHttpErrors_MissingFields(t *T) {
	// Valid JSON but empty version/url is rejected as invalid content.
	server := NewHTTPTestServer(HandlerFunc(func(w ResponseWriter, r *Request) {
		write := WriteString(w, `{"version":"","url":""}`)
		AssertTrue(t, write.OK)
	}))
	defer server.Close()

	result := GetLatestUpdateFromURL(server.URL)

	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "version or url is missing")
}

func TestGenericHttpErrors_FetchError(t *T) {
	// An unreachable host fails the fetch (no server listening on this port).
	result := GetLatestUpdateFromURL("http://127.0.0.1:1")

	AssertFalse(t, result.OK)
	AssertContains(t, result.Error(), "failed to fetch latest.json")
}

func TestGenericHttpErrors_TrailingSlashBaseURL(t *T) {
	// A trailing slash on the base URL is normalised before latest.json append.
	server := NewHTTPTestServer(HandlerFunc(func(w ResponseWriter, r *Request) {
		AssertEqual(t, "/latest.json", r.URL.Path)
		write := WriteString(w, `{"version":"1.0.0","url":"https://updates.example.com/app"}`)
		AssertTrue(t, write.OK)
	}))
	defer server.Close()

	result := GetLatestUpdateFromURL(server.URL + "/")

	AssertTrue(t, result.OK)
	AssertEqual(t, "1.0.0", result.Value.(*GenericUpdateInfo).Version)
}
