data "util_cue" "config" {
  path = "${path.module}/config.cue"
  tags = {
    env      = "production"
    replicas = "3"
  }
}

output "server_port" {
  value = data.util_cue.config.result.server.port
}
