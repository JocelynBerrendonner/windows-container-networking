// Copyright Microsoft Corp.
// All rights reserved.

package cni

import (
	"encoding/json"
	"fmt"
	"net"

	network "github.com/Microsoft/windowscontainernetworking/network"
	cniSkel "github.com/containernetworking/cni/pkg/skel"
	cniTypes "github.com/containernetworking/cni/pkg/types"
	cniTypes020 "github.com/containernetworking/cni/pkg/types/020"
)

const (
	// Supported CNI versions.
	Version = "0.2.0"

	// CNI commands.
	CmdAdd = "ADD"
	CmdDel = "DEL"

	Internal = "internal"
)

// NetworkConfig represents the Windows CNI plugin's network configuration.
// Defined as per https://github.com/containernetworking/cni/blob/master/SPEC.md
type NetworkConfig struct {
	CniVersion string `json:"cniVersion"`
	Name       string `json:"name"` // Name is the Network Name. We would also use this as the Type of HNS Network
	Type       string `json:"type"` // As per SPEC, Type is Name of the Binary
	Ipam       struct {
		Type          string           `json:"type"`
		Environment   string           `json:"environment,omitempty"`
		AddrSpace     string           `json:"addressSpace,omitempty"`
		Subnet        string           `json:"subnet,omitempty"`
		Address       string           `json:"ipAddress,omitempty"`
		QueryInterval string           `json:"queryInterval,omitempty"`
		Routes        []cniTypes.Route `json:"routes,omitempty"`
	}
	DNS            cniTypes.DNS `json:"dns"`
	AdditionalArgs []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
}

type Interface struct {
	Name       string           `json:"name"`
	MacAddress net.HardwareAddr `json:"mac"`
	Sandbox    string           `json:"sandbox"`
}

type IP struct {
	Version        string         `json:"version"` // 4 or 6
	Address        cniTypes.IPNet `json:"address"`
	Gateway        net.IP         `json:"gateway"`
	InterfaceIndex int            `json:"interface"` // Numeric index into 'interfaces' list
}

type Result struct {
	CniVersion string           `json:"cniVersion"`
	Interfaces []Interface      `json:"interfaces"`
	IP         []IP             `json:"ip"`
	DNS        cniTypes.DNS     `json:"dns"`
	Routes     []cniTypes.Route `json:"routes,omitempty"`
}

func (r *Result) Print() {
	fmt.Printf(r.String())
}

func (r *Result) String() string {
	json, _ := json.Marshal(r)
	return string(json)
}

// CNI contract.
type PluginApi interface {
	Add(args *cniSkel.CmdArgs) error
	Delete(args *cniSkel.CmdArgs) error
}

// CallPlugin calls the given CNI plugin through the internal interface.
func CallPlugin(plugin PluginApi, cmd string, args *cniSkel.CmdArgs, config *NetworkConfig) (*cniTypes.Result, error) {
	var err error

	savedType := config.Ipam.Type
	config.Ipam.Type = Internal
	args.StdinData = config.Serialize()

	// Call the plugin's internal interface.
	if cmd == CmdAdd {
		err = plugin.Add(args)
	} else {
		err = plugin.Delete(args)
	}

	config.Ipam.Type = savedType

	if err != nil {
		return nil, err
	}

	// Read back the result.
	var result cniTypes.Result
	err = json.Unmarshal(args.StdinData, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// ParseNetworkConfig unmarshals network configuration from bytes.
func ParseNetworkConfig(b []byte) (*NetworkConfig, error) {
	config := NetworkConfig{}

	err := json.Unmarshal(b, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// Serialize marshals a network configuration to bytes.
func (config *NetworkConfig) Serialize() []byte {
	bytes, _ := json.Marshal(config)
	return bytes
}

// Get NetworkInfo from the NetworkConfig
func (config *NetworkConfig) GetNetworkInfo() *network.NetworkInfo {
	var subnet = network.SubnetInfo{}
	if config.Ipam.Subnet != "" {
		ip, s, _ := net.ParseCIDR(config.Ipam.Subnet)
		gatewayIP := ip.To4()
		gatewayIP[3]++
		subnet = network.SubnetInfo{
			AddressPrefix:  *s,
			GatewayAddress: gatewayIP,
		}
	}
	return &network.NetworkInfo{
		ID:            config.Name,
		Name:          config.Name,
		Type:          network.NetworkType(config.Name),
		Subnets:       []network.SubnetInfo{subnet},
		InterfaceName: "",
		DNS: network.DNSInfo{
			Servers: config.DNS.Nameservers,
			Suffix:  config.DNS.Domain,
		},
	}
}

// Get NetworkInfo from the NetworkConfig
func (config *NetworkConfig) GetEndpointInfo(endpointId string, containerID string) *network.EndpointInfo {
	return &network.EndpointInfo{
		Name:        endpointId,
		ContainerID: containerID,
	}
}

//GetResult
func GetResult(network *network.NetworkInfo, endpoint *network.EndpointInfo) Result {
	var iFace = GetInterface(endpoint)
	var ip = GetIP(network, endpoint)
	return Result{
		CniVersion: Version,
		Interfaces: []Interface{iFace},
		IP:         []IP{ip},
	}
}

func GetResult020(network *network.NetworkInfo, endpoint *network.EndpointInfo) cniTypes020.Result {
	var ip = GetIP(network, endpoint)
	var ip4 = &cniTypes020.IPConfig{
		IP:      net.IPNet(ip.Address),
		Gateway: ip.Gateway,
	}
	return cniTypes020.Result{
		IP4: ip4,
	}
}

// GetIP
func GetIP(network *network.NetworkInfo, endpoint *network.EndpointInfo) IP {
	address := network.Subnets[0].AddressPrefix
	address.IP = endpoint.IPAddress
	return IP{
		Version:        "4",
		Address:        cniTypes.IPNet(address),
		Gateway:        endpoint.Gateway,
		InterfaceIndex: 0,
	}
}

// GetInterface
func GetInterface(endpoint *network.EndpointInfo) Interface {
	return Interface{
		Name:       endpoint.Name,
		MacAddress: endpoint.MacAddress,
		Sandbox:    "",
	}
}