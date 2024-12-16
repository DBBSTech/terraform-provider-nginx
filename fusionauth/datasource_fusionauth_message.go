package fusionauth

import (
	"context"
	"net/http"

	"github.com/FusionAuth/go-client/pkg/fusionauth"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceMessage() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceMessageRead,
		Schema: map[string]*schema.Schema{
			"message_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ExactlyOneOf: []string{"message_id", "name"},
				Description:  "The unique ID of the message.",
				ValidateFunc: validation.IsUUID,
			},
			"name": {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{"message_id", "name"},
				Description:  "The unique name of the message.",
			},
			"data": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "An object containing the data of the message.",
			},
			"language": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The language of the message.",
			},
			"template": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The template of the message.",
			},
		},
	}
}

func dataSourceMessageRead(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	client := i.(Client)

	var searchTerm string
	var res *fusionauth.MessageTemplateResponse
	var err error

	// Either `message_id` or `name` are guaranteed to be set
	if messageID, ok := data.GetOk("message_id"); ok {
		searchTerm = messageID.(string)
		res, err = client.FAClient.RetrieveMessageTemplate(searchTerm)
	} else {
		searchTerm = data.Get("name").(string)
		res, err = client.FAClient.RetrieveMessageTemplates()
	}
	if err != nil {
		return diag.FromErr(err)
	}
	if res.StatusCode == http.StatusNotFound {
		return diag.Errorf("couldn't find message '%s'", searchTerm)
	}
	if err := checkResponse(res.StatusCode, nil); err != nil {
		return diag.FromErr(err)
	}

	var foundEntity fusionauth.MessageTemplate
	if len(res.MessageTemplates) > 0 {
		// Search based on name
		var found = false
		for _, entity := range res.MessageTemplates {
			if entity.Name == searchTerm {
				found = true
				foundEntity = entity
				break
			}
		}
		if !found {
			return diag.Errorf("couldn't find message with name '%s'", searchTerm)
		}
	} else {
		foundEntity = *res.MessageTemplate
	}

	data.SetId(foundEntity.Id)
	return buildResourceDataFromMessage(data, foundEntity)
}

func buildResourceDataFromMessage(data *schema.ResourceData, entity fusionauth.MessageTemplate) diag.Diagnostics {
	if err := data.Set("name", entity.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := data.Set("data", entity.Data); err != nil {
		return diag.FromErr(err)
	}
	if err := data.Set("language", entity.DefaultLanguage); err != nil {
		return diag.FromErr(err)
	}
	if err := data.Set("template", entity.Template); err != nil {
		return diag.FromErr(err)
	}
	return nil
}
