// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package acc

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccSecurityProject(t *testing.T) {
	resId := "my_project"
	resourceName := fmt.Sprintf("ec_security_project.%s", resId)
	randomName := prefix + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	alias := "alias-for-acc-test-project"
	newName := prefix + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	region := getRegion()
	if !strings.HasPrefix("aws-", region) {
		region = fmt.Sprintf("aws-%s", region)
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactory,
		CheckDestroy:             testAccSecurityProjectDestroy,
		Steps: []resource.TestStep{
			{
				// Create a basic project.
				Config: testAccBasicSecurityProject(resId, randomName, region),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", randomName),
					resource.TestCheckResourceAttrSet(resourceName, "alias"),
					resource.TestCheckResourceAttrSet(resourceName, "endpoints.elasticsearch"),
					resource.TestCheckResourceAttrSet(resourceName, "endpoints.kibana"),
					resource.TestCheckResourceAttrSet(resourceName, "credentials.username"),
					resource.TestCheckResourceAttrSet(resourceName, "credentials.password"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_id"),
				),
			},
			{
				// Explicitly set the alias.
				Config: testAccSecurityProjectWithAlias(resId, randomName, region, alias),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", randomName),
					resource.TestCheckResourceAttr(resourceName, "alias", alias),
					resource.TestCheckResourceAttrSet(resourceName, "endpoints.elasticsearch"),
					resource.TestCheckResourceAttrSet(resourceName, "endpoints.kibana"),
					resource.TestCheckResourceAttrSet(resourceName, "credentials.username"),
					resource.TestCheckResourceAttrSet(resourceName, "credentials.password"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_id"),
				),
			},
			{
				// Change the name.
				Config: testAccSecurityProjectWithAlias(resId, newName, region, alias),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", newName),
					resource.TestCheckResourceAttr(resourceName, "alias", alias),
					resource.TestCheckResourceAttrSet(resourceName, "endpoints.elasticsearch"),
					resource.TestCheckResourceAttrSet(resourceName, "endpoints.kibana"),
					resource.TestCheckResourceAttrSet(resourceName, "credentials.username"),
					resource.TestCheckResourceAttrSet(resourceName, "credentials.password"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_id"),
				),
			},
		},
	})
}

func testAccBasicSecurityProject(id string, name string, region string) string {
	return fmt.Sprintf(`
resource ec_security_project "%s" {
	name = "%s"
	region_id = "%s"
	admin_features_package = "standard"
	product_types = [{
	  product_line = "security"
	  product_tier = "essentials"
	}, {
	  product_line = "cloud"
	  product_tier = "essentials"
	}, {
	  product_line = "endpoint"
	  product_tier = "essentials"
	}]
}
`, id, name, region)
}

func testAccSecurityProjectWithAlias(id string, name string, region string, alias string) string {
	return fmt.Sprintf(`
resource ec_security_project "%s" {
	name = "%s"
	region_id = "%s"
	alias = "%s"
	admin_features_package = "standard"
	product_types = [{
	  product_line = "security"
	  product_tier = "essentials"
	}, {
	  product_line = "cloud"
	  product_tier = "essentials"
	}, {
	  product_line = "endpoint"
	  product_tier = "essentials"
	}]
}
`, id, name, region, alias)
}

func testAccSecurityProjectDestroy(s *terraform.State) error {
	// retrieve the connection established in Provider configuration
	client, err := newServerlessAPI()
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "ec_security_project" {
			continue
		}

		res, err := client.GetSecurityProjectWithResponse(context.Background(), rs.Primary.ID)

		// The resource will only exist if it can be obtained via the API and
		// the metadata status is not set to hidden. Currently ESS clients
		// cannot delete a deployment, so even when it's been shut down it will
		// show up on the GET call.
		if err == nil && res.JSON200 != nil {
			res, err := client.DeleteSecurityProjectWithResponse(context.Background(), rs.Primary.ID, nil)
			if err != nil && res.StatusCode() == 200 {
				return nil
			}

			return fmt.Errorf("security project [%s] still exists", rs.Primary.ID)
		}
	}

	return nil
}
