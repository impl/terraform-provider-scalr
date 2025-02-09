package scalr

import (
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccWebhookDataSource_basic(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	rInt := rand.Intn(100)

	cutRInt := strconv.Itoa(rInt)[:len(strconv.Itoa(rInt))-1]

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccWebhookNeitherNameNorIdSetConfig(),
				ExpectError: regexp.MustCompile("\"id\": one of `id,name` must be specified"),
				PlanOnly:    true,
			},
			{
				Config:      testAccWebhookBothNameAndIdSetConfig(),
				ExpectError: regexp.MustCompile("\"name\": conflicts with id"),
				PlanOnly:    true,
			},
			{
				Config: testAccWebhookDataSourceConfig(rInt),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.scalr_webhook.test", "name", fmt.Sprintf("webhook-test-%d", rInt)),
					resource.TestCheckResourceAttr(
						"data.scalr_webhook.test", "enabled", "false"),
					resource.TestCheckResourceAttrSet(
						"data.scalr_webhook.test", "endpoint_id"),
					resource.TestCheckResourceAttrSet(
						"data.scalr_webhook.test", "workspace_id"),
				),
			},
			{
				Config: testAccWebhookDataSourceAccessByNameConfig(rInt),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.scalr_webhook.test", "name", fmt.Sprintf("webhook-test-%d", rInt)),
					resource.TestCheckResourceAttr(
						"data.scalr_webhook.test", "enabled", "false"),
					resource.TestCheckResourceAttrSet(
						"data.scalr_webhook.test", "endpoint_id"),
					resource.TestCheckResourceAttrSet(
						"data.scalr_webhook.test", "workspace_id"),
				),
			},
			{
				Config:      testAccWebhookDataSourceNotFoundAlmostTheSameNameConfig(rInt, cutRInt),
				ExpectError: regexp.MustCompile(fmt.Sprintf("Webhook with name 'test webhook-%s' not found", cutRInt)),
				PlanOnly:    true,
			},
			{
				Config:      testAccWebhookDataSourceNotFoundByNameConfig(),
				ExpectError: regexp.MustCompile("Webhook with name 'webhook-foo-bar-baz' not found or user unauthorized"),
				PlanOnly:    true,
			},
		},
	})
}

func testAccWebhookDataSourceConfig(rInt int) string {
	return fmt.Sprintf(`
resource scalr_environment test {
  name       = "test-env-%[1]d"
  account_id = "%s"
}

resource scalr_workspace test {
  name           = "test-ws-%[1]d"
  environment_id = scalr_environment.test.id
}

resource scalr_endpoint test {
  name           = "test endpoint-%[1]d"
  timeout        = 15
  max_attempts   = 3
  url            = "https://example.com/webhook"
  environment_id = scalr_environment.test.id
}

resource scalr_webhook test {
  enabled      = false
  name         = "webhook-test-%[1]d"
  events       = ["run:completed", "run:errored"]
  endpoint_id  = scalr_endpoint.test.id
  workspace_id = scalr_workspace.test.id
}

data scalr_webhook test {
  id = scalr_webhook.test.id
}`, rInt, defaultAccount)
}

func testAccWebhookDataSourceAccessByNameConfig(rInt int) string {
	return fmt.Sprintf(`
resource scalr_environment test {
  name       = "test-env-%[1]d"
  account_id = "%s"
}

resource scalr_workspace test {
  name           = "test-ws-%[1]d"
  environment_id = scalr_environment.test.id
}

resource scalr_endpoint test {
  name           = "test endpoint-%[1]d"
  timeout        = 15
  max_attempts   = 3
  url            = "https://example.com/webhook"
  environment_id = scalr_environment.test.id
}

resource scalr_webhook test {
  enabled      = false
  name         = "webhook-test-%[1]d"
  events       = ["run:completed", "run:errored"]
  endpoint_id  = scalr_endpoint.test.id
  workspace_id = scalr_workspace.test.id
}

data scalr_webhook test {
  name       = scalr_webhook.test.name
  account_id = scalr_environment.test.account_id
}`, rInt, defaultAccount)
}

func testAccWebhookDataSourceNotFoundByNameConfig() string {
	return `
data scalr_webhook test {
  name       = "webhook-foo-bar-baz"
  account_id = "foobar"
}`
}

func testAccWebhookNeitherNameNorIdSetConfig() string {
	return `data scalr_webhook test {}`
}

func testAccWebhookBothNameAndIdSetConfig() string {
	return `data scalr_webhook test {
		id = "foo"
		name = "bar"
	}`
}

func testAccWebhookDataSourceNotFoundAlmostTheSameNameConfig(rInt int, cutRInt string) string {
	return fmt.Sprintf(`
resource scalr_environment test {
  name       = "test-env-%[1]d"
  account_id = "%s"
}

resource scalr_workspace test {
  name           = "test-ws-%[1]d"
  environment_id = scalr_environment.test.id
}

resource scalr_endpoint test {
  name           = "test endpoint-%[1]d"
  timeout        = 15
  max_attempts   = 3
  url            = "https://example.com/webhook"
  environment_id = scalr_environment.test.id
}

resource scalr_webhook test {
  enabled      = false
  name         = "test webhook-%[1]d"
  events       = ["run:completed", "run:errored"]
  endpoint_id  = scalr_endpoint.test.id
}

data scalr_webhook test {
  name       = "test webhook-%[3]s"
  account_id = scalr_environment.test.account_id
}`, rInt, defaultAccount, cutRInt)
}
