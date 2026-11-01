// Command terraform-provider-zinn is the Zinn Digital Terraform provider.
package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/Zinn-Digital/terraform-provider-zinn/internal/provider"
)

// version is set at release time with -ldflags "-X main.version=<semver>".
var version = "dev"

func main() {
	debug := flag.Bool("debug", false, "run with support for debuggers like delve")
	flag.Parse()
	err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/zinn-digital/zinn",
		Debug:   *debug,
	})
	if err != nil {
		log.Fatal(err.Error())
	}
}
