package main

import (
	"flag"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/cosmos-db/mgmt/2020-04-01/documentdb"
	kvauth "github.com/Azure/azure-sdk-for-go/services/keyvault/auth"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	"github.com/Azure/azure-sdk-for-go/services/web/mgmt/2020-06-01/web"
	"github.com/Azure/azure-sdk-for-go/services/eventgrid/mgmt/2020-06-01/eventgrid"
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
	var cosmos CosmosAccounts
	var event EventGrids
	var secretFlag arrayFlags

	authorizer, err := kvauth.NewAuthorizerFromCLI()
	if err != nil {
		log.Fatalf("unable to create keyvault authorizer: %v\n", err)
	}
	basicClient := keyvault.New()
	basicClient.Authorizer = authorizer

	vaultName := flag.String("vault", "", "Then name of the keyvault where to store secrets")
	runtime.GOMAXPROCS(4)
	//secrets := flag.String("secrets", "", "Add the secrets to store in the keyvaut: --secrets ex: secretname=secretvalue,othersecretname=othersecretvalue")
	flag.Var(&secretFlag, "secret", "Add the secrets to store in the keyvaut: --secrets ex: secretname=secretvalue")
	subscription := flag.String("subscription", "", "The name of the subscription for cosmosdb or function to get the secrets from")
	resourceGroup := flag.String("resource-group", "", "The name of the resource group for cosmosdb or function to get the secrets from")
	storefunckeys := flag.Bool("storefunckeys", false,"This is going to be used only if you want to store the functions key in keyvault")
	storecosmoskeys := flag.Bool("storecosmoskeys", false,"This is going to be used only if you want to store the cosmosdb keys in keyvault")
	storeeventgridkeys := flag.Bool("storeeventskeys", false,"This is going to be used only if you want to store the event grid keys in keyvault")
	secrets := &secretFlag
	flag.Parse()

	if *vaultName != "" {
		vault = *vaultName
	} else {
		log.Fatalln("Please provide a keyvault name with argument: --vault")
	}
	if *secrets != nil {
		newsecrets := map[string]string{}
		for _, e := range *secrets {
			parts := strings.Split(e, "=")
			newsecrets[parts[0]] = parts[1]
		}
		fmt.Println(newsecrets);
		wg.Add(len(newsecrets))
		for k, v := range newsecrets {
			go createUpdateSecret(basicClient, k, v, vault, &wg)
		}
		wg.Wait()
	}


	if *storefunckeys == true || *storecosmoskeys == true {
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
		authorizer, err = aauth.NewAuthorizerFromCLI()
	}
	if *storefunckeys == true {
		webclient := web.NewAppsClient(sub)
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
	if *storecosmoskeys == true {

		cosmosclient := documentdb.NewDatabaseAccountsClient(sub)
		cosmosclient.Authorizer = authorizer
		cosmosfile := "cosmos-secrets.yaml"
		cosmossecrets := cosmos.getConf(&cosmosfile)
		wg.Add(len(*cosmossecrets))
		for _, cosmos := range  *cosmossecrets {
			go createUpdateCosmosSecret(basicClient,cosmosclient,resourceGr,cosmos.AccountName,vault,cosmos.CosmosdbKeys,&wg)
		}
		wg.Wait()
	}

	if *storeeventgridkeys == true {

		eventclient := eventgrid.NewDomainsClient(sub)
		eventclient.Authorizer = authorizer
		eventfile := "eventgrid-secrets.yaml"
		eventsecrets := event.getConf(&eventfile)
		wg.Add(len(*eventsecrets))
		for _, event := range  *eventsecrets {
			go createUpdateEventGridSecret(basicClient,eventclient,resourceGr,event.DomainName,vault,event.AccessKeys,&wg)
		}
		wg.Wait()
	}

}
