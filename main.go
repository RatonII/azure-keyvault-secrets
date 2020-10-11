package main

import (
	"flag"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	kvauth "github.com/Azure/azure-sdk-for-go/services/keyvault/auth"
	"github.com/Azure/azure-sdk-for-go/services/web/mgmt/2020-06-01/web"
	aauth "github.com/Azure/go-autorest/autorest/azure/auth"
	"log"
	"runtime"
	"strings"
	"sync"
)

func main() {
	var wg sync.WaitGroup
	var sub string
	var resourceGr string
	var vault string
	var c Functions

	authorizer, err := kvauth.NewAuthorizerFromCLI()
	if err != nil {
		log.Fatalf("unable to create keyvault authorizer: %v\n", err)
	}
	basicClient := keyvault.New()
	basicClient.Authorizer = authorizer

	vaultName := flag.String("vault", "", "Then name of the keyvault where to store secrets")
	runtime.GOMAXPROCS(4)
	secrets := flag.String("secrets", "", "Add a config file yaml with all the pipelines contains")
	subscription := flag.String("subscription", "", "Add a config file yaml with all the pipelines contains")
	resourceGroup := flag.String("resource-group", "", "Add a config file yaml with all the pipelines contains")
	storefunckeys := flag.Bool("storefunckeys", false,"This is going to be used only if you want to store the functions key in keyvault")
	flag.Parse()

	if *vaultName != "" {
		vault = *vaultName
	} else {
		log.Fatalln("Please provide a keyvault name with argument: --vault")
	}

	if *secrets != "" {
		if *vaultName != "" {
			vault = *vaultName
		} else {
			log.Fatalln("Please provide a keyvault name with argument: --vault")
		}

		entries := strings.Split(*secrets, ",")
		newsecrets := map[string]string{}
		for _, e := range entries {
			parts := strings.Split(e, "=")
			newsecrets[parts[0]] = parts[1]
		}
		wg.Add(len(newsecrets))
		for k, v := range newsecrets {
			go createUpdateSecret(basicClient, k, v, vault, &wg)
		}
		wg.Wait()
	}
	if *storefunckeys == true {
		if *subscription != "" {
			sub = *subscription
		} else {
			log.Fatalln("Please provide a subscription for your azure account: --subscription")
		}
		if *resourceGroup != "" {
			resourceGr = *resourceGroup
		} else {
			log.Fatalln("Please provide a  resource group for your azure account: --resource-group")
		}
		webclient := web.NewAppsClient(sub)
		authorizer, err = aauth.NewAuthorizerFromCLI()
		if err != nil {
			log.Fatalf("unable to create function authorizer: %v\n", err)
		}
		webclient.Authorizer = authorizer

		funcfile := "functions-secrets.yaml"
		funcsecrets := c.getConf(&funcfile)
		wg.Add(len(*funcsecrets) * 2)
		for _, function := range *funcsecrets {
			go createUpdateFunctionsSecret(basicClient, webclient, resourceGr, function.Name, function.SecretKeyName, vault, &wg)
		}
		wg.Wait()
	}
}
