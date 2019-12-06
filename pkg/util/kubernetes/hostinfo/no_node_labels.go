// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

// +build !kubeapiserver

package hostinfo

// getNodeLabels returns node labels for this host
func getNodeLabels() (map[string]string, error) {
	return nil, nil
}

// getNodeClusterNameLabel returns clustername by fetching a node label
func getNodeClusterNameLabel() (string, error) {
	return "", nil
}