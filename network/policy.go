// Copyright Microsoft Corp.
// All rights reserved.

package network

import (
	"encoding/json"
)

type CNIPolicyType string

const (
	NetworkPolicy CNIPolicyType = "NetworkPolicy"
	EndpointPolicy CNIPolicyType = "EndpointPolicy"
	OutBoundNatPolicy CNIPolicyType = "OutBoundNatPolicy"
)

type Policy struct {
	Type CNIPolicyType
	Data json.RawMessage
}

// HNS port mapping policy is like so:
//
// "Policies": [{
//     "Type": "NAT",
//     "ExternalPort": 8080,
//     "InternalPort": 80,
//     "Protocol": "tcp"
// }]
type NatPolicy struct {
	Type         string
	ExternalPort int
	InternalPort int
	Protocol     string
}

