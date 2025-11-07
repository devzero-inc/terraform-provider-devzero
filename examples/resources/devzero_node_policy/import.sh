#!/bin/bash

# Import an existing node policy by its ID
terraform import devzero_node_policy.example "policy-id-here"

# Example with actual ID format
terraform import devzero_node_policy.aws_basic "257a5739-b716-42c7-9bc4-1823277f3e5f"
