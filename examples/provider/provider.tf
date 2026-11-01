terraform {
  required_providers {
    zinn = {
      source = "zinn-digital/zinn"
    }
  }
}

# api_key falls back to ZINN_API_KEY. Catalogue data sources work without one.
provider "zinn" {}
