package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/anthdm/ffaas/pkg/api"
	"github.com/anthdm/ffaas/pkg/config"
	"github.com/anthdm/ffaas/pkg/runtime"
	"github.com/anthdm/ffaas/pkg/storage"
	"github.com/anthdm/ffaas/pkg/types"
	"github.com/anthdm/ffaas/pkg/version"
	"github.com/google/uuid"
	"github.com/tetratelabs/wazero"
)

func main() {

	var (
		modCache    = storage.NewDefaultModCache()
		metricStore = storage.NewMemoryMetricStore()
		configFile  string
		seed        bool
	)
	flagSet := flag.NewFlagSet("ffaas", flag.ExitOnError)
	flagSet.StringVar(&configFile, "config", "ffaas.toml", "")
	flagSet.BoolVar(&seed, "seed", false, "")
	flagSet.Parse(os.Args[1:])

	err := config.Parse(configFile)
	if err != nil {
		log.Fatal(err)
	}

	store, err := storage.NewBoltStore(config.Get().StoragePath)
	if err != nil {
		log.Fatal(err)
	}

	if seed {
		seedEndpoint(store, modCache)
	}

	fmt.Println(banner())
	fmt.Println("The opensource faas platform powered by WASM")
	fmt.Println()
	server := api.NewServer(store, metricStore, modCache)
	fmt.Printf("api server running\t%s\n", config.GetApiUrl())
	log.Fatal(server.Listen(config.Get().APIServerAddr))

	// engine, err := actor.NewEngine(nil)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// eventPID := engine.SpawnFunc(func(c *actor.Context) {
	// 	switch msg := c.Message().(type) {
	// 	case actor.ActorInitializedEvent:
	// 		if strings.Contains(msg.PID.String(), "runtime") {
	// 			fmt.Println("got this", msg.PID)
	// 			engine.Stop(msg.PID)
	// 		}
	// 	}
	// }, "event")
	// engine.Subscribe(eventPID)

	// wasmServer := wasmhttp.NewServer(config.Get().WASMServerAddr, engine, memstore, metricStore, modCache)
	// fmt.Printf("wasm server running\t%s\n", config.GetWasmUrl())
	// log.Fatal(wasmServer.Listen())
}

func seedEndpoint(store storage.Store, cache storage.ModCacher) {
	b, err := os.ReadFile("examples/go/app.wasm")
	if err != nil {
		log.Fatal(err)
	}
	endpoint := &types.Endpoint{
		ID:          uuid.MustParse("09248ef6-c401-4601-8928-5964d61f2c61"),
		Name:        "Catfact parser",
		Environment: map[string]string{"FOO": "bar"},
		CreatedAT:   time.Now(),
	}

	deploy := types.NewDeploy(endpoint, b)
	endpoint.ActiveDeployID = deploy.ID
	endpoint.URL = config.GetWasmUrl() + "/" + endpoint.ID.String()
	endpoint.DeployHistory = append(endpoint.DeployHistory, deploy)
	store.CreateEndpoint(endpoint)
	store.CreateDeploy(deploy)
	fmt.Printf("endpoint: %s\n", endpoint.URL)

	compCache := wazero.NewCompilationCache()
	runtime.Compile(context.Background(), compCache, deploy.Blob)
	cache.Put(endpoint.ID, compCache)
}

func banner() string {
	return fmt.Sprintf(`
  __  __                
 / _|/ _|               
| |_| |_ __ _  __ _ ___ 
|  _|  _/ _  |/ _  / __|
| | | || (_| | (_| \__ \
|_| |_| \__,_|\__,_|___/ V%s
	`, version.Version)
}
