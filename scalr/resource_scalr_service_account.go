package scalr

import (
	"context"
	"errors"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/scalr/go-scalr"
	"log"
)

func resourceScalrServiceAccount() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceScalrServiceAccountCreate,
		ReadContext:   resourceScalrServiceAccountRead,
		UpdateContext: resourceScalrServiceAccountUpdate,
		DeleteContext: resourceScalrServiceAccountDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"email": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"status": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ValidateFunc: validation.StringInSlice(
					[]string{
						string(scalr.ServiceAccountStatusActive),
						string(scalr.ServiceAccountStatusInactive),
					},
					false,
				),
			},
			"account_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				DefaultFunc: scalrAccountIDDefaultFunc,
				ForceNew:    true,
			},
			"created_by": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"username": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"email": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"full_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func resourceScalrServiceAccountRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	scalrClient := meta.(*scalr.Client)
	id := d.Id()

	log.Printf("[DEBUG] Read service account: %s", id)
	sa, err := scalrClient.ServiceAccounts.Read(ctx, id)
	if err != nil {
		if errors.Is(err, scalr.ErrResourceNotFound) {
			log.Printf("[DEBUG] Service account %s not found", id)
			d.SetId("")
			return nil
		}
		return diag.Errorf("Error reading service account %s: %v", id, err)
	}

	// Update config.
	_ = d.Set("name", sa.Name)
	_ = d.Set("email", sa.Email)
	_ = d.Set("description", sa.Description)
	_ = d.Set("status", sa.Status)
	_ = d.Set("account_id", sa.Account.ID)

	var createdBy []interface{}
	if sa.CreatedBy != nil {
		createdBy = append(createdBy, map[string]interface{}{
			"username":  sa.CreatedBy.Username,
			"email":     sa.CreatedBy.Email,
			"full_name": sa.CreatedBy.FullName,
		})
	}
	_ = d.Set("created_by", createdBy)

	return nil
}

func resourceScalrServiceAccountCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	scalrClient := meta.(*scalr.Client)

	name := d.Get("name").(string)
	accountID := d.Get("account_id").(string)

	options := scalr.ServiceAccountCreateOptions{
		Name:    scalr.String(name),
		Account: &scalr.Account{ID: accountID},
	}

	if desc, ok := d.GetOk("description"); ok {
		options.Description = scalr.String(desc.(string))
	}

	if status, ok := d.GetOk("status"); ok {
		saStatus := scalr.ServiceAccountStatus(status.(string))
		options.Status = scalr.ServiceAccountStatusPtr(saStatus)
	}

	log.Printf("[DEBUG] Create service account %s in account %s", name, accountID)
	sa, err := scalrClient.ServiceAccounts.Create(ctx, options)
	if err != nil {
		return diag.Errorf(
			"Error creating service account %s in account %s: %v", name, accountID, err)
	}
	d.SetId(sa.ID)

	return resourceScalrServiceAccountRead(ctx, d, meta)
}

func resourceScalrServiceAccountUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	scalrClient := meta.(*scalr.Client)

	id := d.Id()

	options := scalr.ServiceAccountUpdateOptions{}

	if d.HasChange("description") {
		desc := d.Get("description").(string)
		options.Description = scalr.String(desc)
	}

	if d.HasChange("status") {
		status := scalr.ServiceAccountStatus(d.Get("status").(string))
		options.Status = scalr.ServiceAccountStatusPtr(status)
	}

	log.Printf("[DEBUG] Update service account %s", id)
	_, err := scalrClient.ServiceAccounts.Update(ctx, id, options)
	if err != nil {
		return diag.Errorf("error updating service account %s: %v", id, err)
	}

	return resourceScalrServiceAccountRead(ctx, d, meta)
}

func resourceScalrServiceAccountDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	scalrClient := meta.(*scalr.Client)
	id := d.Id()

	log.Printf("[DEBUG] Delete service account %s", id)
	err := scalrClient.ServiceAccounts.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, scalr.ErrResourceNotFound) {
			return nil
		}
		return diag.Errorf("Error deleting service account %s: %v", id, err)
	}

	return nil
}
