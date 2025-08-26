variable "GO_VERSION" {
  default = null
}
variable "DOCS_FORMATS" {
  default = "md,yaml"
}

target "_common" {
  args = {
    GO_VERSION = GO_VERSION
    BUILDKIT_CONTEXT_KEEP_GIT_DIR = 1
  }
}

group "default" {
  targets = ["validate"]
}

group "validate" {
  targets = [
    "validate-docs",
    "validate-tests"
  ]
}

target "validate-docs" {
  inherits = ["_common"]
  args = {
    DOCS_FORMATS = DOCS_FORMATS
  }
  target = "docs-validate"
  output = ["type=cacheonly"]
}

target "update-docs" {
  inherits = ["_common"]
  args = {
    DOCS_FORMATS = DOCS_FORMATS
  }
  target = "docs-update"
  output = ["./docs/reference"]
}

target "validate-tests" {
  inherits = ["_common"]
  target = "test"
  output = ["type=cacheonly"]
}
