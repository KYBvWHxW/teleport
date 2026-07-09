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
  description = "Name of the ECS cluster."
  default     = "teleport"
  type        = string
}

variable "ecs_service_name" {
  description = "Name of the ECS service."
  default     = "teleport-service"
  type        = string
}

variable "ecs_task_cloudwatch_log_group_name" {
  default     = "ecs-teleport"
  description = "Name for the ECS task CloudWatch log group."
  type        = string
}

variable "ecs_task_cloudwatch_log_group_region" {
  default     = ""
  description = "AWS region for the ECS task CloudWatch log group. Defaults to the AWS provider region."
  type        = string
  nullable    = true
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

variable "ecs_task_memory" {
  description = "Amount (in MiB) of memory used by the ECS task."
  default     = "4096"
  type        = string
}

variable "ecs_task_name" {
  description = "Name of the ECS task."
  default     = "teleport-agent"
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
