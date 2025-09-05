package sql_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-provider-google/google/acctest"
)

func TestAccSqlProvisionScriptMySql(t *testing.T) {
	t.Parallel()

	instance := fmt.Sprintf("tf-test-%d", acctest.RandInt(t))
	scriptName := fmt.Sprintf("tf-test-%d", acctest.RandInt(t))
	script := "CREATE USER IF NOT EXISTS 'user'@'%' IDENTIFIED BY RANDOM PASSWORD; GRANT SELECT ON *.* to 'user'@'%';"
	acctest.VcrTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.AccTestPreCheck(t) },
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories(t),
		CheckDestroy:             testAccSqlUserDestroyProducer(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					testGoogleSqlProvisionScript_mysql, instance, scriptName, script),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr("google_sql_provision_script.script", "database"),
					resource.TestCheckResourceAttr("google_sql_provision_script.script", "deletion_policy", "ABANDON"),
				),
			},
			{
				Config: fmt.Sprintf(
					testGoogleSqlProvisionScript_mysql, instance, scriptName, "CREATE USER"),
				ExpectError: regexp.MustCompile(`.*`),
			},
		},
	})
}

func TestAccSqlProvisionScriptPostgres(t *testing.T) {
	t.Parallel()

	instance := fmt.Sprintf("tf-test-%d", acctest.RandInt(t))
	database := fmt.Sprintf("tf-test-%d", acctest.RandInt(t))
	scriptName := fmt.Sprintf("tf-test-%d", acctest.RandInt(t))
	script := "CREATE TABLE IF NOT EXISTS table1 ( col VARCHAR(16) NOT NULL ); CREATE EXTENSION IF NOT EXISTS vector;"
	acctest.VcrTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.AccTestPreCheck(t) },
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories(t),
		CheckDestroy:             testAccSqlUserDestroyProducer(t),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					testGoogleSqlProvisionScript_postgres, instance, database, scriptName, script),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("google_sql_provision_script.script", "database", database),
					resource.TestCheckResourceAttr("google_sql_provision_script.script", "deletion_policy", "ABANDON"),
				),
			},
			{
				Config: fmt.Sprintf(
					testGoogleSqlProvisionScript_postgres, instance, database, scriptName, "CREATE TABLE"),
				ExpectError: regexp.MustCompile(`.*`),
			},
		},
	})
}

var testGoogleSqlProvisionScript_mysql = `
resource "google_sql_database_instance" "instance" {
  name                = "%s"
  region              = "us-central1"
  database_version    = "MYSQL_8_0"
  deletion_protection = false
  settings {
    tier            = "db-f1-micro"
    #data_api_access = "ALLOW_DATA_API"
    database_flags {
      name  = "cloudsql_iam_authentication"
      value = "on"
    }
  }
}

resource "google_sql_user" "iam_user" {
  name     = "admin@hashicorptest.com"
  instance = google_sql_database_instance.instance.name
  type     = "CLOUD_IAM_USER"
}

resource "google_sql_provision_script" "script" {
  name    = "%s"
  script  = "%s"
  instance = google_sql_database_instance.instance.name
  depends_on = [google_sql_user.iam_user]
}
`

var testGoogleSqlProvisionScript_postgres = `
resource "google_sql_database_instance" "instance" {
  name                = "%s"
  region              = "us-central1"
  database_version    = "POSTGRES_17"
  deletion_protection = false
  settings {
    tier            = "db-f1-micro"
    #data_api_access = "ALLOW_DATA_API"
    database_flags {
      name  = "cloudsql.iam_authentication"
      value = "on"
    }
  }
}

resource "google_sql_user" "iam_user" {
  name     = "admin@hashicorptest.com"
  instance = google_sql_database_instance.instance.name
  type     = "CLOUD_IAM_USER"
}

resource "google_sql_database" "database" {
  name     = "%s"
  instance = google_sql_database_instance.instance.name
}

resource "google_sql_provision_script" "script" {
  name    = "%s"
  script  = "%s"
  database = google_sql_database.database.name
  instance = google_sql_database_instance.instance.name
  depends_on = [google_sql_user.iam_user]
}
`
