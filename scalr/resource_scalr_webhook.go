package scalr

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scalr/go-scalr"
)

var (
	eventDefinitions = map[string]bool{
		"run:completed":       true,
		"run:errored":         true,
		"run:needs_attention": true,
	}
)

func resourceScalrWebhook() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceScalrWebhookCreate,
		ReadContext:   resourceScalrWebhookRead,
		UpdateContext: resourceScalrWebhookUpdate,
		DeleteContext: resourceScalrWebhookDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"last_triggered_at": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"events": {
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Required: true,
			},

			"endpoint_id": {
				Type:     schema.TypeString,
				Required: true,
			},

			"workspace_id": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"environment_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

// remove after https://scalr-labs.atlassian.net/browse/SCALRCORE-16234
func getResourceScope(ctx context.Context, scalrClient *scalr.Client, workspaceID string, environmentID string) (*scalr.Workspace, *scalr.Environment, *scalr.Account, error) {

	// Resource scope
	var workspace *scalr.Workspace
	var environment *scalr.Environment
	var account *scalr.Account

	// Get the workspace.
	if workspaceID != "" {
		var err error
		workspace, err = scalrClient.Workspaces.ReadByID(ctx, workspaceID)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("Error retrieving workspace %s: %v", workspaceID, err)
		}

		if environmentID != "" && environmentID != workspace.Environment.ID {
			return nil, nil, nil, fmt.Errorf("Workspace %s does not belong to an environment %s", workspaceID, environmentID)
		}

		environmentID = workspace.Environment.ID
	}

	// Get the environment.
	if environmentID != "" {
		var err error
		environment, err = scalrClient.Environments.Read(ctx, environmentID)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("Error retrieving environment %s: %v", environmentID, err)
		}
		account = environment.Account
	} else {
		return nil, nil, nil, fmt.Errorf("Missing workspace_id or environment_id")
	}

	return workspace, environment, account, nil
}

func validateEventDefinitions(eventName string) error {
	if val, ok := eventDefinitions[eventName]; ok && val {
		return nil
	}
	i := 0
	eventDefinitionsQuoted := make([]string, len(eventDefinitions))
	for eventDefinition := range eventDefinitions {
		eventDefinitionsQuoted[i] = fmt.Sprintf("'%s'", eventDefinition)
		i++
	}
	return fmt.Errorf(
		"Invalid value for events '%s'. Allowed values: %s", eventName, strings.Join(eventDefinitionsQuoted, ", "))
}

func parseEventDefinitions(d *schema.ResourceData) ([]*scalr.EventDefinition, error) {
	eventDefinitions := make([]*scalr.EventDefinition, 0)

	eventIds := d.Get("events").([]interface{})
	err := ValidateIDsDefinitions(eventIds)
	if err != nil {
		return nil, fmt.Errorf("Got error during parsing events: %s", err.Error())
	}

	for _, eventID := range eventIds {
		id := eventID.(string)
		if err := validateEventDefinitions(id); err != nil {
			return nil, err
		}
		eventDefinitions = append(eventDefinitions, &scalr.EventDefinition{ID: id})
	}

	return eventDefinitions, nil
}

func resourceScalrWebhookCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	scalrClient := meta.(*scalr.Client)

	// Get attributes.
	name := d.Get("name").(string)
	endpointID := d.Get("endpoint_id").(string)
	workspaceID := d.Get("workspace_id").(string)
	environmentID := d.Get("environment_id").(string)

	workspace, environment, account, err := getResourceScope(ctx, scalrClient, workspaceID, environmentID)
	if err != nil {
		return diag.FromErr(err)
	}

	eventDefinitions, err := parseEventDefinitions(d)
	if err != nil {
		return diag.FromErr(err)
	}

	// Create a new options struct.
	options := scalr.WebhookCreateOptions{
		Name:        scalr.String(name),
		Enabled:     scalr.Bool(d.Get("enabled").(bool)),
		Events:      eventDefinitions,
		Endpoint:    &scalr.Endpoint{ID: endpointID},
		Workspace:   workspace,
		Environment: environment,
		Account:     account,
	}

	log.Printf("[DEBUG] Create webhook: %s", name)
	webhook, err := scalrClient.Webhooks.Create(ctx, options)
	if err != nil {
		return diag.Errorf("Error creating webhook %s: %v", name, err)
	}

	d.SetId(webhook.ID)

	return resourceScalrWebhookRead(ctx, d, meta)
}

func resourceScalrWebhookRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	scalrClient := meta.(*scalr.Client)

	// Get the ID
	webhookID := d.Id()

	log.Printf("[DEBUG] Read endpoint with ID: %s", webhookID)
	webhook, err := scalrClient.Webhooks.Read(ctx, webhookID)
	if err != nil {
		if errors.Is(err, scalr.ErrResourceNotFound) {
			return diag.Errorf("Could not find webhook %s: %v", webhookID, err)
		}
		return diag.Errorf("Error retrieving webhook: %v", err)
	}

	// Update the config.
	_ = d.Set("name", webhook.Name)
	_ = d.Set("enabled", webhook.Enabled)
	_ = d.Set("last_triggered_at", webhook.LastTriggeredAt)

	events := make([]string, 0)
	if webhook.Events != nil {
		for _, event := range webhook.Events {
			events = append(events, event.ID)
		}
	}
	_ = d.Set("events", events)

	if webhook.Workspace != nil {
		_ = d.Set("workspace_id", webhook.Workspace.ID)
	}
	if webhook.Environment != nil {
		_ = d.Set("environment_id", webhook.Environment.ID)
	}
	if webhook.Endpoint != nil {
		_ = d.Set("endpoint_id", webhook.Endpoint.ID)
	}

	return nil
}

func resourceScalrWebhookUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	scalrClient := meta.(*scalr.Client)

	eventDefinitions, err := parseEventDefinitions(d)
	if err != nil {
		return diag.FromErr(err)
	}

	// Create a new options struct.
	options := scalr.WebhookUpdateOptions{
		Name:     scalr.String(d.Get("name").(string)),
		Enabled:  scalr.Bool(d.Get("enabled").(bool)),
		Events:   eventDefinitions,
		Endpoint: &scalr.Endpoint{ID: d.Get("endpoint_id").(string)},
	}

	log.Printf("[DEBUG] Update webhook: %s", d.Id())
	_, err = scalrClient.Webhooks.Update(ctx, d.Id(), options)
	if err != nil {
		return diag.Errorf("Error updating webhook %s: %v", d.Id(), err)
	}

	return resourceScalrWebhookRead(ctx, d, meta)
}

func resourceScalrWebhookDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	scalrClient := meta.(*scalr.Client)

	log.Printf("[DEBUG] Delete webhook: %s", d.Id())
	err := scalrClient.Webhooks.Delete(ctx, d.Id())
	if err != nil {
		if errors.Is(err, scalr.ErrResourceNotFound) {
			return nil
		}
		return diag.Errorf("Error deleting webhook %s: %v", d.Id(), err)
	}

	return nil
}
