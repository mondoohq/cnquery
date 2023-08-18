module "org-policy_bucket_policy_only" {
  source  = "terraform-google-modules/org-policy/google//modules/bucket_policy_only"
  version = "5.2.0"
  # insert the 1 required variable here
}
