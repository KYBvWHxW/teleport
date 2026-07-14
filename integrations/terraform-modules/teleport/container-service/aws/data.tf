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

data "http" "auto_update" {
  count = var.create && var.auto_update_enabled ? 1 : 0

  url = (
    local.auto_update_use_v1
    ? format(
      "%s/%s/version",
      trimsuffix(var.auto_update_version_server, "/"),
      var.auto_update_release_channel,
    )
    : format(
      "https://%s/webapi/find?group=%s",
      local.auto_update_proxy_addr,
      urlencode(coalesce(var.auto_update_group, "default")),
    )
  )

  lifecycle {
    precondition {
      condition     = !var.auto_update_enabled || local.auto_update_use_v1 || local.auto_update_proxy_addr != ""
      error_message = "Automatic updates v2 require teleport.proxy_server in teleport_config."
    }

    postcondition {
      condition = self.status_code == 200 && can(
        regex(
          "^v?[0-9]+\\.[0-9]+\\.[0-9]+.*",
          trimspace(
            local.auto_update_use_v1
            ? self.response_body
            : jsondecode(self.response_body).auto_update.agent_version
          ),
        )
      )
      error_message = <<EOF
Automatic updates endpoint must return HTTP 200 and a valid Teleport version.
Ensure the cluster supports automatic updates and the configured update group or release channel exists.
EOF
    }
  }
}
