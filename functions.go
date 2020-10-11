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
	"io/ioutil"
	"log"
	"os"
	"path"
	"sync"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	"github.com/Azure/azure-sdk-for-go/services/web/mgmt/2020-06-01/web"
	"gopkg.in/yaml.v3"
)

func getWebAppsClient(subscriptionID string) (client web.AppsClient, err error) {
	client = web.NewAppsClient(subscriptionID)
	//client.Authorizer, err = iam.GetResourceManagementAuthorizer()
	//if err != nil {
	//	return
	//}
	//err = client.AddToUserAgent(config.UserAgent())
	//if err != nil {
	//	return
	//}
	return
}

// List masterkey for accessing the functionapp
func ListAppHostKeys(ctx context.Context, subscriptionID string, resourceGroupName string, name string) {
	client, err := getWebAppsClient(subscriptionID)
	if err != nil {
		return
	}
	keys, err := client.ListHostKeys(ctx, resourceGroupName, name)
	if err != nil {
		return
	}
	fmt.Println(*keys.MasterKey)

}

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
								resourceGroup,funcname,secname,vaultName string, wg *sync.WaitGroup) {
	f, err := webClient.ListHostKeys(context.Background(), resourceGroup, funcname)
	if err != nil {
		panic(err)
	}
	createUpdateSecret(basicClient, secname, *f.MasterKey, vaultName, wg)
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
