################################################################################
# Required variables
################################################################################

variable "ecs_service_subnets" {
  description = <<EOF
Subnet IDs where the Teleport agent will be deployed.
If var.assign_public_ip is true, then all of these subnets must be public subnets (route to an internet gateway).
If var.assign_public_ip is false, then all of these subnets must be private subnets (route to a NAT gateway).
EOF
  type        = list(string)
}

variable "vpc_id" {
  description = "VPC ID where the Teleport agent will be deployed."
  type        = string
}

variable "teleport_config" {
  type        = any
  description = "Teleport configuration. Write the configuration using native Terraform syntax."
}

################################################################################
# Optional variables
################################################################################

variable "apply_aws_tags" {
  default     = {}
  description = "Additional AWS tags to apply to all created AWS resources."
  type        = map(string)
}

variable "auto_update_enabled" {
  default     = false
  description = "Whether to resolve the Teleport container version from the configured automatic updates endpoint when applying this module."
  type        = bool
}

variable "auto_update_group" {
  default     = "default"
  description = "Update group to query through the v2 automatic updates endpoint."
  type        = string
}

variable "auto_update_release_channel" {
  default     = "stable/cloud"
  description = "Release channel to query through the legacy v1 automatic updates endpoint."
  type        = string
}

variable "auto_update_version_server" {
  default     = null
  description = <<EOF
Base version server URL for legacy v1 automatic updates.
Setting this selects the legacy v1 managed updates protocol instead of the default v2 protocol.
EOF
  nullable    = true
  type        = string

  validation {
    condition     = var.auto_update_version_server == null || strcontains(var.auto_update_version_server, "http")
    error_message = "When var.auto_update_version_server is set, it must be a valid http(s) URL."
  }
}

variable "assign_public_ip" {
  default     = false
  description = <<EOF
Whether to assign public IP addresses to Teleport agent ECS tasks.
If this is set to true, then var.ecs_service_subnets must be public subnets (route to an internet gateway).
Otherwise, var.ecs_service_subnets must be private subnets (route to a NAT gateway).
EOF
  type        = bool
}

variable "create" {
  default     = true
  description = "Toggle creation of all resources."
  type        = bool
}

variable "create_security_group" {
  default     = true
  description = "Whether to create a security group for the Teleport agent ECS tasks."
  type        = bool
}

variable "ecs_cluster_name" {
  default     = "teleport"
  description = "Name of the ECS cluster."
  type        = string
}

variable "ecs_service_name" {
  default     = "teleport-service"
  description = "Name of the ECS service."
  type        = string
}

variable "ecs_task_cloudwatch_log_group_name" {
  default     = "ecs-teleport"
  description = "Name for the ECS task CloudWatch log group."
  type        = string
}

variable "ecs_task_cloudwatch_log_group_region" {
  default     = null
  description = "AWS region for the ECS task CloudWatch log group. Defaults to the AWS provider region."
  nullable    = true
  type        = string
}

variable "ecs_task_cloudwatch_log_group_retention_days" {
  default     = 30
  description = "Number of days to retain logs in the ECS task CloudWatch log group."
  type        = number
}

variable "ecs_task_cloudwatch_log_group_skip_destroy" {
  default     = false
  description = <<EOF
Whether to preserve the ECS task CloudWatch log group when destroying module resources.
Set to true if you do not wish the log group (and any logs it may contain) to be deleted at destroy time, and instead just remove the log group from the Terraform state.
EOF
  type        = bool
}

variable "ecs_task_cpu" {
  default     = "2048"
  description = "Number of cpu units used by the ECS task."
  type        = string
}

variable "ecs_task_desired_count" {
  default     = 2
  description = "Desired number of Teleport ECS tasks to run."
  type        = number
}

variable "ecs_task_force_new_deployment" {
  default     = false
  description = "Set to true to force the ECS service to redeploy tasks without configuration changes."
  type        = bool
}

variable "ecs_task_memory" {
  default     = "4096"
  description = "Amount (in MiB) of memory used by the ECS task."
  type        = string
}

variable "ecs_task_name" {
  default     = "teleport-agent"
  description = "Name of the ECS task."
  type        = string
}

variable "environment_vars" {
  default     = {}
  description = "Environment variables to set on the Teleport ECS container."
  type        = map(string)
}

variable "security_group_ids" {
  default     = []
  description = "Additional security group IDs to attach to the Teleport agent ECS tasks."
  type        = list(string)
}

variable "teleport_container_image" {
  default     = "public.ecr.aws/gravitational/teleport-ent-distroless"
  description = "Container image used for Teleport ECS tasks."
  type        = string
}
