variable "image_id" {
  type        = string
  description = "The id of the machine image (AMI) to use for the server."

  validation {
    condition     = length(var.image_id) > 4 && substr(var.image_id, 0, 4) == "ami-"
    error_message = "The image_id value must be a valid AMI id, starting with \"ami-\"."
  }
}

variable "availability_zone_names" {
  type    = list(string)
  default = ["us-west-1a"]
}

output "instance_ip_addr" {
  value       = aws_instance.example.private_ip
  description = "The private IP address of the main server instance."
}