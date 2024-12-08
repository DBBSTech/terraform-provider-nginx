package main

import (
	"context"
	"terraform-provider-nginx/nginx"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/integralist/terraform-provider-mock/mock"
)

func main() {
	providerserver.Serve(context.Background(), nginx.New, providerserver.ServeOpts{})
	ProviderFunc: mock.Provider,
}
