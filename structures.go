package main

type Functions []Function

type Function struct {
	Name          string `yaml:"funcName"`
	SecretKeyName string `yaml:"secretkeyname"`
}

type CosmosAccounts []CosmosAccount

type CosmosAccount struct {
	AccountName  string            `yaml:"accountName"`
	CosmosdbKeys map[string]string `yaml:"cosmosdbKeys"`
}

type EventGrids []EventGrid

type EventGrid struct {
	DomainName string            `yaml:"domainName"`
	AccessKeys map[string]string `yaml:"eventgridKeys"`
}

type DataFactories []DataFactory

type DataFactory struct {
	FactoryName            string `yaml:"factoryName"`
	IntegrationRuntimeName string `yaml:"integrationRuntimeName"`
	IntegrationRuntimeKey1 string `yaml:"integrationRuntimeKey1"`
	IntegrationRuntimeKey2 string `yaml:"integrationRuntimeKey2"`
}

type StorageAccounts []StorageAccount

type StorageAccount struct {
	StorageName string `yaml:"storageName"`
	AccessKey1  string `yaml:"accessKey1"`
}

type arrayFlags []string
