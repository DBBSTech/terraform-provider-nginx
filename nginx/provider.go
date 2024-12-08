package nginx

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type NginxProvider struct{}

type NginxProviderModel struct {
	Host     types.String `tfsdk:"host"`
	User     types.String `tfsdk:"user"`
	Password types.String `tfsdk:"password"`
}

func New() provider.Provider {
	return &NginxProvider{}
}

func (p *NginxProvider) Metadata(_ context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "nginx"
}

func (p *NginxProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = map[string]schema.Attribute{
		"host": {
			Type:        types.StringType,
			Required:    true,
			Description: "IP or hostname of the Debian host.",
		},
		"user": {
			Type:        types.StringType,
			Required:    true,
			Description: "SSH username for the Debian host.",
		},
		"password": {
			Type:        types.StringType,
			Required:    true,
			Sensitive:   true,
			Description: "SSH password for the Debian host.",
		},
	}
}

func (p *NginxProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewNginxConfigResource,
	}
}

func (p *NginxProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}
