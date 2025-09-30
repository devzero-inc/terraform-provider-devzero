terraform {
  required_providers {
    devzero = {
      source = "devzero-inc/devzero"
    }
  }
}

provider "devzero" {
  team_id = "<YOUR_TEAM_ID>"
  token   = "<YOUR_TOKEN>"
}