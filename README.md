# terraform-provider-zinn

The Zinn Digital Terraform provider — `zinn-digital/zinn` on the Terraform Registry. It is built on the
public API and its request/response models are **generated from `openapi/openapi.yaml`**
(`scripts/terraform-provider-genmodels.py` in the monorepo, CLAUDE.md §2.5, §2.32). The public repository
[`Zinn-Digital/terraform-provider-zinn`](https://github.com/Zinn-Digital/terraform-provider-zinn) is a
generated copy of this directory — never edit it there (§2.56).

```hcl
terraform {
  required_providers {
    zinn = { source = "zinn-digital/zinn" }
  }
}

provider "zinn" {} # api_key falls back to ZINN_API_KEY
```

| Kind | Name | Needs a key |
|---|---|---|
| data source | `zinn_regions` | no |
| data source | `zinn_plans` | no |
| data source | `zinn_sites` | yes |
| data source | `zinn_domains` | yes |
| resource | `zinn_dns_record` | yes |

## Develop (in the monorepo)

```bash
cd terraform-provider
python ../scripts/terraform-provider-genmodels.py   # regenerate models after a spec change (LAST, §2.32)
TF_ACC_TERRAFORM_PATH=$(command -v terraform) go test ./...
go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@v0.25.0 generate --provider-name zinn --rendered-provider-name "Zinn Digital"
```

Releases are cut by `scripts/sdk-release.py terraform` from the box (`docs/1145-sdk-and-provider-publishing.md`).
