package sql

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform-provider-google/google/tpgresource"
	transport_tpg "github.com/hashicorp/terraform-provider-google/google/transport"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	sqladmin "google.golang.org/api/sqladmin/v1beta4"
)

func ResourceSqlProvisionScript() *schema.Resource {
	return &schema.Resource{
		Create: resourceSqlProvisionScriptCreate,
		Read:   resourceSqlProvisionScriptRead,
		Update: resourceSqlProvisionScriptUpdate,
		Delete: resourceSqlProvisionScriptDelete,
		CustomizeDiff: customdiff.All(
			tpgresource.DefaultProviderProject,
		),

		SchemaVersion: 1,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: `The name of the provision script.`,
			},

			"script": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				Description: `The SQL script to provision database resources. Its execution timeout is 5 minutes.
				Changing this forces the script to be rerun. Make sure the script is idempotent.
				You can use statements like "create if not exists …" or
				"if not exists (select …) then … end if" to prevent errors caused by duplicate resources.`,
			},

			"instance": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: `The name of the Cloud SQL instance. Changing this forces the script to be run on the new instance.`,
			},

			"database": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Description: `The name of the database on which the SQL script is executed. This is required for Postgres
				instances. Changing this forces the script to be run using this database.`,
			},

			"deletion_policy": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "ABANDON",
				Description: `The deletion policy for the resources created by the script. The default is "ABANDON".
				It must be "ABANDON" to allow Terraform to abandon the resources. You can delete them by adding statements
				in the script such as "drop … if exists" if necessary.`,
				ValidateFunc: validation.StringInSlice([]string{"ABANDON"}, false),
			},
		},
		UseJSONNumber: true,
	}
}

func resourceSqlProvisionScriptCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*transport_tpg.Config)
	userAgent, err := tpgresource.GenerateUserAgentString(d, config.UserAgent)
	if err != nil {
		return err
	}

	project, err := tpgresource.GetProject(d, config)
	if err != nil {
		return err
	}

	name := d.Get("name").(string)
	instance := d.Get("instance").(string)
	script := d.Get("script").(string)

	var database string
	if db, ok := d.GetOk("database"); ok {
		database = db.(string)
	}

	executeSqlPayload := &sqladmin.ExecuteSqlPayload{
		SqlStatement: script,
		Database:     database,
		AutoIamAuthn: true,
	}

	transport_tpg.MutexStore.Lock(instanceMutexKey(project, instance))
	defer transport_tpg.MutexStore.Unlock(instanceMutexKey(project, instance))

	var databaseInstance *sqladmin.DatabaseInstance
	err = transport_tpg.Retry(transport_tpg.RetryOptions{
		RetryFunc: func() (rerr error) {
			databaseInstance, rerr = config.NewSqlAdminClient(userAgent).Instances.Get(project, instance).Do()
			return rerr
		},
		Timeout:              d.Timeout(schema.TimeoutRead),
		ErrorRetryPredicates: []transport_tpg.RetryErrorPredicateFunc{transport_tpg.IsSqlOperationInProgressError},
	})
	if err != nil {
		return transport_tpg.HandleNotFoundError(err, d, fmt.Sprintf("SQL Database Instance %q", d.Get("instance").(string)))
	}
	if databaseInstance.Settings.ActivationPolicy != "ALWAYS" {
		return fmt.Errorf("Error, failed to run script %s because instance %s is not up", name, instance)
	}

	log.Printf("[INFO] executing script %s on database %s on instance %s", name, database, instance)

	var resp *sqladmin.SqlInstancesExecuteSqlResponse
	resp, err = config.NewSqlAdminClient(userAgent).Instances.ExecuteSql(project, instance,
		executeSqlPayload).Do()

	if err != nil {
		return fmt.Errorf("Error, failed to run script %s on instance %s: %s", name, instance, err)
	}
	log.Printf("[INFO] response from the execution of script %s on instance %s: %s", name, instance, resp)

	d.SetId(fmt.Sprintf("%s/%s", instance, name))
	return nil
}

func resourceSqlProvisionScriptRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceSqlProvisionScriptUpdate(d *schema.ResourceData, meta interface{}) error {
	name := d.Get("name").(string)
	instance := d.Get("instance").(string)
	d.SetId(fmt.Sprintf("%s/%s", instance, name))
	return nil
}

func resourceSqlProvisionScriptDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}
