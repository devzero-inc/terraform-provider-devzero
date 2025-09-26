# Copyright (c) HashiCorp, Inc.

terraform {
  required_providers {
    devzero = {
      source = "devzero-inc/devzero"
    }
  }
}

provider "devzero" {
  url     = "https://dakr.devzero.dev"
  team_id = "team-82122a716f0147ffb274d8f2bf8b56b3"
  token   = "dzu-wy6ZEc-IJhFjekK3j6Gri1JM5gegBCeEMMZwURZrCZk="
}

resource "devzero_cluster" "cluster" {
  name = "terraform-example"
}

resource "devzero_workload_policy" "workload_policy" {
  name            = "terraform-example"
  action_triggers = ["on_schedule", "on_detection"]
}

resource "devzero_workload_policy_target" "workload_policy_target" {
  name       = "terraform-example"
  policy_id  = devzero_workload_policy.workload_policy.id
  priority   = 1
  enabled    = true
  cluster_ids = [devzero_cluster.cluster.id]
}