terraform {
  required_version = "1.9.2"

  required_providers {
    nomad = {
      source  = "hashicorp/nomad"
      version = "2.3.0"
    }
    consul = {
      source  = "hashicorp/consul"
      version = "2.20.0"
    }
  }
}
