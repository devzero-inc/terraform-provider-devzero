# Look up a cluster ID by name using the provider's default team ID
data "devzero_get_cluster_id_by_name" "example" {
  name = "my-cluster"
}

output "cluster_id" {
  value = data.devzero_get_cluster_id_by_name.example.cluster_id
}

# Look up a cluster ID with optional filters
data "devzero_get_cluster_id_by_name" "filtered" {
  name           = "my-cluster"
  region         = "us-east-1"   # optional: filter by region
  cloud_provider = "AWS"         # optional: filter by cloud provider (AWS | GCP | AKS | OCI)
  liveness       = "PREFER_LIVE" # optional: IGNORE | PREFER_LIVE | REQUIRE_LIVE
}

output "filtered_cluster_id" {
  value = data.devzero_get_cluster_id_by_name.filtered.cluster_id
}
