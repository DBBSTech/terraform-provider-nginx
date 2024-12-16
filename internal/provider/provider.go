// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package nginx

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"golang.org/x/crypto/ssh"
)

// Ensure NginxProvider satisfies various provider interfaces.
var _ provider.Provider = &NginxProvider{}

//var _ provider.ProviderWithFunctions = &NginxProvider{}

// NginxProvider defines the provider implementation.
type NginxProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// NginxProviderModel describes the provider data model.
type NginxProviderModel struct {
	Host     types.String `tfsdk:"host"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

func (p *NginxProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "Nginx"
	resp.Version = p.version
}

func (p *NginxProvider) Schema(_ context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Optional: true,
			},
			"username": schema.StringAttribute{
				Optional: true,
			},
			"password": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
			},
		},
	}
}

// func (p *NginxProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
// 	var data NginxProviderModel

// 	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

// 	if resp.Diagnostics.HasError() {
// 		return
// 	}

// 	// Configuration values are now available.
// 	// if data.Endpoint.IsNull() { /* ... */ }

// 	// Example client configuration for data sources and resources
// 	client := http.DefaultClient
// 	resp.DataSourceData = client
// 	resp.ResourceData = client
// }

func (p *NginxProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Retrieve provider data from configuration
	var config NginxProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.
	host := os.Getenv("HOST")
	username := os.Getenv("USERNAME")
	password := os.Getenv("PASSWORD")

	if !config.Host.IsNull() {
		host = config.Host.ValueString()
	}

	if !config.Username.IsNull() {
		username = config.Username.ValueString()
	}

	if !config.Password.IsNull() {
		password = config.Password.ValueString()
	}

	// If any of the expected configurations are missing, return
	// errors with provider-specific guidance.
	if host == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Missing Host",
			"A valid hostname or IP is required.",
		)
	}

	if username == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Missing Username",
			"A valid username is required to connect to the host.",
		)
	}

	if password == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Missing Password",
			"A valid password is required to connect to the host.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Configure SSH client
	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Replace this with a secure callback in production
	}

	client, err := ssh.Dial("tcp", host+":22", sshConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to SSH to host",
			"An unexpected error occurred when creating the SSH connection. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"SSH Client Error: "+err.Error(),
		)
		return
	}

	// Make the SSH client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *NginxProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewSiteResource,
	}
}

func (p *NginxProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewExampleDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &NginxProvider{
			version: version,
		}
	}
}
