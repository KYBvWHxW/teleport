################################################################################
# Task definition
################################################################################

resource "aws_ecs_task_definition" "teleport_agent" {
  count = var.create ? 1 : 0

  family                   = var.ecs_task_name
  requires_compatibilities = ["FARGATE"]
  cpu                      = var.ecs_task_cpu
  memory                   = var.ecs_task_memory
  network_mode             = "awsvpc"
  task_role_arn            = one(aws_iam_role.ecs_task[*].arn)
  execution_role_arn       = one(aws_iam_role.ecs_execution[*].arn)
  tags                     = var.apply_aws_tags

  container_definitions = jsonencode([
    {
      name       = "teleport"
      image      = "${var.teleport_container_image}:${var.teleport_version}"
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
}
