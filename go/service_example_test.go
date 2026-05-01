package updater

import . "dappco.re/go"

func ExampleNewUpdateService() {
	result := NewUpdateService(UpdateServiceConfig{RepoURL: "https://github.com/core/update"})
	Println(result.OK)
}

func ExampleUpdateService_Start() {
	service := &UpdateService{config: UpdateServiceConfig{CheckOnStartup: NoCheck}}
	result := service.Start()
	Println(result.OK)
}

func ExampleParseRepoURL() {
	result := ParseRepoURL("https://github.com/core/update")
	Println(result.Value.([]string)[0])
}
