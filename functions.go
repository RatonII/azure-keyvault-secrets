package main

// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.
//
//
// You need to set four environment variables before using the app:
// AZURE_TENANT_ID: Your Azure tenant ID
// AZURE_CLIENT_ID: Your Azure client ID. This will be an app ID from your AAD.
// KVAULT_NAME: The name of your vault (just the name, not the full URL/path)
//
// Optional command line argument:
// If you have a secret already, set KVAULT_SECRET_NAME to the secret's name.
//
// NOTE: Do NOT set AZURE_CLIENT_SECRET. This example uses Managed identities.
// The README.md provides more information.
//
//

import (
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/storage/mgmt/storage"
	"github.com/Azure/azure-sdk-for-go/services/cosmos-db/mgmt/2020-04-01/documentdb"
	"github.com/Azure/azure-sdk-for-go/services/datafactory/mgmt/2018-06-01/datafactory"
	"github.com/Azure/azure-sdk-for-go/services/eventgrid/mgmt/2020-06-01/eventgrid"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	"github.com/Azure/azure-sdk-for-go/services/web/mgmt/2020-06-01/web"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"os"
	"path"
	"sync"
)

func listSecrets(basicClient keyvault.BaseClient, vaultName string) {
	secretList, err := basicClient.GetSecrets(context.Background(), "https://"+vaultName+".vault.azure.net", nil)
	if err != nil {
		log.Fatalf("unable to get list of secrets: %v\n", err)
	}
	// group by ContentType
	secWithType := make(map[string][]string)
	secWithoutType := make([]string, 1)
	for _, secret := range secretList.Values() {
		if secret.ContentType != nil {
			_, exists := secWithType[*secret.ContentType]
			if exists {
				secWithType[*secret.ContentType] = append(secWithType[*secret.ContentType], path.Base(*secret.ID))
			} else {
				tempSlice := make([]string, 1)
				tempSlice[0] = path.Base(*secret.ID)
				secWithType[*secret.ContentType] = tempSlice
			}
		} else {
			secWithoutType = append(secWithoutType, path.Base(*secret.ID))
		}
	}

	for k, v := range secWithType {
		fmt.Println(k)
		for _, sec := range v {
			fmt.Println(sec)
		}
	}
	fmt.Println(len(secWithoutType))
	for _, wov := range secWithoutType {
		fmt.Println(wov)
	}
}

