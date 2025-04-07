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
var _ resource.Resource = &APIResource{}
var _ resource.ResourceWithImportState = &APIResource{}

func NewAPIResource() resource.Resource {
	return &APIResource{}
}

// APIResource defines the resource implementation.
type APIResource struct {
	client interface{} // Use interface{} to accept SSH client passed from provider.go
}

// APIResourceModel describes the resource data model.
type APIResourceModel struct {
	ServerName types.String `tfsdk:"server_name"`
	ListenPort types.Int64  `tfsdk:"listen_port"`
	Root       types.String `tfsdk:"root"`
	Path       types.String `tfsdk:"path"`
	Content    types.String `tfsdk:"content"`
	Id         types.String `tfsdk:"id"`
	APIName    types.String `tfsdk:"api_name"`
}

func (r *APIResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api" // Changed to lowercase "_api"
}

func (r *APIResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "API resource",

		Attributes: map[string]schema.Attribute{
			"api_name": schema.StringAttribute{
				MarkdownDescription: "A unique name for the API resource.",
				Required:            true, // User must define it
			},
			"server_name": schema.StringAttribute{
				MarkdownDescription: "The name of the server.",
				Optional:            true,
			},
			"listen_port": schema.Int64Attribute{
				MarkdownDescription: "The port the API listens on.",
				Optional:            true,
			},
			"root": schema.StringAttribute{
				MarkdownDescription: "The root directory of the API.",
				Optional:            true,
			},
			"path": schema.StringAttribute{
				MarkdownDescription: "The path of the API configuration file.",
				Optional:            true,
			},
			"content": schema.StringAttribute{
				MarkdownDescription: "The content of the API.",
				Computed:            true,
				Optional:            true,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the API resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *APIResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *APIResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data APIResourceModel

	// Retrieve the plan data
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build the NGINX server block content
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

	command := fmt.Sprintf("echo '%s' | sudo tee %s > /dev/null", shellEscape(configContent), data.Path.ValueString())

	if err := session.Run(command); err != nil {
		resp.Diagnostics.AddError(
			"Command Execution Error",
			fmt.Sprintf("Failed to execute command: %s", err),
		)
		return
	}

	// Set the resource ID to the API_name
	data.Id = types.StringValue(data.APIName.ValueString())

	// Explicitly set the content
	data.Content = types.StringValue(configContent)

	// Save the data into the Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, fmt.Sprintf("Created API resource: %s", data.APIName.ValueString()))
}

func (r *APIResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data APIResourceModel

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
	data.Id = types.StringValue(data.APIName.ValueString())

	// Save the updated state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *APIResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan APIResourceModel
	var state APIResourceModel

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

	// Build the updated NGINX configuration
	updatedConfig := fmt.Sprintf(`
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

	command := fmt.Sprintf("echo '%s' | sudo tee %s > /dev/null", shellEscape(updatedConfig), plan.Path.ValueString())

	if err := session.Run(command); err != nil {
		resp.Diagnostics.AddError(
			"Command Execution Error",
			fmt.Sprintf("Failed to execute command: %s", err),
		)
		return
	}

	// Set the resource ID to the stable API_name
	plan.Id = types.StringValue(plan.APIName.ValueString())

	// Set the content to the updated configuration
	plan.Content = types.StringValue(updatedConfig)

	// Save the updated state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, fmt.Sprintf("Updated API resource: %s", plan.APIName.ValueString()))
}

func (r *APIResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data APIResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Use SSH to delete the configuration file
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

	command := fmt.Sprintf("sudo rm -f %s", data.Path.ValueString())
	if err := session.Run(command); err != nil {
		resp.Diagnostics.AddError(
			"Command Execution Error",
			fmt.Sprintf("Failed to delete file at %s: %s", data.Path.ValueString(), err),
		)
		return
	}

	tflog.Trace(ctx, fmt.Sprintf("Deleted API resource: %s", data.APIName.ValueString()))
}

func (r *APIResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by path or name
	idParts := strings.Split(req.ID, ":")
	if len(idParts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Expected import ID in format 'api_name:path'",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("api_name"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("path"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), idParts[0])...)
}
