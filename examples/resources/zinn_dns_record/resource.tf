data "zinn_domains" "all" {}

resource "zinn_dns_record" "www" {
  domain_id = one([for d in data.zinn_domains.all.domains : d.id if d.fqdn == "example.com"])
  name      = "www"
  type      = "CNAME"
  value     = "example.com."
  ttl       = 3600
}
