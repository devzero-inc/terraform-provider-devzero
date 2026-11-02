# A cluster of the provider's team
terraform import devzero_cluster.cluster <cluster_id>

# A cluster of another team, such as one a devzero_team started
terraform import devzero_cluster.cluster <team_id>/<cluster_id>
