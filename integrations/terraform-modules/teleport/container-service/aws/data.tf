data "aws_caller_identity" "this" {
  count = var.create ? 1 : 0
}

data "aws_region" "this" {
  count = var.create ? 1 : 0
}

data "aws_partition" "this" {
  count = var.create ? 1 : 0
}

data "aws_subnet" "teleport_agent" {
  count = var.create ? length(var.ecs_service_subnets) : 0

  id = var.ecs_service_subnets[count.index]
}
