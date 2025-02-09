package scalr

import (
	"context"
	"errors"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scalr/go-scalr"
)

func dataSourceScalrEndpoint() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceScalrEndpointRead,

		Schema: map[string]*schema.Schema{

			"id": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				AtLeastOneOf: []string{"name"},
			},

			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ConflictsWith: []string{"id"},
			},

			"max_attempts": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"secret_key": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},

			"url": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"timeout": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"account_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				DefaultFunc: scalrAccountIDDefaultFunc,
			},

			"environment_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceScalrEndpointRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	scalrClient := meta.(*scalr.Client)

	// Get the ID
	endpointID := d.Get("id").(string)
	endpointName := d.Get("name").(string)

	accountID := d.Get("account_id").(string)

	var endpoint *scalr.Endpoint
	var err error

	if endpointID != "" {
		log.Printf("[DEBUG] Read endpoint with ID: %s", endpointID)
		endpoint, err = scalrClient.Endpoints.Read(ctx, endpointID)
	} else {
		log.Printf("[DEBUG] Read configuration of endpoint: %s", endpointName)
		options := GetEndpointByNameOptions{
			Name:    &endpointName,
			Account: &accountID,
		}
		endpoint, err = GetEndpointByName(ctx, options, scalrClient)
	}

	if err != nil {
		if errors.Is(err, scalr.ErrResourceNotFound) {
			return diag.Errorf("Could not find endpoint %s: %v", endpointID, err)
		}
		return diag.Errorf("Error retrieving endpoint: %v", err)
	}

	// Update the config.
	_ = d.Set("name", endpoint.Name)
	_ = d.Set("timeout", endpoint.Timeout)
	_ = d.Set("max_attempts", endpoint.MaxAttempts)
	_ = d.Set("secret_key", endpoint.SecretKey)
	_ = d.Set("url", endpoint.Url)
	if endpoint.Environment != nil {
		_ = d.Set("environment_id", endpoint.Environment.ID)
	}
	d.SetId(endpoint.ID)

	return nil
}
