resource "teleport_foo" "test" {
  version = "v1"
  scope = "/example/basic"
  metadata = {
    name = "test"
  }
  spec = {
    value = "value1"
  }
}
