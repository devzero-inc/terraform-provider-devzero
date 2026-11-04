# The first apply after an import issues a new cluster token, which the
# cluster's agents need: they stop reporting until they run with it.

# A cluster of the provider's team
terraform import devzero_cluster.cluster <cluster_id>

# A cluster of another team, such as one a devzero_team started
terraform import devzero_cluster.cluster <team_id>/<cluster_id>
