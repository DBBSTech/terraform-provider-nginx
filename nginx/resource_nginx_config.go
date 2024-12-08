package nginx

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceNginxConfig() *schema.Resource {
	return &schema.Resource{
		CreateContext: createConfig,
		ReadContext:   readConfig,
		UpdateContext: updateConfig,
		DeleteContext: deleteConfig,
		Schema: map[string]*schema.Schema{
			"server_name": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The server name for the NGINX configuration.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"listen_port": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The port number to listen on.",
			},
			"root": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The root directory for the NGINX server.",
			},
			"config_path": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The path to the generated NGINX configuration file.",
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func createConfig(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Retrieve input values
	serverName := d.Get("server_name").(string)
	listenPort := d.Get("listen_port").(int)
	root := d.Get("root").(string)

	// Generate NGINX configuration content
	configContent := fmt.Sprintf(`
server {
    listen %d;
    server_name %s;

    root %s;
    index index.html;

    location / {
        try_files $uri $uri/ =404;
    }
}`, listenPort, serverName, root)

	// Define remote config path
	configPath := fmt.Sprintf("/etc/nginx/sites-available/%s.conf", serverName)

	// Placeholder: Replace with actual logic to write the file via SSH
	err := uploadConfig(configPath, configContent)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to upload NGINX config: %s", err))
	}

	// Set resource ID and other attributes
	d.SetId(serverName)
	_ = d.Set("config_path", configPath)

	return nil
}

func readConfig(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Retrieve config path
	configPath := d.Get("config_path").(string)

	// Placeholder: Replace with actual logic to read the file via SSH
	exists, err := configExists(configPath)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to check if NGINX config exists: %s", err))
	}

	if !exists {
		// If config doesn't exist, remove from state
		d.SetId("")
		return nil
	}

	// Update state with current values
	return nil
}

func updateConfig(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChanges("server_name", "listen_port", "root") {
		// Recreate the configuration file
		return createConfig(ctx, d, meta)
	}
	return nil
}

func deleteConfig(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Retrieve config path
	configPath := d.Get("config_path").(string)

	// Placeholder: Replace with actual logic to delete the file via SSH
	err := deleteConfigFile(configPath)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete NGINX config: %s", err))
	}

	d.SetId("")
	return nil
}

// Placeholder function to upload NGINX configuration via SSH
func uploadConfig(path, content string) error {
	// Implement your logic to connect to the server and upload the configuration
	fmt.Printf("Uploading config to %s:\n%s\n", path, content)
	return nil
}

// Placeholder function to check if a configuration file exists via SSH
func configExists(path string) (bool, error) {
	// Implement your logic to connect to the server and verify if the file exists
	fmt.Printf("Checking if config exists at %s\n", path)
	return true, nil
}

// Placeholder function to delete a configuration file via SSH
func deleteConfigFile(path string) error {
	// Implement your logic to connect to the server and delete the configuration
	fmt.Printf("Deleting config at %s\n", path)
	return nil
}