func getSecret(basicClient keyvault.BaseClient, secname string, vaultName string) {
	secretResp, err := basicClient.GetSecret(context.Background(), "https://"+vaultName+".vault.azure.net", secname, "")
	if err != nil {
		fmt.Printf("unable to get value for secret: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(*secretResp.Value)
}

func createUpdateSecret(basicClient keyvault.BaseClient, secname, secvalue string, vaultName string, wg *sync.WaitGroup) {
	var secParams keyvault.SecretSetParameters
	secParams.Value = &secvalue
	newBundle, err := basicClient.SetSecret(context.Background(), "https://"+vaultName+".vault.azure.net", secname, secParams)
	if err != nil {
		log.Fatalf("unable to add/update secret: %v\n", err)
	}
	fmt.Println("added/updated: " + *newBundle.ID)
	defer wg.Done()
}

func createUpdateFunctionsSecret(basicClient keyvault.BaseClient, webClient web.AppsClient,
	resourceGroup, funcname, secname, vaultName string, wg *sync.WaitGroup) {
	f, err := webClient.ListHostKeys(context.Background(), resourceGroup, funcname)
	if err != nil {
		panic(err)
	}
	createUpdateSecret(basicClient, secname, *f.MasterKey, vaultName, wg)
	defer wg.Done()
}

func createUpdateCosmosSecret(basicClient keyvault.BaseClient, cosmosClient documentdb.DatabaseAccountsClient,
	resourceGroup, cosmosAccountName, vaultName string, keysNames map[string]string, wg *sync.WaitGroup) {
	var wf sync.WaitGroup
	f, err := cosmosClient.ListKeys(context.Background(), resourceGroup, cosmosAccountName)
	if err != nil {
		panic(err)
	}
	cosmoskeys := map[string]string{}
	for k, v := range keysNames {
		if k == "primaryMasterKey" {
			cosmoskeys[v] = *f.PrimaryMasterKey
			cosmoskeys[fmt.Sprintf("%s-conn-string", v)] = fmt.Sprintf("AccountEndpoint=https://%s.documents.azure.com:443/;AccountKey=%s;", cosmosAccountName, *f.PrimaryMasterKey)
		} else if k == "primaryReadonlyKey" {
			cosmoskeys[v] = *f.PrimaryReadonlyMasterKey
		} else if k == "secondaryMasterKey" {
			cosmoskeys[v] = *f.SecondaryReadonlyMasterKey
		} else if k == "secondaryReadonlyKey" {
			cosmoskeys[v] = *f.PrimaryMasterKey
		}
	}
	wf.Add(len(cosmoskeys))
	for k, v := range cosmoskeys {
		go createUpdateSecret(basicClient, k, v, vaultName, &wf)
	}
	wf.Wait()
	defer wg.Done()
}

func createUpdateEventGridSecret(basicClient keyvault.BaseClient, eventGridClient eventgrid.DomainsClient,
	resourceGroup, domainName, vaultName string, keysNames map[string]string, wg *sync.WaitGroup) {
	var wf sync.WaitGroup
	e, err := eventGridClient.ListSharedAccessKeys(context.Background(), resourceGroup, domainName)
	if err != nil {
		panic(err)
	}
	eventgridkeys := map[string]string{}
	for k, v := range keysNames {
		if k == "Key1" {
			eventgridkeys[v] = *e.Key1
		} else if k == "Key2" {
			eventgridkeys[v] = *e.Key2
		}
	}
	wf.Add(len(eventgridkeys))
	for k, v := range eventgridkeys {
		go createUpdateSecret(basicClient, k, v, vaultName, &wf)
	}
	wf.Wait()
	defer wg.Done()
}

func createUpdateAdfIntegratedRuntimeSecret(basicClient keyvault.BaseClient,
	integratedRuntimeClient datafactory.IntegrationRuntimesClient,
	resourceGroup, factoryName, integrationRuntimeName,key1,key2, vaultName string, wg *sync.WaitGroup) {
	var wf sync.WaitGroup
	d, err := integratedRuntimeClient.ListAuthKeys(context.Background(), resourceGroup, factoryName, integrationRuntimeName)
	if err != nil {
		panic(err)
	}
	integratedRuntimekeys := map[string]string{}
	integratedRuntimekeys[key1] = *d.AuthKey1
	integratedRuntimekeys[key2] = *d.AuthKey2

	wf.Add(len(integratedRuntimekeys))
	for k, v := range integratedRuntimekeys {
		go createUpdateSecret(basicClient, k, v, vaultName, &wf)
	}
	wf.Wait()
	defer wg.Done()
}

func createUpdateStorageAccountSecret(basicClient keyvault.BaseClient,
	storageAccountClient storage.AccountsClient,
	resourceGroup, accountName, key1,vaultName string, wg *sync.WaitGroup) {
	var wf sync.WaitGroup
	s, err := storageAccountClient.ListKeys(context.Background(), resourceGroup, accountName, storage.Kerb)
	if err != nil {
		panic(err)
	}
	//storagekeys := map[string]string{}
	storagekey1 := (*s.Keys)[0].Value
	//for _, j := range *s.Keys {
	//
	//		storagekeys[v] = *j.Value
	//}
	wf.Add(1)
	go createUpdateSecret(basicClient, key1, *storagekey1, vaultName, &wf)
	//wf.Add(len(integratedRuntimekeys))
	//for k, v := range integratedRuntimekeys {
	//	go createUpdateSecret(basicClient, k, v, vaultName, &wf)
	//}
	wf.Wait()
	defer wg.Done()
}

func deleteSecret(basicClient keyvault.BaseClient, secname string, vaultName string) {
	_, err := basicClient.DeleteSecret(context.Background(), "https://"+vaultName+".vault.azure.net", secname)
	if err != nil {
		log.Fatalf("error deleting secret: %v\n", err)
	}
	_, err = basicClient.PurgeDeletedSecret(context.Background(), "https://"+vaultName+".vault.azure.net", secname)
	if err != nil {
		log.Fatalf("error purging secret: %v\n", err)
	}
	fmt.Println(secname + "was deleted and purged successfully")
}

func (f *Functions) getConf(FunctionsFile *string) *Functions {

	yamlFile, err := ioutil.ReadFile(*FunctionsFile)
	if err != nil {
		log.Fatalf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, f)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return f
}

func (c *CosmosAccounts) getConf(FunctionsFile *string) *CosmosAccounts {

	yamlFile, err := ioutil.ReadFile(*FunctionsFile)
	if err != nil {
		log.Fatalf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return c
}

func (e *EventGrids) getConf(EventGridFile *string) *EventGrids {

	yamlFile, err := ioutil.ReadFile(*EventGridFile)
	if err != nil {
		log.Fatalf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, e)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return e
}

func (d *DataFactories) getConf(DataFactoryFile *string) *DataFactories {

	yamlFile, err := ioutil.ReadFile(*DataFactoryFile)
	if err != nil {
		log.Fatalf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, d)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return d
}

func (s *StorageAccounts) getConf(StorageFile *string) *StorageAccounts {

	yamlFile, err := ioutil.ReadFile(*StorageFile)
	if err != nil {
		log.Fatalf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, s)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return s
}


func (i *arrayFlags) String() string {
	return fmt.Sprint(*i)
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}
