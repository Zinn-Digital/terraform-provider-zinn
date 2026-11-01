data "zinn_domains" "all" {}

locals {
  example_com = one([for d in data.zinn_domains.all.domains : d if d.fqdn == "example.com"])
}
