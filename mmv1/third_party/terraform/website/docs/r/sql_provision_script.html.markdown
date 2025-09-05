---
subcategory: "Cloud SQL"
description: |-
  Executes a SQL script to provision in-database resources in Google Cloud SQL.
---

# google_sql_provision_script

~> **Warning:** This resource is in beta, and should be used with the terraform-provider-google-beta provider.
See [Provider Versions](https://terraform.io/docs/providers/google/guides/provider_versions.html) for more details on beta resources. The SQL script and its execution response might transit through intermediate locations between your client and the location of the target instance.

Executes a SQL script to provision in-database resources on a Cloud SQL Instance. For more information, see the [official documentation](https://cloud.google.com/sql/), or the [JSON API](https://cloud.google.com/sql/docs/admin-api/v1beta4/instances/executeSql).

~> **Note:** Terraform connects to the instance via [IAM database authentication](https://cloud.google.com/sql/docs/mysql/authentication) so the GCP account in use must exist as an IAM user in the instance. You also need to grant roles or privileges to this IAM user so it has permission to execute statements in your provision scripts. You may need to directly connect to the instance to grant roles or privileges to the IAM user because it's not supported via Terraform yet.


## Example Usage

Example managing a Cloud SQL instance with a provision script.

```hcl
resource "google_sql_database_instance" "instance" {
  name             = "my-instance"
  database_version = "POSTGRES_17"

  settings {
    tier            = "db-f1-micro"
    data_api_access = "ALLOW_DATA_API"
    database_flags {
      name  = "cloudsql.iam_authentication"
      # For MySQL, the flag name is "cloudsql_iam_authentication"
      value = "on"
    }
  }
}

resource "google_sql_user" "iam_user" {
  name     = "gcp-account-used-by-terraform@example.com"
  instance = google_sql_database_instance.instance.name
  type     = "CLOUD_IAM_USER"
}

resource "google_sql_database" "database" {
  name     = "my-database"
  instance = google_sql_database_instance.instance.name
}

resource "google_sql_provision_script" "extensions" {
  script  = "CREATE TABLE IF NOT EXISTS table1 ( col VARCHAR(16) NOT NULL );"
  database = google_sql_database.database.name
  instance = google_sql_database_instance.instance.name
  depends_on = [google_sql_user.iam_user]
}

resource "google_sql_provision_script" "tables" {
  script  = "file("${path.module}/tables.sql")
  database = google_sql_database.database.name
  instance = google_sql_database_instance.instance.name
  depends_on = [google_sql_user.iam_user]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the provision script.

* `script` - (Required) The SQL script to provision database resources. Its execution timeout
    is 5 minutes and it will be canceled if it takes longer than 5 minutes. Setting a higher
    timeout using `SET SESSION MAX_EXECUTION_TIME` isn't supported. For Cloud SQL for MySQL 5.6
    and 5.7, long running DDL statements timing out may cause orphaned files or tables that can't
    be safely rolled back. Be cautious with queries like ALTER TABLE on large tables.
    Changing this field forces the script to be rerun. Make sure the script is idempotent or
    safe to run multiple times. You can use patterns like `create if not exists …` or
    `if not exists (select …) then … end if` to avoid errors. If it's not possible to make a
    statement idempotent, you can run it once and then remove it from this script.

* `instance` - (Required) The name of the Cloud SQL instance. Changing this forces the script to
    be run on the new instance.

* `database` - (Required) The name of the database on which the SQL script is executed. This is
    required for Postgres instances. Changing this forces the script to be run using this database.

* `deletion_policy` - (Optional) The deletion policy for the resources created by the script. The
    default is "ABANDON". It must be "ABANDON" to allow Terraform to abandon the resources. You can
    delete them by adding statements in the script such as `drop … if exists` if necessary.

## Attributes Reference

Only the arguments listed above are exposed as attributes.

## Timeouts

This resource provides the following
[Timeouts](https://developer.hashicorp.com/terraform/plugin/sdkv2/resources/retries-and-customizable-timeouts) configuration options: configuration options:

- `create` - Default is 20 minutes. Note the provider has its own timeout too -- the provision script execution must finish in 5 minutes or it will be canceled by the database. 

