package main

import (
	"context"
	"flag"
	"log"

	"github.com/cloider/terraform-provider-cloider/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// Generates docs/ from the schema and examples/, which the registry renders.
//go:generate go tool tfplugindocs generate --provider-name cloider

// Set by goreleaser, see .goreleaser.yml
var version = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "run the provider with support for debuggers like delve")
	flag.Parse()

	err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/cloider/cloider",
		Debug:   debug,
	})
	if err != nil {
		log.Fatal(err.Error())
	}
}
