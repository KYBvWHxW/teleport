################################################################################
# Task definition
################################################################################

locals {
  auto_update_proxy_addr = try(var.teleport_config.teleport.proxy_server, "")
  auto_update_use_v1     = var.auto_update_version_server != null
  auto_update_version = (
    length(data.http.auto_update) == 1
    ? (
      local.auto_update_use_v1
      ? data.http.auto_update[0].response_body
      : jsondecode(data.http.auto_update[0].response_body).auto_update.agent_version
    )
    : null
  )
  teleport_version = trimprefix(
    trimspace(
      coalesce(
        local.auto_update_version,
        var.teleport_version
      ),
    ),
    "v"
  )
}

resource "aws_ecs_task_definition" "teleport_agent" {
  count = var.create ? 1 : 0

  cpu                      = var.ecs_task_cpu
  execution_role_arn       = one(aws_iam_role.ecs_execution[*].arn)
  family                   = var.ecs_task_name
  memory                   = var.ecs_task_memory
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  tags                     = var.apply_aws_tags
  task_role_arn            = one(aws_iam_role.ecs_task[*].arn)

  container_definitions = jsonencode([
    {
      name       = "teleport"
      image      = "${var.teleport_container_image}:${local.teleport_version}"
      entryPoint = ["/usr/bin/dumb-init"]
      environment = [
        for name in sort(keys(var.environment_vars)) : {
          name  = name
          value = var.environment_vars[name]
        }
      ]
      command = [
        # rewrite SIGTERM (15) to SIGQUIT (3) so ECS stop signal triggers graceful Teleport shutdown
        "--rewrite",
        "15:3",
        "--",
        "teleport",
        "start",
        "--config-string",
        base64encode(yamlencode(var.teleport_config)),
      ]
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = one(aws_cloudwatch_log_group.this[*].name)
          "awslogs-region"        = one(aws_cloudwatch_log_group.this[*].region)
          "awslogs-stream-prefix" = "${var.ecs_cluster_name}-${var.ecs_service_name}"
        }
      }
    }
  ])

  lifecycle {
    precondition {
      condition     = !var.auto_update_enabled || local.auto_update_use_v1 || local.auto_update_proxy_addr != ""
      error_message = "Automatic updates v2 require teleport.proxy_server in teleport_config."
    }
  }
}
