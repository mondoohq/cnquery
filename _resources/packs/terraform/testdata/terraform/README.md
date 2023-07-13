# Terraform Example

This test file is used to test the Terraform project with HCL MQL resources as well as the state. To generate the statefile, run the following commands:

```bash
terraform init
terraform plan -out tfplan
terraform show -json tfplan > plan.json
```