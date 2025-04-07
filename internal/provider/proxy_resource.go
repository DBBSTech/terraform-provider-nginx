package nginx

import (
	"bufio"
	"context"
	"fmt"
	"strings"

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
var _ resource.Resource = &ProxyResource{}
var _ resource.ResourceWithImportState = &ProxyResource{}

func NewProxyResource() resource.Resource {
	return &ProxyResource{}
}

// ProxyResource defines the resource implementation.
type ProxyResource struct {
	client interface{} // Use interface{} to accept SSH client passed from provider.go
}

// ProxyResourceModel describes the resource data model.
type ProxyResourceModel struct {
	ServerName types.String `tfsdk:"server_name"`
	ListenPort types.Int64  `tfsdk:"listen_port"`
	Root       types.String `tfsdk:"root"`
	Path       types.String `tfsdk:"path"`
	Content    types.String `tfsdk:"content"`
	Id         types.String `tfsdk:"id"`
	ProxyName  types.String `tfsdk:"proxy_name"`
}

func (r *ProxyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_Proxy"
}

func (r *ProxyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Proxy resource",

		Attributes: map[string]schema.Attribute{
			"proxy_name": schema.StringAttribute{
				MarkdownDescription: "A unique name for the Proxy resource.",
				Required:            true, // User must define it
			},
			"server_name": schema.StringAttribute{
				MarkdownDescription: "The name of the server.",
				Optional:            true,
			},
			"listen_port": schema.Int64Attribute{
				MarkdownDescription: "The port the Proxy listens on.",
				Optional:            true,
			},
			"root": schema.StringAttribute{
				MarkdownDescription: "The root directory of the Proxy.",
				Optional:            true,
			},
			"path": schema.StringAttribute{
				MarkdownDescription: "The path of the Proxy Proxyuration file.",
				Optional:            true,
			},
			"content": schema.StringAttribute{
				MarkdownDescription: "The content of the Proxy.",
				Computed:            true,
				Optional:            true,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Proxy resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ProxyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ProxyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ProxyResourceModel

	// Retrieve the plan data
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build the NGINX server block content
	ProxyContent := fmt.Sprintf(`
	server {
		listen %d;
		server_name %s;

		root %s;
		index index.html;

		location / {
			try_files $uri $uri/ =404;
		}
	}`, data.ListenPort.ValueInt64(), data.ServerName.ValueString(), data.Root.ValueString())

	// Use SSH to write the content to the file
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

	command := fmt.Sprintf("echo '%s' | sudo tee %s > /dev/null", shellEscape(ProxyContent), data.Path.ValueString())

	if err := session.Run(command); err != nil {
		resp.Diagnostics.AddError(
			"Command Execution Error",
			fmt.Sprintf("Failed to execute command: %s", err),
		)
		return
	}

	// Set the resource ID to the Proxy_name
	data.Id = types.StringValue(data.ProxyName.ValueString())

	// Explicitly set the content
	data.Content = types.StringValue(ProxyContent)

	// Save the data into the Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, fmt.Sprintf("Created Proxy resource: %s", data.ProxyName.ValueString()))
}

func (r *ProxyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ProxyResourceModel

	// Retrieve the current state
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Use SSH client to verify the file existence and retrieve its content
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

	// Command to check file existence and read content
	checkCommand := fmt.Sprintf("if [ -f %s ]; then cat %s; else echo 'NOT_FOUND'; fi", data.Path.ValueString(), data.Path.ValueString())
	stdout, err := session.StdoutPipe()
	if err != nil {
		resp.Diagnostics.AddError(
			"SSH Pipe Error",
			fmt.Sprintf("Failed to create stdout pipe: %s", err),
		)
		return
	}

	if err := session.Start(checkCommand); err != nil {
		resp.Diagnostics.AddError(
			"SSH Command Execution Error",
			fmt.Sprintf("Failed to execute command: %s", err),
		)
		return
	}

	// Read the command output
	var result strings.Builder
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		result.WriteString(scanner.Text() + "\n")
	}

	if err := session.Wait(); err != nil {
		resp.Diagnostics.AddError(
			"SSH Command Error",
			fmt.Sprintf("Failed to complete command: %s", err),
		)
		return
	}

	// Handle 'NOT_FOUND' scenario
	if strings.TrimSpace(result.String()) == "NOT_FOUND" {
		resp.Diagnostics.AddWarning(
			"File Not Found",
			fmt.Sprintf("The file at path '%s' does not exist.", data.Path.ValueString()),
		)
		data.Content = types.StringNull()
	} else {
		data.Content = types.StringValue(result.String())
	}

	// Ensure the ID remains consistent
	data.Id = types.StringValue(data.ProxyName.ValueString())

	// Save the updated state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProxyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ProxyResourceModel
	var state ProxyResourceModel

	// Retrieve the updated plan data
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Retrieve the current state data
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build the updated NGINX Proxyuration
	updatedProxy := fmt.Sprintf(`
	server {
		listen %d;
		server_name %s;

		root %s;
		index index.html;

		location / {
			try_files $uri $uri/ =404;
		}
	}`, plan.ListenPort.ValueInt64(), plan.ServerName.ValueString(), plan.Root.ValueString())

	// Use SSH to update the file content
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

	command := fmt.Sprintf("echo '%s' | sudo tee %s > /dev/null", shellEscape(updatedProxy), plan.Path.ValueString())

	if err := session.Run(command); err != nil {
		resp.Diagnostics.AddError(
			"Command Execution Error",
			fmt.Sprintf("Failed to execute command: %s", err),
		)
		return
	}

	// Set the resource ID to the stable Proxy_name
	plan.Id = types.StringValue(plan.ProxyName.ValueString())

	// Set the content to the updated Proxyuration
	plan.Content = types.StringValue(updatedProxy)

	// Save the updated state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, fmt.Sprintf("Updated Proxy resource: %s", plan.ProxyName.ValueString()))
}

func (r *ProxyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ProxyResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete Proxy, got error: %s", err))
	//     return
	// }
}

func (r *ProxyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
