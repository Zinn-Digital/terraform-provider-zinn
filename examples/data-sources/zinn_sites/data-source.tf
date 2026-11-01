data "zinn_sites" "all" {}

output "site_domains" {
  value = [for s in data.zinn_sites.all.sites : s.primary_domain]
}
