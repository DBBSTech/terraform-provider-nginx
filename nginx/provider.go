package nginx

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"host": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The hostname or IP address of the NGINX server.",
				DefaultFunc: schema.EnvDefaultFunc("NGINX_HOST", nil),
			},
			"user": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The SSH username to connect to the NGINX server.",
				DefaultFunc: schema.EnvDefaultFunc("NGINX_USER", nil),
			},
			"password": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The SSH password to connect to the NGINX server.",
				DefaultFunc: schema.EnvDefaultFunc("NGINX_PASSWORD", nil),
				Sensitive:   true,
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"nginx_config": resourceNginxConfig(),
		},
		ConfigureContextFunc: configureClient,
	}
}
