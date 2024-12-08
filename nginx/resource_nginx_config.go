package nginx

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type NginxConfigResource struct{}

func NewNginxConfigResource() resource.Resource {
	return &NginxConfigResource{}
}

type NginxConfigModel struct {
	ID         types.String `tfsdk:"id"`
	ServerName types.String `tfsdk:"server_name"`
	ListenPort types.Int64  `tfsdk:"listen_port"`
	Root       types.String `tfsdk:"root"`
	Upstreams  types.List   `tfsdk:"upstreams"`
	ConfigPath types.String `tfsdk:"config_path"`
}

func (r *NginxConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "nginx_config"
}

func (r *NginxConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"server_name": schema.StringAttribute{
				Required:    true,
				Description: "The server name for the NGINX config.",
			},
			"listen_port": schema.Int64Attribute{
				Required:    true,
				Description: "Port to listen on.",
			},
			"root": schema.StringAttribute{
				Required:    true,
				Description: "Root directory for the server.",
			},
			"upstreams": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "List of upstream servers for load balancing.",
			},
			"config_path": schema.StringAttribute{
				Computed:    true,
				Description: "Path to the generated NGINX configuration file.",
			},
		},
	}
}

func (r *NginxConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NginxConfigModel
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	configContent := fmt.Sprintf(`
server {
    listen %d;
    server_name %s;
    
    root %s;
    index index.html;

    location / {
        try_files $uri $uri/ =404;
    }
}`, data.ListenPort.ValueInt64(), data.ServerName.ValueString(), data.Root.ValueString())

	// Placeholder: Implement SSH connection and write the file
	remotePath := fmt.Sprintf("/etc/nginx/sites-available/%s.conf", data.ServerName.ValueString())
	err := uploadConfig(data.Host.ValueString(), data.User.ValueString(), data.Password.ValueString(), remotePath, configContent)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create config", err.Error())
		return
	}

	data.ConfigPath = types.StringValue(remotePath)
	data.ID = types.StringValue(data.ServerName.ValueString())
	resp.State.Set(ctx, &data)
}

// Implement Read, Update, and Delete methods similarly.
