package nginx

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"golang.org/x/crypto/ssh"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &SiteResource{}
var _ resource.ResourceWithImportState = &SiteResource{}

func NewSiteResource() resource.Resource {
	return &SiteResource{}
}

// SiteResource defines the resource implementation.
type SiteResource struct {
	client interface{} // Use interface{} to accept SSH client passed from provider.go
}

// SiteResourceModel describes the resource data model.
type SiteResourceModel struct {
	ServerName types.String `tfsdk:"server_name"`
	ListenPort types.Int64  `tfsdk:"listen_port"`
	Root       types.String `tfsdk:"root"`
	Path       types.String `tfsdk:"path"`
	Content    types.String `tfsdk:"content"`
	Id         types.String `tfsdk:"id"`
}

func (r *SiteResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_site"
}

func (r *SiteResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Site resource",

		Attributes: map[string]schema.Attribute{
			"server_name": schema.StringAttribute{
				MarkdownDescription: "The name of the server.",
				Optional:            true,
			},
			"listen_port": schema.Int64Attribute{
				MarkdownDescription: "The port the site listens on.",
				Optional:            true,
			},
			"root": schema.StringAttribute{
				MarkdownDescription: "The root directory of the site.",
				Optional:            true,
			},
			"path": schema.StringAttribute{
				MarkdownDescription: "The path of the site.",
				Optional:            true,
			},
			"content": schema.StringAttribute{
				MarkdownDescription: "The content of the site.",
				Optional:            true,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the site resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *SiteResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Use the SSH client passed from the provider
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*ssh.Client) // Type assertion to retrieve the SSH client

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ssh.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *SiteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SiteResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Use the SSH client to create the resource on the server
	sshClient := r.client.(*ssh.Client)

	// Example: Execute a command on the remote server
	session, err := sshClient.NewSession()
	if err != nil {
		resp.Diagnostics.AddError(
			"SSH Session Error",
			fmt.Sprintf("Failed to create SSH session: %s", err),
		)
		return
	}
	defer session.Close()

	command := fmt.Sprintf("echo '%s' > %s", data.Content.ValueString(), data.Path.ValueString())
	if err := session.Run(command); err != nil {
		resp.Diagnostics.AddError(
			"Command Execution Error",
			fmt.Sprintf("Failed to execute command: %s", err),
		)
		return
	}

	// Set the resource ID to a unique identifier
	data.Id = types.StringValue(fmt.Sprintf("%s:%d", data.ServerName.ValueString(), data.ListenPort.ValueInt64()))

	tflog.Trace(ctx, "created a site resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SiteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SiteResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Use the SSH client to read the resource state from the server
	sshClient := r.client.(*ssh.Client)

	session, err := sshClient.NewSession()
	if err != nil {
		resp.Diagnostics.AddError(
			"SSH Session Error",
			fmt.Sprintf("Failed to create SSH session: %s", err),
		)
		return
	}
	defer session.Close()

	// Example: Check if the file exists on the server
	command := fmt.Sprintf("test -f %s", data.Path.ValueString())
	if err := session.Run(command); err != nil {
		resp.Diagnostics.AddError(
			"Resource Not Found",
			fmt.Sprintf("The resource at path '%s' does not exist: %s", data.Path.ValueString(), err),
		)
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// func (r *SiteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
// 	var data SiteResourceModel

// 	// Read Terraform plan data into the model
// 	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
// 	if resp.Diagnostics.HasError() {
// 		return
// 	}

// 	// Use the SSH client to update the resource on the server
// 	sshClient := r.client.(*ssh.Client)

// 	session, err := sshClient.NewSession()
// 	if err != nil {
// 		resp.Diagnostics.AddError(
// 			"SSH Session Error",
// 			fmt.Sprintf("Failed to create SSH session: %s", err),
// 		)
// 		return
// 	}
// 	defer session.Close()

// 	command := fmt.Sprintf("echo '%s' > %s", data.Content.ValueString(), data.Path.ValueString())
// 	if err := session.Run(command); err != nil {
// 		resp.Diagnostics.AddError(
// 			"Command Execution Error",
// 			fmt.Sprintf("Failed to execute command: %s", err),
// 		)
// 		return
// 	}

// 	tflog.Trace(ctx, "updated a site resource")

// 	// Save updated data into Terraform state
// 	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
// }

// func (r *SiteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
// 	var data SiteResourceModel

// 	// Read Terraform prior state data into the model
// 	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
// 	if resp.Diagnostics.HasError() {
// 		return
// 	}

// 	// Use the SSH client to delete the resource from the server
// 	sshClient := r.client.(*ssh.Client)

// 	session, err := sshClient.NewSession()
// 	if err != nil {
// 		resp.Diagnostics.AddError(
// 			"SSH Session Error",
// 			fmt.Sprintf("Failed to create SSH session: %s", err),
// 		)
// 		return
// 	}
// 	defer session.Close()

// 	command := fmt.Sprintf("rm -f %s", data.Path.ValueString())
// 	if err := session.Run(command); err != nil {
// 		resp.Diagnostics.AddError(
// 			"Command Execution Error",
// 			fmt.Sprintf("Failed to execute command: %s", err),
// 		)
// 		return
// 	}

// 	tflog.Trace(ctx, "deleted a site resource")
// }

func (r *SiteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SiteResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update Site, got error: %s", err))
	//     return
	// }

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SiteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SiteResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete Site, got error: %s", err))
	//     return
	// }
}

func (r *SiteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
