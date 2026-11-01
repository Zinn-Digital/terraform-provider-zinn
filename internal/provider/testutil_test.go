package provider

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

type terraformState = terraform.State

func regexpMust(s string) *regexp.Regexp { return regexp.MustCompile(regexp.QuoteMeta(s)) }
