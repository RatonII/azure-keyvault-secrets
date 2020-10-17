package main

type Functions []Function

type Function struct {
	Name 			 	string	`yaml:"funcName"`
	SecretKeyName 		string	`yaml:"secretkeyname"`
}

type CosmosAccounts []CosmosAccount

type CosmosAccount struct {
	AccountName 			 	string	`yaml:"accountName"`
	CosmosdbKeys				map[string]string `yaml:"cosmosdbKeys"`
}

type EventGrids []EventGrid

type EventGrid struct {
	DomainName 	string 			 `yaml:"domainName"`
	AccessKeys map[string]string `yaml:"eventgridKeys"`
}

type arrayFlags []string

