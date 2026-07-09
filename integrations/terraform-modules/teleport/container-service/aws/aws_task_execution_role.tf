################################################################################
# Execution role
################################################################################

resource "aws_iam_role" "ecs_execution" {
  count = var.create ? 1 : 0

  assume_role_policy = one(data.aws_iam_policy_document.ecs_execution_trust[*].json)
  description        = "Execution role used by the Teleport ECS agent task."
  name_prefix        = "${var.ecs_cluster_name}-exec"
  tags               = var.apply_aws_tags
}

data "aws_iam_policy_document" "ecs_execution_trust" {
  count = var.create ? 1 : 0

  statement {
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["ecs-tasks.amazonaws.com"]
    }

    actions = ["sts:AssumeRole"]

    condition {
      test     = "StringEquals"
      variable = "aws:SourceAccount"
      values   = [one(data.aws_caller_identity.this[*].account_id)]
    }

    condition {
      test     = "ArnLike"
      variable = "aws:SourceArn"
      values = [
        format(
          "arn:%s:ecs:%s:%s:*",
          one(data.aws_partition.this[*].partition),
          one(data.aws_region.this[*].name),
          one(data.aws_caller_identity.this[*].account_id),
        ),
      ]
    }
  }
}

resource "aws_iam_role_policy" "ecs_execution" {
  count = var.create ? 1 : 0

  name_prefix = one(aws_iam_role.ecs_execution[*].name)
  role        = one(aws_iam_role.ecs_execution[*].id)
  policy      = one(data.aws_iam_policy_document.ecs_execution[*].json)
}

data "aws_iam_policy_document" "ecs_execution" {
  count = var.create ? 1 : 0

  statement {
    effect = "Allow"
    actions = [
      "logs:CreateLogStream",
      "logs:PutLogEvents",
    ]
    resources = [
      one(aws_cloudwatch_log_group.this[*].arn),
      "${one(aws_cloudwatch_log_group.this[*].arn)}:*",
    ]
  }
}
