data "zinn_regions" "all" {}

output "region_slugs" {
  value = [for r in data.zinn_regions.all.regions : r.slug]
}
