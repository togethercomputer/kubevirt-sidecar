package schema

import (
	"encoding/xml"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

// FilesystemDriver extends the kubevirt FilesystemDriver
type FilesystemDriver struct {
	api.FilesystemDriver
	CacheSize string `xml:"cache-size,attr,omitempty"`
}

// Filesystem extends kubevirt's Filesystem to use our extended FilesystemDriver
type Filesystem struct {
	api.Filesystem
	Driver FilesystemDriver `xml:"driver,omitempty"`
	Target FilesystemTarget `xml:"target,omitempty"`
}

// DomainSpec extends kubevirt's DomainSpec to use our extended Filesystem
type DomainSpec struct {
	api.DomainSpec
	XMLName xml.Name     `xml:"domain"`
	Devices DevicesSpec  `xml:"devices"`
	QEMUCmd *Commandline `xml:"qemu:commandline,omitempty"`
}

type DevicesSpec struct {
	api.Devices
	Filesystems []Filesystem `xml:"filesystem"`
}

// Add these types
type Commandline struct {
	QEMUArg []Arg `xml:"qemu:arg"`
}

type Arg struct {
	Value string `xml:"value,attr"`
}

type FilesystemTarget struct {
	Dir string `xml:"dir,attr,omitempty"`
}
