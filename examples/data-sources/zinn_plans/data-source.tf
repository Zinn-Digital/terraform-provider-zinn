data "zinn_plans" "agency" {
  product_line = "agency"
}

output "agency_plan_codes" {
  value = [for p in data.zinn_plans.agency.plans : p.code]
}
